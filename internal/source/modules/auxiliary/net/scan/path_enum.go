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
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

type DiagnosticResult struct {
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
	Type       string `json:"type"`
}

// Global baseline untuk mendeteksi anomali Soft 404
var soft404Size int64 = -1

func main() {
	// Configuration
	targetURL := flag.String("url", "", "Base URL target")
	wordlistPath := flag.String("wordlist", "", "Path to wordlist file (opsional)")
	threads := flag.Int("threads", 1, "Number of concurrent workers")
	flag.Parse()

	if *targetURL == "" {
		log.Printf("Error => The url parameter is absolutely required.")
		os.Exit(1)
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

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	// Regex untuk mengekstrak path internal aplikasi dari source code HTML
	re := regexp.MustCompile(`(?:href|src)=["'](?:/)([^"'#\s>]+)["']`)
	matches := re.FindAllStringSubmatch(bodyStr, -1)

	// Set internal untuk de-duplikasi path agar tidak melakukan scan ganda
	visited := make(map[string]bool)
	for _, match := range matches {
		path := match[1]
		if !visited[path] && !strings.HasPrefix(path, "http") {
			visited[path] = true
			jobs <- path
		}
	}
	fmt.Printf("[SUCCESS] Successfully extracted => %d\n", len(visited))
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

		// FILTERING LOGIC: Validasi keaslian path
		if statusCode == http.StatusNotFound {
			continue // Valid 404, abaikan.
		}

		if statusCode == http.StatusOK && size == soft404Size {
			continue // Terdeteksi sebagai anomali Soft 404, abaikan.
		}

		// Kirim data valid atau anomali konfigurasi (403/500/301) ke aggregator
		results <- DiagnosticResult{
			Path:       cleanPath,
			StatusCode: statusCode,
			Size:       size,
			Type:       "Detected",
		}
	}
}

