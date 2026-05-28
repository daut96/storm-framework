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
		Timeout:   5 * time.Second,
	}

	// Kalibrasi Anomali Jaringan (Deteksi Soft 404)
	calibrateSoft404(client, *targetURL)

	// Inisialisasi Channels
	jobs := make(chan CrawlJob, 5000)
	results := make(chan DiagnosticResult, 5000)
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
			fmt.Printf("[RESULT] [%s] Path: %s | Status: %d | Size: %d | Type: %s\n",
				res.Source, res.Path, res.StatusCode, res.Size, res.Type,
			)
		}
		doneAggregator <- true
	}()

	// Penentuan Mekanisme Input
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
