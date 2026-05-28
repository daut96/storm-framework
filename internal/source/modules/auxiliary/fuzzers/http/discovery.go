// discovery.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/net/html"
)

func initDynamicRegex(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	rawRegex := strings.TrimSpace(string(content))
	if rawRegex == "" {
		return fmt.Errorf("regex file is empty")
	}

	compiled, err := regexp.Compile(rawRegex)
	if err != nil {
		return fmt.Errorf("invalid regex syntax: %w", err)
	}

	linkFinderEngine = compiled
	return nil
}

func loadWordlist(path string, jobs chan<- CrawlJob) {
	file, err := os.Open(path)
	if err != nil {
		log.Fatalf("Failed to read wordlist => %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			jobs <- CrawlJob{Path: line, Source: "WORDLIST"}
		}
	}
}

func discoverPathsAutomatically(client *http.Client, baseURL string, jobs chan<- CrawlJob) {
	resp, err := client.Get(baseURL)
	if err != nil {
		log.Printf("[ERROR] Failed to perform basic crawl => %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Mitigasi OOM: Baca maksimal 5MB dari base URL
	safeBodyReader := io.LimitReader(resp.Body, 5*1024*1024)

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("[ERROR] Failed to parse base URL => %v\n", err)
		return
	}

	tokenizer := html.NewTokenizer(safeBodyReader)
	var jsWG sync.WaitGroup
	var localNewPaths []string

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			break
		}

		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			token := tokenizer.Token()

			for _, attr := range token.Attr {
				if attr.Key == "href" || attr.Key == "src" {
					rawPath := strings.TrimSpace(attr.Val)
					if rawPath == "" || strings.HasPrefix(rawPath, "#") || strings.HasPrefix(rawPath, "javascript:") {
						continue
					}

					resolvedURL, err := parsedBase.Parse(rawPath)
					if err != nil || resolvedURL.Host != parsedBase.Host {
						continue
					}

					pathOnly := resolvedURL.Path
					if idx := strings.IndexAny(pathOnly, "?#"); idx != -1 {
						pathOnly = pathOnly[:idx]
					}

					cleanPath := strings.TrimPrefix(pathOnly, "/")
					if cleanPath == "" {
						continue
					}

					// Atomic Deduplication: Cek dan simpan dalam 1 instruksi mesin (Thread-Safe)
					if _, loaded := visitedMap.LoadOrStore(cleanPath, true); !loaded {
						localNewPaths = append(localNewPaths, cleanPath)

						if strings.HasSuffix(strings.ToLower(cleanPath), ".js") {
							jsWG.Add(1)
							go extractFromJS(client, resolvedURL.String(), parsedBase, jobs, &jsWG)
						}
					}
				}
			}
		}
	}

	for _, path := range localNewPaths {
		jobs <- CrawlJob{Path: path, Source: "HTML"}
	}
	jsWG.Wait()
}

func extractFromJS(client *http.Client, jsURL string, parsedBase *url.URL, jobs chan<- CrawlJob, wg *sync.WaitGroup) {
	// Meminta izin masuk ke goroutine (mengambil token dari semaphore)
	jsParseSemaphore <- struct{}{}
	defer func() {
		<-jsParseSemaphore // Mengembalikan token saat selesai
		wg.Done()
	}()

	if linkFinderEngine == nil {
		return
	}

	req, err := http.NewRequest("GET", jsURL, nil)
	if err != nil {
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return
	}
	defer resp.Body.Close()

	// Mitigasi OOM: Tarpit Defense (Hanya proses 5MB awal dari file JS)
	safeReader := io.LimitReader(resp.Body, 5*1024*1024)
	body, err := io.ReadAll(safeReader)
	if err != nil {
		return
	}

	parsedJSURL, err := url.Parse(jsURL)
	if err != nil {
		parsedJSURL = parsedBase
	}

	matches := linkFinderEngine.FindAllSubmatch(body, -1)
	var localNewEndpoints []CrawlJob

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		rawPath := strings.TrimSpace(string(match[1]))
		if rawPath == "" {
			continue
		}

		if strings.HasPrefix(rawPath, "//") {
			rawPath = parsedBase.Scheme + ":" + rawPath
		}

		var finalAbsoluteURL *url.URL

		if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") {
			resolvedURL, err := url.Parse(rawPath)
			if err != nil || resolvedURL.Host != parsedBase.Host {
				continue
			}
			finalAbsoluteURL = resolvedURL
		} else {
			resolvedURL, err := parsedJSURL.Parse(rawPath)
			if err != nil || resolvedURL.Host != parsedBase.Host {
				continue
			}
			finalAbsoluteURL = resolvedURL
		}

		pathOnly := finalAbsoluteURL.Path
		if idx := strings.IndexAny(pathOnly, "?#"); idx != -1 {
			pathOnly = pathOnly[:idx]
		}

		cleanPath := strings.TrimPrefix(pathOnly, "/")
		lowerPath := strings.ToLower(cleanPath)

		if lowerPath == "" ||
			strings.HasSuffix(lowerPath, "text/") ||
			strings.HasSuffix(lowerPath, ".css") ||
			strings.HasSuffix(lowerPath, ".png") ||
			strings.HasSuffix(lowerPath, ".jpg") ||
			strings.HasSuffix(lowerPath, ".jpeg") ||
			strings.HasSuffix(lowerPath, ".ico") ||
			strings.HasSuffix(lowerPath, ".svg") ||
			strings.Contains(lowerPath, "cdn-cgi/") {
			continue
		}

		// Atomic Deduplication
		if _, loaded := visitedMap.LoadOrStore(cleanPath, true); !loaded {
			jobItem := CrawlJob{Path: cleanPath, Source: "JS"}
			localNewEndpoints = append(localNewEndpoints, jobItem)

			if strings.HasSuffix(lowerPath, ".js") {
				wg.Add(1)
				go extractFromJS(client, finalAbsoluteURL.String(), parsedBase, jobs, wg)
			}
		}
	}

	for _, job := range localNewEndpoints {
		jobs <- job
	}
}

