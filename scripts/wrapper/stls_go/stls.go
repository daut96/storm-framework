package stls

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include <dlfcn.h>

// Definisi pointer fungsi ke library Rust
typedef char* (*storm_request_func)(const char*, const char*, const char*, const unsigned char*, size_t);
typedef void (*storm_free_string_func)(char*);

static void* stls_handle = NULL;
static storm_request_func req_func = NULL;
static storm_free_string_func free_func = NULL;

// Fungsi C untuk meload .so secara dinamis saat runtime (seperti ctypes di Python)
static const char* load_stls_library(const char* path) {
    stls_handle = dlopen(path, RTLD_LAZY);
    if (!stls_handle) {
        return dlerror(); // Kembalikan error jika gagal load
    }
    
    req_func = (storm_request_func)dlsym(stls_handle, "storm_request");
    free_func = (storm_free_string_func)dlsym(stls_handle, "storm_free_string");
    
    if (!req_func || !free_func) {
        return dlerror();
    }
    return NULL; // NULL berarti sukses
}

// Wrapper C untuk eksekusi
static char* do_request(const char* url, const char* method, const char* headers, const unsigned char* body, size_t body_len) {
    if (!req_func) return "ERROR: STLS library has not been loaded";
    return req_func(url, method, headers, body, body_len);
}

// Wrapper C untuk free memory
static void do_free(char* ptr) {
    if (free_func && ptr) {
        free_func(ptr);
    }
}
*/
import "C"
import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"unsafe"
)

// =====================================================================
// MANUAL INTERNAL LOGGER
// =====================================================================
var logger = log.New(os.Stdout, "[STLS-CORE] ", log.Ldate|log.Ltime|log.Lmsgprefix)

func printLog(level, msg string) {
	logger.Printf("[%s] %s\n", level, msg)
}

// =====================================================================
// DYNAMIC BINARY LOCATOR
// =====================================================================

// findSTLS melacak libstls.so secara instan berdasarkan arsitektur fix Storm Framework.
// Algoritma: Traversal ke atas (Upward Lookup) tanpa iterasi folder anak.
func findSTLS() (string, error) {
	// Target path relatif yang sudah kita ketahui letaknya
	targetRelPath := filepath.Join("external", "source", "out", "core", "tls", "libstls.so")

	// 1. Cek via Environment Variable (Best Practice Produksi)
	// Jika user/sistem menetapkan STORM_ROOT, gunakan ini sebagai prioritas absolut (O(1)).
	if envRoot := os.Getenv("STORM_ROOT"); envRoot != "" {
		fullPath := filepath.Join(envRoot, targetRelPath)
		if _, err := os.Stat(fullPath); err == nil {
			absPath, _ := filepath.Abs(fullPath)
			return absPath, nil
		}
	}

	// 2. Traversal Ke Atas (Smart Lookup)
	// Mencari letak root 'storm-framework' dengan naik direktori satu per satu.
	currentDir, err := os.Getwd()
	if err != nil {
		return "", errors.New("failed to get current executable directory")
	}

	for {
		// Gabungkan direktori saat ini dengan letak relatif STLS
		checkPath := filepath.Join(currentDir, targetRelPath)
		
		// Cek apakah file benar-benar ada dan bukan direktori
		if info, err := os.Stat(checkPath); err == nil && !info.IsDir() {
			absPath, _ := filepath.Abs(checkPath)
			return absPath, nil
		}

		// Jika tidak ada, naik satu tingkat ke parent directory (cd ..)
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			// Jika parent dir sama dengan current dir, berarti kita sudah mentok di root OS (misal: /)
			break
		}
		currentDir = parentDir
	}

	return "", errors.New("failed to resolve libstls.so")
}


// =====================================================================
// MAIN WRAPPER LOGIC
// =====================================================================

type Response struct {
	Text string
}

func Request(method, url string, headers map[string]string, body []byte) (*Response, error) {
	// 1. Marshalling JSON Headers
	if headers == nil {
		headers = make(map[string]string)
	}
	headersJSON, _ := json.Marshal(headers)

	// 2. Konversi ke C String (CGO)
	cMethod := C.CString(strings.ToUpper(method))
	cURL := C.CString(url)
	cHeaders := C.CString(string(headersJSON))

	// PENTING 1: Bebaskan memori yang dialokasikan Go untuk C
	defer C.free(unsafe.Pointer(cMethod))
	defer C.free(unsafe.Pointer(cURL))
	defer C.free(unsafe.Pointer(cHeaders))

	// 3. Konversi Body Bytes ke C Pointer
	var cBody *C.uchar
	var cBodyLen C.size_t
	if len(body) > 0 {
		cBody = (*C.uchar)(unsafe.Pointer(&body[0]))
		cBodyLen = C.size_t(len(body))
	}

	// 4. Panggil Mesin Rust
	cResultPtr := C.do_request(cURL, cMethod, cHeaders, cBody, cBodyLen)
	
	if cResultPtr == nil {
		printLog("WARN", "STLS returned a null pointer")
		return nil, errors.New("STLS returned a null pointer")
	}

	// PENTING 2: Bebaskan memori yang dialokasikan Rust
	defer C.do_free(cResultPtr)

	// 5. Ekstrak Hasil
	rawResult := C.GoString(cResultPtr)
	if strings.HasPrefix(rawResult, "ERROR:") {
		printLog("ERROR", fmt.Sprintf("Internal Error: %s", rawResult))
		return nil, errors.New(rawResult)
	}

	return &Response{Text: rawResult}, nil
}

// =====================================================================
// SYNTACTIC SUGARS (HTTP METHODS)
// =====================================================================

// Get mengeksekusi HTTP GET request (tanpa body)
func Get(url string, headers map[string]string) (*Response, error) {
	return Request("GET", url, headers, nil)
}

// Post mengeksekusi HTTP POST request (dengan body)
func Post(url string, headers map[string]string, body []byte) (*Response, error) {
	return Request("POST", url, headers, body)
}

// Put mengeksekusi HTTP PUT request untuk update/replace data (dengan body)
func Put(url string, headers map[string]string, body []byte) (*Response, error) {
	return Request("PUT", url, headers, body)
}

// Delete mengeksekusi HTTP DELETE request (umumnya tanpa body)
func Delete(url string, headers map[string]string) (*Response, error) {
	return Request("DELETE", url, headers, nil)
}

// Patch mengeksekusi HTTP PATCH request untuk modifikasi parsial (dengan body)
func Patch(url string, headers map[string]string, body []byte) (*Response, error) {
	return Request("PATCH", url, headers, body)
}

// Head mengeksekusi HTTP HEAD request untuk mengambil header saja (tanpa body)
func Head(url string, headers map[string]string) (*Response, error) {
	return Request("HEAD", url, headers, nil)
}

// Options mengeksekusi HTTP OPTIONS request untuk mengecek kapabilitas server (tanpa body)
func Options(url string, headers map[string]string) (*Response, error) {
	return Request("OPTIONS", url, headers, nil)
}
