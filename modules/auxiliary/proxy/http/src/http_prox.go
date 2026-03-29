package main

import (
	"fmt"
	"net"
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
	port := "0.0.0.0:7900" // Paksa ke IPv4 agar tidak bentrok dengan IPv6 sistem

	// 1. BUAT LISTENER MANUAL (Langkah paling aman)
	// Dengan menggunakan net.Listen secara terpisah, kita punya kontrol 
	// lebih besar sebelum server HTTP mengambil alih.
	l, err := net.Listen("tcp4", port)
	if err != nil {
		log.Fatalf("Gagal mengunci port %s: %v. Pastikan tidak ada zombie process!", port, err)
	}

	log.Printf("Proxy Berjalan di %s", l.Addr().String())

	// 2. MASUKKAN KE SERVER
	server := &http.Server{
		Handler: http.HandlerFunc(proxyHandler),
	}

	// Gunakan .Serve(l), bukan .ListenAndServe()
	if err := server.Serve(l); err != nil {
		log.Fatal(err)
	}
}
