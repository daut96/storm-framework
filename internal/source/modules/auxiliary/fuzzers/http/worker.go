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

func recordProfile(client *http.Client, targetURL string, category string) {
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("[!] Failed category calibration => %s : %v\n", category, err)
		return
	}
	defer resp.Body.Close()

	safeReader := io.LimitReader(resp.Body, 5*1024*1024)
	body, _ := io.ReadAll(safeReader)
	bodyString := string(body)

	// Simpan profil anomali ke dalam memori global
	Soft404Monsters[category] = Soft404Profile{
		StatusCode:  resp.StatusCode,
		Size:        int64(len(body)),
		WordCount:   len(strings.Fields(bodyString)),
		Fingerprint: getHTMLStructureFingerprint(bodyString),
	}
}

func advancedCalibration(client *http.Client, baseURL string) {
	// Inisialisasi Map Global
	Soft404Monsters = make(map[string]Soft404Profile)
	
	// Root dengan ekstensi statis
	probe1 := fmt.Sprintf("%s/anomaly_storm_%d.html", baseURL, time.Now().Unix())
	recordProfile(client, probe1, "first anomaly")

	// Rute bersarang / multi-level
	probe2 := fmt.Sprintf("%s/fastj/anomaly/%dq17rrp", baseURL, time.Now().UnixNano())
	recordProfile(client, probe2, "second anomaly")

	// Subfolder multi-level file
	probe3 := fmt.Sprintf("%s/bj4l40krd/yyanon/%d1.html", baseURL, time.Now().UnixNano())
	recordProfile(client, probe3, "third anomaly")

	// Root tanpa ekstensi
	probe4 := fmt.Sprintf("%s/ushqrt_%d", baseURL, time.Now().Unix())
	recordProfile(client, probe4, "fourth anomaly")

	probe5 := fmt.Sprintf("%s/0xjktt99/%d", baseURL, time.Now().Unix())
	recordProfile(client, probe5, "fifth anomaly")

	probe6 := fmt.Sprintf("%s/00PBB190/%dh2PP.html", baseURL, time.Now().Unix())
	recordProfile(client, probe6, "sixth anomaly")
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
			isSoft404 := false
			
			for _, profile := range Soft404Monsters {
				// Cek apakah status code-nya sama dengan salah satu profil jebakan
				if statusCode == profile.StatusCode {
					// Lakukan pengecekan berlapis (Size ATAU Word Count ATAU Fuzzy HTML)
					if size == profile.Size || currentWordCount == profile.WordCount || isSoft404Fuzzy(currentBodyString, profile.Fingerprint) {
						isSoft404 = true
						break // Langsung keluar loop, halaman ini positif sampah!
					}
				}
			}
			
			if isSoft404 {
				logType = "Soft"
			} else if statusCode == http.StatusNotFound {
				logType = "Not"
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
