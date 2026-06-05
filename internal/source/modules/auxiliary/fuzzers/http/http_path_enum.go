// main.go
// GPL License.
// Copyright (c) 2026 Storm Framework
// See LICENSE file in the project root for full license information.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	targetURL := flag.String("url", "", "Base URL target")
	wordlistPath := flag.String("wordlist", "", "Path to wordlist file (opsional)")
	threads := flag.Int("threads", 1, "Number of concurrent workers")
	regex := flag.String("regex", "", "Path to custom regex file for JS parsing")
	flag.Parse()

	if *targetURL == "" {
		fmt.Errorf("Error => The url parameter is absolutely required.")
		os.Exit(1)
	}

	if *regex != "" {
		err := initDynamicRegex(*regex)
		if err != nil {
			fmt.Errorf("Error => Failed to initialize custom regex: %v\n", err)
		}
	}

	if !strings.HasSuffix(*targetURL, "/") {
		*targetURL += "/"
	}

	// Transport layer yang di-tuning untuk agresivitas tanpa mematikan local port OS (Port Exhaustion)
	transport := &http.Transport{
		MaxConnsPerHost:     100,             // Batasi koneksi simultan ke 1 target
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,            // Optimasi Multiplexing jika target support
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,          // Hard timeout per round-trip request
	}

	advancedCalibration(client, *targetURL)

	results := make(chan DiagnosticResult, 100000)
	
	var workerWG sync.WaitGroup

	go JobDispatcher()
	
	for i := 0; i < *threads; i++ {
		workerWG.Add(1)
		go worker(client, *targetURL, results, &workerWG)
	}

	doneAggregator := make(chan bool)

	go func() {
		for res := range results {
			fmt.Printf("[RESULT] [%s] Path: %s | Status: %d | Size: %d | Words: %d | Type: %s\n",
				res.Source, res.Path, res.StatusCode, res.Size, res.Words, res.Type,
			)
		}
		doneAggregator <- true
	}()

	GlobalTaskTracker.Add(1)
	go discoverPathsAutomatically(client, *targetURL)
	
	if *wordlistPath != "" {
		fmt.Println("[INFO] Mode => Wordlist detected. Running Combo Wordlist & Crawling")
		go func(path string) {
			loadWordlist(path)
		}(*wordlistPath)
	} else {
		fmt.Println("[INFO] Mode => Empty wordlist. Running JIT Crawling Recursive")
	}
	GlobalTaskTracker.Wait()

	// GRACEFUL SHUTDOWN
	close(DiscoveryChannel) // Mematikan Dispatcher -> Mematikan WorkerQueue
	workerWG.Wait()         // Memastikan Worker meletakkan alat kerjanya
	close(results)          // Mematikan Aggregator layar
	<-doneAggregator
}

