package main

import (
	
	"context"
	"syscall"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
)

// proxyHandler bertanggung jawab mencegat, melogging, dan meneruskan request
func proxyHandler(w http.ResponseWriter, req *http.Request) {
	// 1. Intercept & Log Request
	// Parameter 'true' memastikan kita juga mendump body (isi) dari request
	requestDump, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Printf("[ERROR] Gagal membaca request: %v\n", err)
		http.Error(w, "Gagal membaca request", http.StatusInternalServerError)
		return
	}

	// Menampilkan log secara rapi ke stdout
	log.Printf("\n========== INCOMING PACKET ==========\n%s\n=====================================\n", string(requestDump))

	// 2. Persiapan Forwarding
	// Saat bertindak sebagai proxy, RequestURI tidak boleh diisi pada object request 
	// yang akan dikirim menggunakan http.Client. Kita harus mengosongkannya.
	req.RequestURI = ""

	// 3. Eksekusi Request ke Destination Server
	// Menggunakan custom client untuk mencegah auto-redirect.
	// Proxy tidak boleh mengikuti redirect, melainkan harus mengembalikan response redirect tersebut ke client asal.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[ERROR] Gagal meneruskan ke %s: %v\n", req.Host, err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 4. Proses Respon: Salin Header dari Target kembali ke Client asal
	copyHeaders(w.Header(), resp.Header)

	// 5. Proses Respon: Salin Status Code
	w.WriteHeader(resp.StatusCode)

	// 6. Proses Respon: Stream Body secara real-time ke Client asal
	// io.Copy sangat efisien karena memindahkan data dalam chunk tanpa memuat seluruhnya ke RAM
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("[ERROR] Gagal menyalin response body: %v\n", err)
	}
}

// copyHeaders adalah helper untuk menyalin map header secara persis
func copyHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func main() {
	addr := ":7900"
	
	// 1. Buat konfigurasi Listener khusus
	lc := net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				// Paksa OS agar mengizinkan penggunaan ulang port (SO_REUSEADDR)
				syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
			})
		},
	}

	// 2. Lakukan Listen secara manual menggunakan konfigurasi di atas
	lp, err := lc.Listen(context.Background(), "tcp4", addr)
	if err != nil {
		log.Fatalf("Gagal merebut port %s: %v", addr, err)
	}

	log.Printf("Proxy aktif di %s (Mode: Force Bind)", addr)

	// 3. Masukkan listener ke HTTP Server
	server := &http.Server{
		Handler: http.HandlerFunc(proxyHandler),
	}

	if err := server.Serve(lp); err != nil {
		log.Fatal(err)
	}
}
