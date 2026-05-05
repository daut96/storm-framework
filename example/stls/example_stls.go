package main

import (
	"fmt"
	"log"
	"github.com/StormWorld0/storm-framework/scripts/wrapper/stls_go" 
)

func main() {
	fmt.Println("[*] Memulai Storm STLS Engine...")

	// ==========================================================
	// TAHAP 1: INISIALISASI MESIN (WAJIB DILAKUKAN PERTAMA)
	// ==========================================================
	// Ini akan memicu Upward Traversal untuk mencari libstls.so
	// dan memuatnya ke memori via C-ABI dlopen.
	if err := stls_go.InitSTLS(); err != nil {
		log.Fatalf("[FATAL] Gagal menyalakan mesin STLS: %v\n", err)
	}

	// ==========================================================
	// TAHAP 2: PERSIAPAN REQUEST
	// ==========================================================
	targetURL := "https://tls.peet.ws/api/all" // Endpoint terbaik untuk cek JA3 Fingerprint

	// Kamu bisa memasukkan header dinamis dari logic recon/dos kamu di sini
	headers := map[string]string{
		"Accept":          "application/json",
		"Accept-Language": "en-US,en;q=0.9",
		"Content-Type":    "application/json",
		// WAF biasanya mengecek header referer dan user-agent origin
		"Referer":         "https://google.com", 
	}

	// ==========================================================
	// TAHAP 3: EKSEKUSI (BISA DI-LOOP / DIMASUKKAN KE GOROUTINE)
	// ==========================================================
	fmt.Printf("[*] Mengirim GET request ke: %s\n", targetURL)
	
	// Contoh pemanggilan GET (tanpa body)
	response, err := stls_go.Get(targetURL, headers)
	if err != nil {
		log.Fatalf("[ERROR] WAF memblokir atau koneksi gagal: %v\n", err)
	}

	fmt.Println("[+] GET Request Berhasil! Ekstraksi Payload:")
	fmt.Println(response.Text)

	// ----------------------------------------------------------
	// Contoh pemanggilan POST (dengan body / payload eksploitasi)
	// ----------------------------------------------------------
	fmt.Println("\n[*] Mengirim POST request...")
	postTarget := "https://httpbin.org/post"
	payloadByte := []byte(`{"storm_command": "execute", "bypass": true}`)

	postResp, err := stls_go.Post(postTarget, headers, payloadByte)
	if err != nil {
		log.Fatalf("[ERROR] POST gagal: %v\n", err)
	}

	fmt.Println("[+] POST Request Berhasil:")
	fmt.Println(postResp.Text)
}
