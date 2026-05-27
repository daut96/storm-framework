// GPL License.
// Copyright (c) 2026 Storm Framework
// See LICENSE file in the project root for full license information.
package main

import (
	"bufio"
	"flag"
	"log"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"os"
	"strings"
	"sync"
	"time"
	"golang.org/x/net/html"
)

type DiagnosticResult struct {
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
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
		Timeout:   2 * time.Second,
	}

	// Kalibrasi Anomali Jaringan (Deteksi Soft 404)
	calibrateSoft404(client, *targetURL)

	jobs := make(chan string, 1000)
	results := make(chan DiagnosticResult, 1000)
	var wg sync.WaitGroup

	// Spawning Worker Pool
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go worker(client, *targetURL, jobs, results, &wg)
	}

	doneAggregator := make(chan bool)
	// Sinkronisasi Output (Result Aggregator)
	go func() {
		for res := range results {
			// Format output standard terstruktur agar mudah diparsing oleh regex Python
			fmt.Printf("[RESULT] PATH:%s | STATUS:%d | SIZE:%d | TYPE:%s\n", 
				res.Path, res.StatusCode, res.Size, res.Type)
		}
		// Sinyal dikirim SETELAH channel results ditutup dan semua buffer terbaca habis
		doneAggregator <- true 
	}()

	// 5. Penentuan Mekanisme Input (Wordlist vs Otomatis)
	if *wordlistPath != "" {
		fmt.Println("[RESULT] Mode => Using Static Wordlist Input")
		loadWordlist(*wordlistPath, jobs)
	} else {
		fmt.Println("[RESULT] Mode => Empty wordlist. Enable JIT Crawling")
		discoverPathsAutomatically(client, *targetURL, jobs)
	}

	// GRACEFUL SHUTDOWN SEQUENCE
	close(jobs)          // 1. Tutup input stream (memberitahu worker tidak ada job lagi)
	wg.Wait()            // 2. Tunggu semua worker selesai memproses job yang tersisa
	close(results)       // 3. Tutup output stream (memberitahu aggregator tidak ada data lagi)
	<-doneAggregator     // 4. BLOCKING: Tunggu aggregator selesai nge-print semuanya ke stdout!
}

func initDynamicRegex(filePath string) error {
	// Baca file regex eksternal yang ditunjuk oleh flag
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read file: %w", err)
	}

	// Ambil string murni dan bersihkan spasi di ujung-ujungnya (jika ada)
	rawRegex := strings.TrimSpace(string(content))
	if rawRegex == "" {
		return fmt.Errorf("regex file is empty")
	}
	
	// Compile string menjadi executable regex state-machine
	compiled, err := regexp.Compile(rawRegex)
	if err != nil {
		return fmt.Errorf("invalid regex syntax: %w", err)
	}
	
	// Masukkan ke global variable agar bisa diakses oleh fungsi crawler
	linkFinderEngine = compiled
	return nil
}

func calibrateSoft404(client *http.Client, baseURL string) {
	// Membuat string acak resolusi tinggi untuk memicu 404 murni pada server
	randomPath := fmt.Sprintf("anomaly_test_%d.html", time.Now().UnixNano())
	resp, err := client.Get(baseURL + randomPath)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		// Server mengembalikan 200 OK untuk path tidak valid (Soft 404 dideteksi!)
		body, _ := io.ReadAll(resp.Body)
		soft404Size = int64(len(body))
		log.Printf("Warning => Soft 404 Detection Active. Baseline Size => %d bytes\n", soft404Size)
	}
}

func loadWordlist(path string, jobs chan<- string) {
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
			jobs <- line
		}
	}
}

func discoverPathsAutomatically(client *http.Client, baseURL string, jobs chan<- string) {
	resp, err := client.Get(baseURL)
	if err != nil {
		log.Fatalf("Failed to perform basic crawl => %v\n", err)
		return
	}
	defer resp.Body.Close()

	// Parsing base URL untuk keperluan validasi domain & normalisasi relative path
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("Failed to parse base URL => %v\n", err)
		return
	}

	// Inisialisasi HTML tokenizer dari body response
	tokenizer := html.NewTokenizer(resp.Body)
	
	var jsWG sync.WaitGroup
	
	visited := make(map[string]bool)

	for {
		tokenType := tokenizer.Next()

		// Selesai membaca dokumen (EOF) atau terjadi error
		if tokenType == html.ErrorToken {
			break
		}

		// Kita hanya tertarik pada Start Tag (e.g., <a>, <img>) dan Self-Closing Tag (e.g., <script/>, <link/>)
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			token := tokenizer.Token()

			// Iterasi seluruh atribut di dalam tag untuk mencari href atau src
			for _, attr := range token.Attr {
				if attr.Key == "href" || attr.Key == "src" {
					rawPath := strings.TrimSpace(attr.Val)
					if rawPath == "" || strings.HasPrefix(rawPath, "#") || strings.HasPrefix(rawPath, "javascript:") {
						continue
					}

					// Normalisasi URL: Mengubah relative path menjadi absolute URL objek
					resolvedURL, err := parsedBase.Parse(rawPath)
					if err != nil {
						continue
					}

					// SECURITY FILTER: Pastikan path yang diekstrak masih satu host dengan target
					// Mencegah fuzzer melompat keluar ke external domain (e.g., google.com, github.com)
					if resolvedURL.Host != parsedBase.Host {
						continue
					}

					// Ambil path yang sudah bersih (tanpa query parameters/fragment untuk fuzzing murni)
					cleanPath := strings.TrimPrefix(resolvedURL.Path, "/")
					
					// Jika path merujuk ke root halaman utama, abaikan agar tidak looping
					if cleanPath == "" {
						continue
					}

					// js path detection
					if strings.HasSuffix(cleanPath, ".js") {
						jsWG.Add(1)
                        go extractFromJS(client, resolvedURL.String(), parsedBase, visited, jobs, &jsWG)
                    }
					mapMutex.Lock()
					
					// De-duplikasi menggunakan map state
					if !visited[cleanPath] && cleanPath != "" {
                        visited[cleanPath] = true
                        jobs <- cleanPath
                    }
					mapMutex.Unlock()
				}
			}
		}
	}
	jsWG.Wait()
	fmt.Printf("[SUCCESS] Successfully extracted via HTML Parser => %d\n", len(visited))
}

