// main.go
// GPL License.
// Copyright (c) 2026 Storm Framework
// See LICENSE file in the project root for full license information.
package main

import (
	"flag"
	"fmt"
	"log"
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
		log.Printf("Error => The url parameter is absolutely required.")
		os.Exit(1)
	}

	if *regex != "" {
		err := initDynamicRegex(*regex)
		if err != nil {
			log.Fatalf("Error => Failed to initialize custom regex: %v\n", err)
		}
	}

	if !strings.HasSuffix(*targetURL, "/") {
		*targetURL += "/"
	}

	// Transport layer yang di-tuning untuk agresivitas tanpa mematikan local port OS (Port Exhaustion)
	transport := &http.Transport{
		MaxConnsPerHost:     200,             // Batasi koneksi simultan ke 1 target
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		ForceAttemptHTTP2:   true,            // Optimasi Multiplexing jika target support
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,          // Hard timeout per round-trip request
	}

	calibrateSoft404(client, *targetURL)

	jobs := make(chan CrawlJob, 10000)
	results := make(chan DiagnosticResult, 10000)
	var wg sync.WaitGroup

	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go worker(client, *targetURL, jobs, results, &wg)
	}

	doneAggregator := make(chan bool)

	go func() {
		for res := range results {
			fmt.Printf("[RESULT] [%s] Path: %s | Status: %d | Size: %d | Type: %s\n",
				res.Source, res.Path, res.StatusCode, res.Size, res.Type,
			)
		}
		doneAggregator <- true
	}()

	if *wordlistPath != "" {
		fmt.Println("[INFO] Mode => Using Static Wordlist Input")
		loadWordlist(*wordlistPath, jobs)
	} else {
		fmt.Println("[INFO] Mode => Empty wordlist. Enable JIT Crawling & Parsing")
		discoverPathsAutomatically(client, *targetURL, jobs)
	}

	// GRACEFUL SHUTDOWN
	close(jobs)
	wg.Wait()
	close(results)
	<-doneAggregator
}

