package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	addr := getEnv("ECHO_ADDR", ":9001")
	id := getEnv("ECHO_ID", "u")
	greeting := os.Getenv("ECHO_GREETING")
	latMs := getEnvInt("ECHO_LATENCY_MS", 0)

	mux := http.NewServeMux()

	// liveness/readiness
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// configurable status: /status/418
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		codeStr := strings.TrimPrefix(r.URL.Path, "/status/")
		code, err := strconv.Atoi(codeStr)
		if err != nil || code < 100 || code > 599 {
			http.Error(w, "bad status", http.StatusBadRequest)
			return
		}
		w.WriteHeader(code)
	})

	// sleep handler: /sleep/250  -> 250ms
	mux.HandleFunc("/sleep/", func(w http.ResponseWriter, r *http.Request) {
		msStr := strings.TrimPrefix(r.URL.Path, "/sleep/")
		ms, err := strconv.Atoi(msStr)
		if err != nil || ms < 0 || ms > 60000 {
			http.Error(w, "bad sleep", http.StatusBadRequest)
			return
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	// default echo
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// upstream visibility headers for E2E assertions
		w.Header().Set("X-Upstream-ID", id)
		if v := r.Header.Get("Connection"); v != "" {
			w.Header().Set("X-Seen-Connection", v)
		} else {
			w.Header().Set("X-Seen-Connection", "<empty>")
		}
		if v := r.Header.Get("Upgrade"); v != "" {
			w.Header().Set("X-Seen-Upgrade", v)
		} else {
			w.Header().Set("X-Seen-Upgrade", "<empty>")
		}
		if v := r.Header.Get("X-Forwarded-Proto"); v != "" {
			w.Header().Set("X-Seen-XFP", v)
		}
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			w.Header().Set("X-Seen-XFF", v)
		}
		if greeting != "" {
			w.Header().Set("X-Greeting", greeting)
		}
		// optional baseline latency
		if latMs > 0 {
			time.Sleep(time.Duration(latMs) * time.Millisecond)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprintf(w, "echo %s %s\n", r.Method, r.URL.Path)
	})

	log.Printf("http-echo %s listening on %s", id, addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Fatal(srv.ListenAndServe())
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func getEnvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
