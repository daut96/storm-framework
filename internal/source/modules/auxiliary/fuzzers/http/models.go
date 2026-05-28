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

// Global regex & Mutex
var linkFinderEngine *regexp.Regexp
var mapMutex sync.Mutex

