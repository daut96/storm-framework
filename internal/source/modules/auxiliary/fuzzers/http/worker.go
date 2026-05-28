// worker.go
package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
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
		safeReader := io.LimitReader(resp.Body, 1*1024*1024) // 1MB cukup untuk kalibrasi
		body, _ := io.ReadAll(safeReader)
		soft404Size = int64(len(body))
		log.Printf("[INFO] Soft 404 Detection Active. Baseline Size => %d bytes\n", soft404Size)
	}
}

func worker(client *http.Client, baseURL string, jobs <-chan CrawlJob, results chan<- DiagnosticResult, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range jobs {
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
			getReq.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
			
			getResp, err := client.Do(getReq)
			if err == nil {
				// Mitigasi OOM: Limit 5MB saat melakukan validasi payload
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

		results <- DiagnosticResult{
			Source:     job.Source,
			Path:       cleanPath,
			StatusCode: statusCode,
			Size:       size,
			Type:       logType,
		}
	}
}
