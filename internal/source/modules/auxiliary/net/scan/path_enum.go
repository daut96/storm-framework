// GPL License.
// Copyright (c) 2026 Storm Framework
// See LICENSE file in the project root for full license information.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type DiagnosticResult struct {
	Source     string `json:"source"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
}

type CrawlJob struct {
	Path   string
	Source string
}

// Global baseline untuk mendeteksi anomali Soft 404
var soft404Size int64 = -1

// Global regex
var linkFinderEngine *regexp.Regexp
var mapMutex sync.Mutex

func main() {
	// Configuration
	targetURL := flag.String("url", "", "Base URL target")
	wordlistPath := flag.String("wordlist", "", "Path to wordlist file (opsional)")
	threads := flag.Int("threads", 1, "Number of concurrent workers")
	regex := flag.String("regex", "", "Path to custom regex file for JS parsing")
	flag.Parse()

	if *targetURL == "" {
		log.Printf("Error => The url parameter is absolutely required.")
		os.Exit(1)
	}

	if *regex != "" {
		err := initDynamicRegex(*regex)
		if err != nil {
			log.Fatalf("Error => Failed to initialize custom regex: %v\n", err)
		}
	}

	// Memastikan format URL konsisten memiliki trailing slash
	if !strings.HasSuffix(*targetURL, "/") {
		*targetURL += "/"
	}

	// Inisialisasi HTTP Client berkinerja tinggi (Connection Pooling)
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true, // Save CPU
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second, // Dinaikkan ke 5s demi kestabilan jaringan cloud target
	}

	// Kalibrasi Anomali Jaringan (Deteksi Soft 404)
	calibrateSoft404(client, *targetURL)

	// SINKRONISASI TIPE DATA: Ubah channel menjadi penampung objek CrawlJob
	jobs := make(chan CrawlJob, 5000)
	results := make(chan DiagnosticResult, 5000)
	var wg sync.WaitGroup

	// Spawning Worker Pool
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go worker(client, *targetURL, jobs, results, &wg)
	}

	doneAggregator := make(chan bool)
	
	// Sinkronisasi Output (Result Aggregator) - FIX VARIABEL UNDEFINED
	go func() {
		for res := range results {
			// Menggunakan data asli dari struct DiagnosticResult (res)
			// Format output disesuaikan agar dibaca mulus oleh regex Python baru kita
			fmt.Printf("[RESULT] [%s] Path: %s | Status: %d | Size: %d | Type: %s\n", 
				res.Source, res.Path, res.StatusCode, res.Size, res.Type,
			)
		}
		doneAggregator <- true 
	}()

	// Penentuan Mekanisme Input (Wordlist vs Otomatis)
	if *wordlistPath != "" {
		fmt.Println("[RESULT] Mode => Using Static Wordlist Input")
		loadWordlist(*wordlistPath, jobs)
	} else {
		fmt.Println("[RESULT] Mode => Empty wordlist. Enable JIT Crawling")
		discoverPathsAutomatically(client, *targetURL, jobs)
	}

	// GRACEFUL SHUTDOWN SEQUENCE
	close(jobs)          
	wg.Wait()            
	close(results)       
	<-doneAggregator     
}

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

func calibrateSoft404(client *http.Client, baseURL string) {
	randomPath := fmt.Sprintf("anomaly_test_%d.html", time.Now().UnixNano())
	resp, err := client.Get(baseURL + randomPath)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		soft404Size = int64(len(body))
		log.Printf("Warning => Soft 404 Detection Active. Baseline Size => %d bytes\n", soft404Size)
	}
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
			// Bungkus kata dari wordlist dengan label SOURCE: WORDLIST
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

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("[ERROR] Failed to parse base URL => %v\n", err)
		return
	}

	tokenizer := html.NewTokenizer(resp.Body)
	var jsWG sync.WaitGroup
	visited := make(map[string]bool)
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
					if err != nil {
						continue
					}

					if resolvedURL.Host != parsedBase.Host {
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

					if strings.HasSuffix(strings.ToLower(cleanPath), ".js") {
						jsWG.Add(1)
						go extractFromJS(client, resolvedURL.String(), parsedBase, visited, jobs, &jsWG)
					}

					mapMutex.Lock()
					if !visited[cleanPath] {
						visited[cleanPath] = true
						localNewPaths = append(localNewPaths, cleanPath)
					}
					mapMutex.Unlock()
				}
			}
		}
	}

	for _, path := range localNewPaths {
		jobs <- CrawlJob{Path: path, Source: "HTML"}
	}

	jsWG.Wait()

	mapMutex.Lock()
	totalFound := len(visited)
	mapMutex.Unlock()

	fmt.Printf("[SUCCESS] Crawl Engine finished. Total unique targets in state map => %d\n", totalFound)
}


func extractFromJS(client *http.Client, jsURL string, parsedBase *url.URL, visited map[string]bool, jobs chan<- CrawlJob, wg *sync.WaitGroup) {
	defer wg.Done()

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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Selesaikan base URL relative path berdasarkan lokasi file JS saat ini, bukan domain utama
	parsedJSURL, err := url.Parse(jsURL)
	if err != nil {
		parsedJSURL = parsedBase
	}

	matches := linkFinderEngine.FindAllSubmatch(body, -1)
	var localNewEndpoints []CrawlJob
	localNewCount := 0

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

		// Resolve path secara akurat berdasarkan standar RFC
		if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") {
			resolvedURL, err := url.Parse(rawPath)
			if err != nil {
				continue
			}
			if resolvedURL.Host != parsedBase.Host {
				continue
			}
			finalAbsoluteURL = resolvedURL
		} else {
			// Hubungkan relative path dengan sub-direktori file JS berada
			resolvedURL, err := parsedJSURL.Parse(rawPath)
			if err != nil {
				continue
			}
			if resolvedURL.Host != parsedBase.Host {
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
		
		// Filter Noise yang tidak perlu di-fuzzing
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

		mapMutex.Lock()
		if !visited[cleanPath] {
			visited[cleanPath] = true
			localNewCount++
			
			// Simpan objek pekerjaan baru ke array lokal
			jobItem := CrawlJob{Path: cleanPath, Source: "JS"}
			localNewEndpoints = append(localNewEndpoints, jobItem)

			// =========================================================================
			// RECURSIVE ENGINE TRIGGER: Jika menemukan file JS baru di dalam file JS, 
			// kejar secara rekursif sampai mentok (ujung dunia)
			// =========================================================================
			if strings.HasSuffix(lowerPath, ".js") {
				wg.Add(1)
				go extractFromJS(client, finalAbsoluteURL.String(), parsedBase, visited, jobs, wg)
			}
		}
		mapMutex.Unlock()
	}

	// Mengirim seluruh target temuan ke worker pool fuzzer utama
	for _, job := range localNewEndpoints {
		jobs <- job
	}
}

// FIX WORKER PARAMETER: Sekarang membaca channel tipe <-chan CrawlJob
func worker(client *http.Client, baseURL string, jobs <-chan CrawlJob, results chan<- DiagnosticResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
		// Bongkar isi struct CrawlJob
		cleanPath := strings.TrimPrefix(job.Path, "/")
		fullURL := baseURL + cleanPath

		req, err := http.NewRequest("HEAD", fullURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")

		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		statusCode := resp.StatusCode
		var size int64 = 0

		if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusOK {
			getReq, _ := http.NewRequest("GET", fullURL, nil)
			getResp, err := client.Do(getReq)
			if err == nil {
				body, _ := io.ReadAll(getResp.Body)
				size = int64(len(body))
				statusCode = getResp.StatusCode
				getResp.Body.Close()
			}
		}

		logType := "Valid"
		if statusCode == http.StatusNotFound {
			logType = "Not Found (404)"
		} else if statusCode == http.StatusOK && size == soft404Size {
			logType = "Soft 404 Anomaly"
		}

		// Kirim hasil diagnostic lengkap beserta metadata ASAL-USUL mesin (Source) ke aggregator
		results <- DiagnosticResult{
			Source:     job.Source, // Teruskan informasi asal-usul (HTML / JS_Regex / WORDLIST)
			Path:       cleanPath,
			StatusCode: statusCode,
			Size:       size,
			Type:       logType,
		}
	}
}
