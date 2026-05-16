package main

import (
	"log"
	"net/http"
	"net/http/httputil"

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
	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			log.Printf("[DPI-REQ] Intercept packets from: %s -> %s", req.RemoteAddr, req.Host)

			// Menggunakan httputil.DumpRequest untuk membaca seluruh header dan body.
			// Secara internal, fungsi ini aman digunakan karena setelah membaca stream body,
			// ia akan mengalokasikan ulang buffer (ReadCloser) sehingga stream tidak terputus (drained).
			requestDump, err := httputil.DumpRequest(req, true)
			if err != nil {
				log.Printf("[ERROR] Failed to perform request inspection: %v\n", err)
				// Kembalikan request asli agar traffic tidak drop jika terjadi error logis
				return req, nil 
			}
			
			// Output payload ke stdout
			log.Printf("\n========== INCOMING DECRYPTED REQUEST ==========\n%s\n================================================\n", string(requestDump))

			// CONTOH IMPLEMENTASI DPI RULE ENGINE:
			// Di sini Anda bisa melakukan inspeksi signature/pattern matching pada payload mentah.
			// if bytes.Contains(requestDump, []byte("XSS-Payload-Attack")) {
			//     log.Printf("[ALERT] Serangan terdeteksi! Memblokir request.")
			//     return req, goproxy.NewResponse(req, goproxy.ContentTypeText, http.StatusForbidden, "Blocked by DPI Engine")
			// }

			return req, nil
		})

	// 4. Logika DPI: Response Inspection (Mencegat & Membaca Response dari Target Server)
	// Hook ini memungkinkan Anda menginspeksi data yang dikembalikan oleh server tujuan sebelum di-enkripsi
	// kembali oleh proxy dan dikirim ke client asal.
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if resp == nil {
				return resp
			}

			log.Printf("[DPI-RES] Intercepting the return payload from: %s", ctx.Req.Host)

			// DumpResponse membaca seluruh data HTTP response (Status, Header, dan Body plaintext)
			responseDump, err := httputil.DumpResponse(resp, true)
			if err != nil {
				log.Printf("[ERROR] Failed to perform response inspection: %v\n", err)
				return resp
			}

			log.Printf("\n========== INCOMING DECRYPTED RESPONSE ==========\n%s\n=================================================\n", string(responseDump))

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

