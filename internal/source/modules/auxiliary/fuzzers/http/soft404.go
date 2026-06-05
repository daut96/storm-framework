package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// 1. FUNGSI EKSTRAKTOR TOKENIZER (Pengganti Regex)
func extractFeaturesRobust(htmlContent string) HeuristicBaseline {
	tokenizer := html.NewTokenizer(strings.NewReader(htmlContent))
	var dom, text strings.Builder
	var title, h1, currentTag string

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		token := tokenizer.Token()
		switch tt {
		case html.StartTagToken, html.SelfClosingTagToken:
			dom.WriteString("<" + token.Data + ">")
			currentTag = token.Data
		case html.EndTagToken:
			dom.WriteString("</" + token.Data + ">")
			currentTag = ""
		case html.TextToken:
			if currentTag != "script" && currentTag != "style" {
				trimmed := strings.TrimSpace(token.Data)
				if trimmed != "" {
					text.WriteString(trimmed + " ")
				}
				if currentTag == "title" {
					title += token.Data
				} else if currentTag == "h1" {
					h1 += token.Data
				}
			}
		}
	}
	return HeuristicBaseline{
		DOMFingerprint: dom.String(),
		BodyText:       strings.Join(strings.Fields(text.String()), " "),
		Title:          strings.TrimSpace(title),
		H1:             strings.TrimSpace(h1),
	}
}

// 2. FUNGSI MATEMATIKA LEVENSHTEIN & SIMILARITY
func levenshteinDistance(s1, s2 string) int {
	r1, r2 := []rune(s1), []rune(s2)
	len1, len2 := len(r1), len(r2)
	if len1 == 0 { return len2 }
	if len2 == 0 { return len1 }

	row := make([]int, len2+1)
	for i := 0; i <= len2; i++ { row[i] = i }

	for i := 1; i <= len1; i++ {
		prev := i
		for j := 1; j <= len2; j++ {
			val := row[j-1]
			if r1[i-1] != r2[j-1] {
				min := row[j] + 1
				if prev+1 < min { min = prev + 1 }
				if row[j-1]+1 < min { min = row[j-1] + 1 }
				val = min
			}
			row[j-1] = prev
			prev = val
		}
		row[len2] = prev
	}
	return row[len2]
}

func calculateSimilarity(s1, s2 string) float64 {
	if s1 == "" && s2 == "" { return 1.0 }
	maxLen := float64(len(s1))
	if float64(len(s2)) > maxLen { maxLen = float64(len(s2)) }
	if maxLen == 0 { return 0.0 }
	dist := float64(levenshteinDistance(s1, s2))
	return 1.0 - (dist / maxLen)
}

// 3. FUNGSI EVALUASI UTAMA (Dipanggil oleh Worker)
func isHeuristicSoft404(currentHTML string) bool {
	current := extractFeaturesRobust(currentHTML)
	errorKeywords := []string{
        "404",
        "not found",
        "page not found",
        "resource not found",
        "file not found",
        "document not found",
        "content not found",

        "does not exist",
        "doesn't exist",
        "page does not exist",
        "page doesn't exist",

        "cannot find",
        "can't find",
        "could not find",
        "couldn't find",

        "page unavailable",
        "resource unavailable",
        "requested resource",

        "invalid url",
        "invalid request",
        "invalid path",
        "unknown page",
        "unknown route",

        "oops",
        "something went wrong",

        // Indonesia
        "tidak ditemukan",
        "halaman tidak ditemukan",
        "resource tidak ditemukan",
        "konten tidak ditemukan",
        "file tidak ditemukan",
        "url tidak valid",
        "permintaan tidak valid",
        "halaman tidak tersedia",
    }
	
	// Cek Keyword Error
	simKW := 0.0
	currentLower := strings.ToLower(current.Title + " " + current.H1)
	for _, kw := range errorKeywords {
		if strings.Contains(currentLower, kw) {
			simKW = 1.0
			break
		}
	}

	// Loop pengecekan terhadap 6 profil kalibrasi
	for _, baseline := range Soft404Baselines {
		simDOM := calculateSimilarity(current.DOMFingerprint, baseline.DOMFingerprint)
		simText := calculateSimilarity(current.BodyText, baseline.BodyText)
		simTitle := calculateSimilarity(current.Title, baseline.Title)
		simH1 := calculateSimilarity(current.H1, baseline.H1)

		finalScore := (simDOM * 40.0) + (simText * 25.0) + (simTitle * 15.0) + (simH1 * 10.0) + (simKW * 10.0)
		
		if finalScore >= 80.0 {
			return true // Positif Soft 404
		}
	}
	return false // Valid (Bukan Soft 404)
}

// 4. FUNGSI KALIBRASI (Dipindahkan dari worker.go)
func recordProfile(client *http.Client, targetURL string, category string) {
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:126.0) Gecko/20100101 Firefox/126.0")
	resp, err := client.Do(req)
	if err != nil { return }
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	Soft404Baselines[category] = extractFeaturesRobust(string(body))
}

func advancedCalibration(client *http.Client, baseURL string) {
	Soft404Baselines = make(map[string]HeuristicBaseline)
	
	probes := map[string]string{
		"first anomaly":  fmt.Sprintf("%sanomaly_storm_%d.html", baseURL, time.Now().Unix()),
		"second anomaly": fmt.Sprintf("%sfastj/anomaly/%dq17rrp", baseURL, time.Now().UnixNano()),
		"third anomaly":  fmt.Sprintf("%sbj4l40krd/yyanon/%d1.html", baseURL, time.Now().UnixNano()),
		"fourth anomaly": fmt.Sprintf("%sushqrt_%d", baseURL, time.Now().Unix()),
		"fifth anomaly":  fmt.Sprintf("%s0xjktt99/%d", baseURL, time.Now().Unix()),
		"sixth anomaly":  fmt.Sprintf("%s00PBB190/%dh2PP.html", baseURL, time.Now().Unix()),
	}

	for category, url := range probes {
		recordProfile(client, url, category)
	}
}
