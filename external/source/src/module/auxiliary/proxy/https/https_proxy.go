package main

import (
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/elazarl/goproxy"
)

func main() {
	// 1. Definisi Command Line Flags
	certPath := flag.String("cert", "", "Path to custom Root CA certificate (.crt)")
	keyPath := flag.String("key", "", "Path to custom Root CA private key (.key)")
	ip := flag.String("ip", "0.0.0.0", "IP address determines network traffic")
	port := flag.String("port", "6880", "Port for the proxy server")
	flag.Parse() // Mengeksekusi parser argumen

	log.Printf("[INIT] Initializing HTTPS Intercepting Proxy on %s:%s", *ip, *port)

	proxy := goproxy.NewProxyHttpServer()
	proxy.Verbose = true

	// 2. Logika Injeksi Custom Root CA
	if *certPath != "" && *keyPath != "" {
		log.Printf("[INIT] Loading Custom Root CA from: %s & %s", *certPath, *keyPath)
		
		// Memuat pasangan kunci publik (CRT) dan privat (KEY)
		caCert, err := tls.LoadX509KeyPair(*certPath, *keyPath)
		if err != nil {
			log.Fatalf("[FATAL] Failed to load Custom CA: %v", err)
		}
		
		// Override CA internal goproxy
		goproxy.GoproxyCa = caCert
		
		// Set aturan MITM dengan CA yang baru
		proxy.OnRequest().HandleConnect(goproxy.FuncHttpsHandler(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			return &goproxy.ConnectAction{
				Action:    goproxy.ConnectMitm,
				TLSConfig: goproxy.TLSConfigFromCA(&caCert),
			}, host
		}))
	} else {
		log.Println("[WARN] No CA flags provided. Using goproxy default internal CA.")
		proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	}

	// 3. Mesin DPI (Response Inspection)
	proxy.OnResponse().DoFunc(
		func(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
			if resp == nil || resp.Body == nil {
				return resp
			}

			contentType := resp.Header.Get("Content-Type")
			contentEncoding := resp.Header.Get("Content-Encoding")

			log.Printf("[DPI-RES] Intercepting responses from: %s", ctx.Req.Host)

			// Dump Header
			responseHeaders, _ := httputil.DumpResponse(resp, false)
			log.Printf("\n========== INCOMING RESPONSE HEADERS ==========\n%s\n", string(responseHeaders))

			// Ekstrak Stream Body
			rawBodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("[ERROR] Failed to read response body: %v\n", err)
				return resp
			}
			resp.Body.Close()
			resp.Body = io.NopCloser(bytes.NewBuffer(rawBodyBytes))

			if strings.Contains(contentType, "text/html") {
				log.Printf("[DPI-BYPASS] Ignoring HTML payload from: %s", ctx.Req.Host)
				return resp
			}

			if strings.Contains(contentType, "text/css") {
				log.Printf("[DPI-BYPASS] Ignoring CSS payload from: %s", ctx.Req.Host)
				return resp
			}

			if strings.Contains(contentType, "application/x-javascript") {
				log.Printf("[DPI-BYPASS] Ignoring JavaScript payload from: %s", ctx.Req.Host)
				return resp
			}
			
			// Logika Dekompresi dan Sanitasi UTF-8
			if contentEncoding == "gzip" {
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

				safeGzipText := strings.ToValidUTF8(string(uncompressedBytes), "")
				log.Printf("========== DECOMPRESSED GZIP PAYLOAD ==========\n%s\n===============================================\n\n", safeGzipText)
			
			} else if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/") {
				safePlainText := strings.ToValidUTF8(string(rawBodyBytes), "")
				log.Printf("========== PLAINTEXT PAYLOAD ==========\n%s\n=======================================\n\n", safePlainText)

			} else {
				log.Printf("[DPI-INFO] Ignore binary payloads with type: %s\n", contentType)
			}

			return resp
		})

	server := &http.Server{
		Addr:    *ip + ":" + *port,
		Handler: proxy,
	}

	log.Printf("[START] Waiting for encryption traffic...")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("[FATAL] Failed to run proxy server: %v", err)
	}
}
