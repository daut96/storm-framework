// orchestrator.go
package main

import "strings"

// SubmitJob adalah gerbang tunggal (Single Point of Entry) untuk semua path baru.
// Fungsi ini menangani deduplikasi Atomic O(1) DAN inkremen pelacak tugas.
func SubmitJob(path string, source string) {
	cleanPath := strings.TrimPrefix(path, "/")
	
	if _, loaded := visitedMap.LoadOrStore(cleanPath, true); !loaded {
		GlobalTaskTracker.Add(1) // Tambah 1 tugas ke sistem
		DiscoveryChannel <- CrawlJob{Path: cleanPath, Source: source}
	}
}

// JobDispatcher bertindak sebagai Unbounded Queue.
// Mengurai kemacetan lalu lintas antara Produsen (JS Extractor) dan Konsumen (Worker).
func JobDispatcher() {
	var queue []CrawlJob
	for {
		if len(queue) == 0 {
			// Mode Menunggu: Jika antrean kosong, blokir sampai ada temuan baru
			job, ok := <-DiscoveryChannel
			if !ok {
				// Jika Discovery ditutup, tutup juga WorkerQueue
				close(WorkerQueue)
				return
			}
			queue = append(queue, job)
		} else {
			// Mode Sibuk: Bisa menerima temuan baru ATAU melempar tugas ke Worker
			select {
			case job, ok := <-DiscoveryChannel:
				if !ok {
					// Kuras sisa antrean darurat sebelum mati
					for _, j := range queue {
						WorkerQueue <- j
					}
					close(WorkerQueue)
					return
				}
				queue = append(queue, job)
			case WorkerQueue <- queue[0]:
				// Optimasi Garbage Collector (Mencegah Memory Leak)
				queue[0] = CrawlJob{} 
				queue = queue[1:]
				
				// Reset kapasitas memori dasar jika antrean sudah habis terproses
				if len(queue) == 0 {
					queue = nil
				}
			}
		}
	}
}

