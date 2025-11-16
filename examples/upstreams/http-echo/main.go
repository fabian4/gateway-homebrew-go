package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

func main() {
	addr := getEnv("ECHO_ADDR", ":9001")
	mux := http.NewServeMux()

	// /sleep/{ms} : sleep for N milliseconds then echo
	mux.HandleFunc("/sleep/", func(w http.ResponseWriter, r *http.Request) {
		msStr := strings.TrimPrefix(r.URL.Path, "/sleep/")
		ms, err := strconv.Atoi(msStr)
		if err != nil || ms < 0 {
			http.Error(w, "bad sleep value", http.StatusBadRequest)
			return
		}
		time.Sleep(time.Duration(ms) * time.Millisecond)
		echo(w, r)
	})

	// /status/{code} : respond with given status code and echo headers/body
	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		codeStr := strings.TrimPrefix(r.URL.Path, "/status/")
		code, err := strconv.Atoi(codeStr)
		if err != nil || code < 100 || code > 999 {
			http.Error(w, "bad status code", http.StatusBadRequest)
			return
		}
		w.WriteHeader(code)
		echoBodyOnly(w, r)
	})

	// default: echo everything
	mux.HandleFunc("/", echo)

	log.Printf("http-echo listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, withCommonHeaders(mux)))
}

func echo(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB cap
	_ = r.Body.Close()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	_, _ = fmt.Fprintf(w, "time:   %s\n", time.Now().Format(time.RFC3339Nano))
	_, _ = fmt.Fprintf(w, "remote: %s\n", r.RemoteAddr)
	_, _ = fmt.Fprintf(w, "proto:  %s\n", r.Proto)
	_, _ = fmt.Fprintf(w, "method: %s\n", r.Method)
	_, _ = fmt.Fprintf(w, "host:   %s\n", r.Host)
	_, _ = fmt.Fprintf(w, "url:    %s\n", r.URL.String())
	_, _ = fmt.Fprintf(w, "---- headers ----\n")
	keys := make([]string, 0, len(r.Header))
	for k := range r.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		_, _ = fmt.Fprintf(w, "%s: %s\n", k, strings.Join(r.Header[k], ", "))
	}
	_, _ = fmt.Fprintf(w, "---- body (%d bytes) ----\n", len(body))
	const maxPrint = 8192
	if len(body) > maxPrint {
		_, err := w.Write(body[:maxPrint])
		if err != nil {
			return
		}
		_, _ = fmt.Fprintf(w, "\n...[truncated %d bytes]\n", len(body)-maxPrint)
	} else {
		_, err := w.Write(body)
		if err != nil {
			return
		}
	}
}

func echoBodyOnly(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	_ = r.Body.Close()
	if ct := r.Header.Get("Content-Type"); ct != "" {
		w.Header().Set("Content-Type", ct)
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
	_, err := w.Write(body)
	if err != nil {
		return
	}
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Echo-Server", "go-http-echo")
		next.ServeHTTP(w, r)
	})
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
