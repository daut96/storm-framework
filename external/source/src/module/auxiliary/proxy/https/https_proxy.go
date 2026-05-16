package main

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/elazarl/goproxy"
)


func main() {
	port := "0.0.0.0:6880"
	log.Printf("[INIT] Initializing HTTPS Intercepting Proxy on %s...", port)

	// 1. Inisialisasi goproxy engine
	// Object ini menggantikan seluruh HTTP handler manual dan mengelola pool koneksi internal
	proxy := goproxy.NewProxyHttpServer()
	
	// Set ke true jika Anda ingin melihat log internal goproxy (state handshake, connection pooling)
	proxy.Verbose = false 

	// 2. Intercept Method CONNECT (Mengaktifkan Man-in-the-Middle)
	// Baris ini mengubah proxy dari Layer 4 Tunnel biasa menjadi Layer 7 Interceptor.
	// Setiap kali ada request "CONNECT target.com:443", goproxy akan membajak koneksi tersebut,
	// melakukan TLS termination, dan menandatangani sertifikat tiruan secara on-the-fly.
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)

	// 3. Logika DPI: Request Inspection (Mencegat & Membaca Request Plaintext)
	// Hook ini dieksekusi SETELAH TLS didekripsi. Paket yang Anda baca di sini sudah berupa plaintext.
	proxy.OnResponse().DoFunc(
	func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
		if resp == nil || resp.Body == nil {
			return resp
		}

		log.Printf("[DPI-RES] Intercepting responses from: %s", ctx.Req.Host)

		// 1. DUMP HEADER SAJA
		// Set false agar httputil tidak membaca body biner dan merusak terminal
		responseHeaders, _ := httputil.DumpResponse(resp, false)
		log.Printf("\n========== INCOMING RESPONSE HEADERS ==========\n%s\n", string(responseHeaders))

		// 2. EKSTRAK BODY STREAM KE MEMORI (RAM)
		rawBodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[ERROR] Failed to read response body: %v\n", err)
			return resp
		}
		resp.Body.Close()

		// [KRITIKAL] KEMBALIKAN STREAM KE STATE SEMULA
		// Kita membuat objek ReadCloser baru dari raw bytes yang sudah kita salin,
		// lalu memasukkannya kembali ke resp.Body agar goproxy bisa mengirimnya ke klien.
		resp.Body = io.NopCloser(bytes.NewBuffer(rawBodyBytes))

		// 3. MESIN INSPEKSI PAYLOAD (DPI ENGINE)
		contentType := resp.Header.Get("Content-Type")
		contentEncoding := resp.Header.Get("Content-Encoding")

		if contentEncoding == "gzip" {
			// Logika Dekompresi
			gzipReader, err := gzip.NewReader(bytes.NewReader(rawBodyBytes))
			if err != nil {
				log.Printf("[WARN] Failed to initialize gzip decompressor: %v\n", err)
				return resp
			}
			defer gzipReader.Close()

			uncompressedBytes, err := io.ReadAll(gzipReader)
			if err != nil {
				log.Printf("[WARN] Failed to read gzip contents: %v\n", err)
				return resp
			}

			// Tampilkan hasil yang sudah didekode menjadi UTF-8
			log.Printf("========== DECOMPRESSED GZIP PAYLOAD ==========\n%s\n===============================================\n", string(uncompressedBytes))
				
		} else if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json") {
			// Jika payload aslinya memang sudah plaintext (tidak di-gzip)
			log.Printf("========== PLAINTEXT PAYLOAD ==========\n%s\n=======================================\n", string(rawBodyBytes))
				
		} else {
			// Jika yang lewat adalah gambar (.jpg), video, atau binary file (.apk), 
			// kita abaikan untuk mencegah terminal/Python loader Anda crash.
			log.Printf("[DPI-INFO] Ignore binary payloads with type: %s\n", contentType)
		}

		return resp
	})

	// 5. Konfigurasi dan Penayangan Server
	server := &http.Server{
		Addr:    port,
		Handler: proxy, // goproxy memenuhi kriteria interface http.Handler
	}

	log.Printf("[START] Waiting for encryption traffic...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[FATAL] Failed to run proxy server: %v", err)
	}
}

