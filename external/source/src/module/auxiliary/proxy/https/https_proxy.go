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

	////
	// Mesin DPI (Request Inspection)
	proxy.OnRequest().DoFunc(
		func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			if req == nil {
				return req, nil
			}

			contentType := req.Header.Get("Content-Type")
			contentEncoding := req.Header.Get("Content-Encoding")

			log.Printf("[DPI-REQ] Intercepting request to: %s%s", req.Host, req.URL.Path)

			// Dump request header
			reqDump, _ := httputil.DumpRequestOut(req, false)
			log.Printf("\n========== OUTGOING REQUEST HEADERS ==========\n%s\n", string(reqDump))

			// Baca body jika ada
			if req.Body != nil && req.Body != http.NoBody {
				rawBodyBytes, err := io.ReadAll(req.Body)
				req.Body.Close()
				if err != nil {
					log.Printf("[ERROR] Failed to read request body: %v", err)
					// Kembalikan body kosong agar request tidak rusak
					req.Body = io.NopCloser(bytes.NewBuffer(nil))
					return req, nil
				}
				// Selalu pulihkan body agar bisa digunakan oleh handler lain
				req.Body = io.NopCloser(bytes.NewBuffer(rawBodyBytes))

				// Parsing contentType html
				if strings.Contains(contentType, "text/html") {
				    log.Printf("[DPI-BYPASS] Ignoring HTML request from: %s", req.Host)
				    return req, nil
			    }

				// Parsing contentType css
			    if strings.Contains(contentType, "text/css") {
				    log.Printf("[DPI-BYPASS] Ignoring CSS request from: %s", req.Host)
				    return req, nil
			    }

				// Parsing contentType js
			    if strings.Contains(contentType, "application/x-javascript") {
				    log.Printf("[DPI-BYPASS] Ignoring JavaScript request from: %s", req.Host)
				    return req, nil
		    	}

				// Logika dekompresi dan sanitasi UTF-8
				if contentEncoding == "gzip" {
					gzipReader, err := gzip.NewReader(bytes.NewReader(rawBodyBytes))
					if err != nil {
						log.Printf("[WARN] Failed to init gzip for requests: %v", err)
						return req, nil
					}
					defer gzipReader.Close()

					uncompressedBytes, err := io.ReadAll(gzipReader)
					if err != nil {
						log.Printf("[WARN] Failed to read gzip request body: %v", err)
						return req, nil
					}
					safeText := strings.ToValidUTF8(string(uncompressedBytes), "")
					log.Printf("========== DECOMPRESSED GZIP REQUEST PAYLOAD ==========\n%s\n=====================================================\n\n", safeText)
				
				} else if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json") ||
                          strings.HasPrefix(contentType, "application/xml") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
					safeText := strings.ToValidUTF8(string(rawBodyBytes), "")
					log.Printf("========== PLAINTEXT REQUEST PAYLOAD ==========\n%s\n===============================================\n\n", safeText)

			    } else {
					log.Printf("[DPI-REQ-INFO] Ignoring binary request body with type: %s\n", contentType)
				}
			}
			return req, nil
		})
    
	////
	// Mesin DPI (Response Inspection)
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
			
			} else if strings.HasPrefix(contentType, "text/") || strings.HasPrefix(contentType, "application/json") ||
                      strings.HasPrefix(contentType, "application/xml") || strings.HasPrefix(contentType, "application/x-www-form-urlencoded") {
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
