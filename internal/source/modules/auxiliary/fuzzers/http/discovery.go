// discovery.go
package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

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

func loadWordlist(path string) {
	file, err := os.Open(path)
	if err != nil {
		fmt.Errorf("Failed to read wordlist => %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			SubmitJob(line, "WORDLIST")
		}
	}
}

func discoverPathsAutomatically(client *http.Client, baseURL string) {
	// Karena ini adalah fungsi yang men-spawn otomatis dari main(),
	// kita tandai bahwa pekerjaannya sudah selesai saat fungsi ini return.
	defer GlobalTaskTracker.Done()

	resp, err := client.Get(baseURL)
	if err != nil {
		fmt.Errorf("[ERROR] Failed to perform basic crawl => %v\n", err)
		return
	}
	defer resp.Body.Close()

	safeBodyReader := io.LimitReader(resp.Body, 5*1024*1024)

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		fmt.Errorf("[ERROR] Failed to parse base URL => %v\n", err)
		return
	}

	tokenizer := html.NewTokenizer(safeBodyReader)

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

					// OPTIMASI: Langsung lempar ke SubmitJob. 
					// Deduplikasi akan diurus oleh Central Dispatcher!
					SubmitJob(cleanPath, "HTML")
				}
			}
		}
	}
}

func extractFromJS(client *http.Client, jsURL string, parsedBase *url.URL) {
	// PERBAIKAN: Fungsi ini di-spawn oleh Worker (go extractFromJS).
	// Maka ia wajib melapor ke GlobalTaskTracker saat tugasnya selesai agar program tidak mati.
	defer GlobalTaskTracker.Done()

	jsParseSemaphore <- struct{}{}
	defer func() {
		<-jsParseSemaphore
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
			strings.HasSuffix(lowerPath, ".css") ||
			strings.HasSuffix(lowerPath, ".png") ||
			strings.HasSuffix(lowerPath, ".jpg") ||
			strings.HasSuffix(lowerPath, ".jpeg") ||
			strings.HasSuffix(lowerPath, ".ico") ||
			strings.HasSuffix(lowerPath, ".svg") ||
		    strings.HasSuffix(lowerPath, ".woff2") ||
		    strings.HasSuffix(lowerPath, ".gif") ||

		    strings.Contains(lowerPath, "text/") ||
			strings.Contains(lowerPath, "cdn-cgi/") {
			continue
		}

		// OPTIMASI: Langsung lempar temuan JS ini ke Central Dispatcher!
		// Pekerjaan HTTP Request dan pengecekan file .js rekursif akan diserahkan kembali ke pasukan Worker.
		SubmitJob(cleanPath, "JS")
	}
}
