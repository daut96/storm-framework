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
	Type       string `json:"type"`
}

type CrawlJob struct {
	Path   string
	Source string
}

// Global baseline untuk mendeteksi anomali Soft 404
var soft404Size int64 = -1

// Engine Regex Global
var linkFinderEngine *regexp.Regexp

// Pengganti map & mutex manual. sync.Map sangat efisien untuk operasi "Read-Mostly" / "Append-Only"
var visitedMap sync.Map

// Semaphore: Mencegah Goroutine Explosion. Maksimal 50 ekstraksi JS berjalan bersamaan.
var jsParseSemaphore = make(chan struct{}, 50)
