// models.go
package main

import (
	"regexp"
	"sync"
)

type DiagnosticResult struct {
	Source     string `json:"source"`
	Path       string `json:"path"`
	StatusCode int    `json:"status_code"`
	Size       int64  `json:"size"`
	Words      int    `json:"words"`
	Type       string `json:"type"`
}

type CrawlJob struct {
	Path   string
	Source string
}

// Soft 404 detection
type HeuristicBaseline struct {
	StatusCode     int
	DOMFingerprint string
	BodyText       string
	Title          string
	H1             string
}
var Soft404Baselines map[string]HeuristicBaseline

// Regex / Sync / Max Goroutine
var linkFinderEngine *regexp.Regexp
var visitedMap sync.Map
var jsParseSemaphore = make(chan struct{}, 50)

// === ARSITEKTUR BARU ===
// GlobalTaskTracker memastikan program tidak berhenti sebelum SELURUH rekursi JS selesai
var GlobalTaskTracker sync.WaitGroup

// DiscoveryChannel menerima temuan path baru dari Worker / Mesin JS secara dinamis
var DiscoveryChannel = make(chan CrawlJob)

// WorkerQueue adalah antrean yang dikonsumsi oleh barisan Worker untuk eksekusi HTTP
var WorkerQueue = make(chan CrawlJob)