func extractFromJS(client *http.Client, jsURL string, parsedBase *url.URL, visited map[string]bool, jobs chan<- string, wg *sync.WaitGroup) {
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

	// OPTIMASI: Gunakan FindAllSubmatch untuk memproses []byte langsung tanpa alokasi string(body) raksasa
	matches := linkFinderEngine.FindAllSubmatch(body, -1)
	
	var localNewEndpoints []string
	localNewCount := 0

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		// Konversi ke string hanya untuk bagian yang match saja (Hemat Memori Heap)
		rawPath := strings.TrimSpace(string(match[1]))
		if rawPath == "" {
			continue
		}

		// Normalisasi Schemeless URL (//) menjadi URL absolut yang valid sebelum di-parse
		if strings.HasPrefix(rawPath, "//") {
			rawPath = parsedBase.Scheme + ":" + rawPath
		}

		// Evaluasi Absolute URL
		if strings.HasPrefix(rawPath, "http://") || strings.HasPrefix(rawPath, "https://") {
			resolvedURL, err := url.Parse(rawPath)
			if err != nil {
				continue
			}
			// Scope Protection
			if resolvedURL.Host != parsedBase.Host {
				continue
			}
			rawPath = resolvedURL.Path
		}

		// Bersihkan karakter query string atau fragment (?v=1.0 atau #token)
		if idx := strings.IndexAny(rawPath, "?#"); idx != -1 {
			rawPath = rawPath[:idx]
		}

		cleanPath := strings.TrimPrefix(rawPath, "/")
		
		// Filter Agresif: Singkirkan static assets & Cloudflare noise
		lowerPath := strings.ToLower(cleanPath)
		if lowerPath == "" || 
			strings.HasSuffix(lowerPath, ".js") || 
			strings.HasSuffix(lowerPath, ".css") ||
			strings.HasSuffix(lowerPath, ".png") || 
			strings.HasSuffix(lowerPath, ".jpg") || 
			strings.HasSuffix(lowerPath, ".jpeg") ||
			strings.HasSuffix(lowerPath, ".ico") ||
			strings.HasSuffix(lowerPath, ".svg") ||
			strings.Contains(lowerPath, "cdn-cgi/") {
			continue
		}

		// THREAD-SAFE BLOCK: Hanya untuk verifikasi & modifikasi map
		mapMutex.Lock()
		if !visited[cleanPath] {
			visited[cleanPath] = true
			localNewCount++
			// Simpan ke array lokal, jangan langsung kirim ke channel di dalam lock!
			localNewEndpoints = append(localNewEndpoints, cleanPath) 
		}
		mapMutex.Unlock()
	}

	// SAFE CHANNEL EMISSION: Kirim data ke fuzzer di luar cakupan lock (Anti-Deadlock)
	for _, endpoint := range localNewEndpoints {
		jobs <- endpoint
	}

	// LOGGING AKURAT: Hanya mencetak jika file JS ini menyumbang penemuan unik baru
	if localNewCount > 0 {
		fmt.Printf("[SUCCESS] JS Deep Parser murni menemukan %d endpoint baru dari: %s\n", localNewCount, jsURL)
	}
}

func worker(client *http.Client, baseURL string, jobs <-chan string, results chan<- DiagnosticResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for path := range jobs {
		// Bersihkan karakter awalan slash jika ada agar tidak terjadi double slash (//) pada URL
		cleanPath := strings.TrimPrefix(path, "/")
		fullURL := baseURL + cleanPath

		// Menggunakan HEAD request sebagai pertahanan pertama efisiensi I/O jaringan
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

		// Jika server menolak HEAD atau mengembalikan anomali, lakukan fallback ke GET untuk analisis mendalam
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

		// Kirim data valid atau anomali konfigurasi (403/500/301) ke aggregator
		results <- DiagnosticResult{
			Path:       cleanPath,
			StatusCode: statusCode,
			Size:       size,
			Type:       logType,
		}
	}
}

