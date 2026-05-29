package main

import (
	"strings"
)

// Ekstrak struktur HTML menjadi sidik jari string yang bersih
func getHTMLStructureFingerprint(htmlContent string) string {
	// Temukan semua tag HTML
	matches := tagRegex.FindAllString(htmlContent, -1)
	
	// Gabungkan semua tag menjadi satu string panjang sebagai "sidik jari"
	return strings.Join(matches, "")
}

// Hitung persentase kemiripan sederhana berbasis panjang sidik jari
func isSoft404Fuzzy(currentHTML, baselineFingerprint string) bool {
	if baselineFingerprint == "" {
		return false
	}
	
	currentFingerprint := getHTMLStructureFingerprint(currentHTML)
	
	// Jika strukturnya sama persis (100% cocok)
	if currentFingerprint == baselineFingerprint {
		return true
	}
	
	// Logika Kemiripan Panjang Struktur (Pendekatan Cepat 90%)
	lenCurrent := float64(len(currentFingerprint))
	lenBaseline := float64(len(baselineFingerprint))
	
	if lenBaseline == 0 {
		return false
	}
	
	ratio := lenCurrent / lenBaseline
	if ratio > 0.90 && ratio < 1.10 {
		// Jika panjang struktur tag mirip dalam rentang 90% - 110%, 
		// besar kemungkinan ini adalah template error yang sama dengan input dinamis
		return true
	}
	
	return false
}

