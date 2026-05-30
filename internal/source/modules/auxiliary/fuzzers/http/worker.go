// worker.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

func calibrateSoft404(client *http.Client, baseURL string) {
	randomPath := fmt.Sprintf("Anomaly_Storm_%d.html", time.Now().UnixNano())
	resp, err := client.Get(baseURL + randomPath)
	if err != nil {
		fmt.Printf("[ERROR] Failed to perform initial calibration: %v\n", err)
		return
	}
	defer resp.Body.Close()

	safeReader := io.LimitReader(resp.Body, 1*1024*1024)
	body, _ := io.ReadAll(safeReader)
	soft404StatusCode = resp.StatusCode
	soft404Size = int64(len(body))
	soft404WordCount = len(strings.Fields(string(body)))
	soft404Fingerprint = getHTMLStructureFingerprint(string(body))
	fmt.Printf("[INFO] Soft 404 Detected. Byte Size => %d bytes\n", soft404Size)
	fmt.Printf("[INFO] Soft 404 Detected. Word Size => %d word\n", soft404WordCount)
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
			var currentWordCount int = 0
			var currentBodyString string = ""

			if statusCode == http.StatusMethodNotAllowed || statusCode == http.StatusOK {
				getReq, _ := http.NewRequest("GET", fullURL, nil)
				getReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
				
				getResp, err := client.Do(getReq)
				if err == nil {
					safeReader := io.LimitReader(getResp.Body, 5*1024*1024)
					body, _ := io.ReadAll(safeReader)
					size = int64(len(body))
					currentBodyString = string(body)
					currentWordCount = len(strings.Fields(string(body)))
					statusCode = getResp.StatusCode
					getResp.Body.Close()
				}
			}

			logType := "Valid"
			if statusCode == http.StatusNotFound {
				logType = "Not"
			} else if statusCode == soft404StatusCode {
				if size == soft404Size || currentWordCount == soft404WordCount || isSoft404Fuzzy(currentBodyString, soft404Fingerprint) {
					logType = "Soft"
				}
			} else if statusCode >= 500 {
				logType = "Error"
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
				Path:       currentJob.Path,
				StatusCode: statusCode,
				Size:       size,
				Words:      currentWordCount,
				Type:       logType,
			}
		}(job) 
	}
}
