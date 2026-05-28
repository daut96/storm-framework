// worker.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url" // WAJIB ditambahkan untuk url.Parse
	"strings"
	"sync"
	"time"
)

func calibrateSoft404(client *http.Client, baseURL string) {
	randomPath := fmt.Sprintf("anomaly_test_%d.html", time.Now().UnixNano())
	resp, err := client.Get(baseURL + randomPath)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		safeReader := io.LimitReader(resp.Body, 1*1024*1024)
		body, _ := io.ReadAll(safeReader)
		soft404Size = int64(len(body))
		log.Printf("[INFO] Soft 404 Detection Active. Baseline Size => %d bytes\n", soft404Size)
	}
}

// Parameter 'jobs' dihapus, worker membaca langsung dari WorkerQueue global
func worker(client *http.Client, baseURL string, results chan<- DiagnosticResult, wg *sync.WaitGroup) {
	// Menandakan bahwa worker ini sudah berhenti beroperasi ke WaitGroup lokal di main()
	defer wg.Done()

	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		log.Printf("[FATAL] Worker engine failed to parse Base URL: %v", err)
		return
	}

	for job := range WorkerQueue {
		// PENGGUNAAN CLOSURE: 
		// Memastikan instruksi defer tereksekusi pada level iterasi, bukan saat fungsi worker selesai.
		func(currentJob CrawlJob) {
			// Menandakan 1 tugas dari Antrean Pusat telah selesai, apapun hasil HTTP-nya
			defer GlobalTaskTracker.Done()

			fullURL := baseURL + currentJob.Path

			req, err := http.NewRequest("HEAD", fullURL, nil)
			if err != nil {
				return // return di sini setara dengan continue pada loop biasa
			}
			req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")

			resp, err := client.Do(req)
			if err != nil {
				return 
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()

			statusCode := resp.StatusCode
			var size int64 = 0

			if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusOK {
				getReq, _ := http.NewRequest("GET", fullURL, nil)
				getReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
				
				getResp, err := client.Do(getReq)
				if err == nil {
					safeReader := io.LimitReader(getResp.Body, 5*1024*1024)
					body, _ := io.ReadAll(safeReader)
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

			// =========================================================================
			// LOGIKA HYBRID CROSSOVER: JIT TRIGGER
			// =========================================================================
			if statusCode == http.StatusOK && strings.HasSuffix(strings.ToLower(currentJob.Path), ".js") && logType == "Valid" {
				
				// Tambahkan 1 beban kerja ke mesin sebelum men-spawn JS Extractor
				GlobalTaskTracker.Add(1)
				
				// Karena defer GlobalTaskTracker.Done() sudah kita tanam di DALAM fungsi extractFromJS
				// pada revisi file discovery.go sebelumnya, kita bisa men-spawn secara fire-and-forget.
				go extractFromJS(client, fullURL, parsedBase)
			}

			results <- DiagnosticResult{
				Source:     currentJob.Source,
				Path:       currentJob.Path, // BUG FIX: Menggunakan currentJob.Path, bukan cleanPath
				StatusCode: statusCode,
				Size:       size,
				Type:       logType,
			}
		}(job) 
	}
}
