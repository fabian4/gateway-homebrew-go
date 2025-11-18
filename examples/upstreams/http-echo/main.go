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
	addr := getenv("ECHO_ADDR", ":9001")
	id := getenv("ECHO_ID", "u")
	greeting := os.Getenv("ECHO_GREETING")
	latMs := getenvInt("ECHO_LATENCY_MS", 0)

	mux := http.NewServeMux()

	// liveness/readiness
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// status endpoints: support both /status/<code> and /api/status/<code>
	mux.HandleFunc("/status/", statusWithPrefix(id, "/status/"))
	mux.HandleFunc("/api/status/", statusWithPrefix(id, "/api/status/"))

	// sleep endpoints: support both /sleep/<ms> and /api/sleep/<ms>
	mux.HandleFunc("/sleep/", sleepWithPrefix(id, "/sleep/"))
	mux.HandleFunc("/api/sleep/", sleepWithPrefix(id, "/api/sleep/"))

	// default echo
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		setCommonHeaders(w, r, id, greeting)
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

func statusWithPrefix(id, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCommonHeaders(w, r, id, "")
		codeStr := strings.TrimPrefix(r.URL.Path, prefix)
		code, err := strconv.Atoi(codeStr)
		if err != nil || code < 100 || code > 599 {
			http.Error(w, "bad status", http.StatusBadRequest)
			return
		}
		w.WriteHeader(code)
	}
}

func sleepWithPrefix(id, prefix string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCommonHeaders(w, r, id, "")
		msStr := strings.TrimPrefix(r.URL.Path, prefix)
		ms, err := strconv.Atoi(msStr)
		if err != nil || ms < 0 || ms > 60000 {
			http.Error(w, "bad sleep", http.StatusBadRequest)
			return
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}
}

func setCommonHeaders(w http.ResponseWriter, r *http.Request, id, greeting string) {
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
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}
