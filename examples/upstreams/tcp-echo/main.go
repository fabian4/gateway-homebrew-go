package main

import (
	"errors"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"
)

func main() {
	addr := getenv("TCP_ECHO_ADDR", ":9002")
	idleSec := getenvInt("TCP_ECHO_IDLE", 300) // idle timeout in seconds
	greeting := os.Getenv("TCP_ECHO_GREETING")

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	log.Printf("tcp-echo listening on %s (idle=%ds)", addr, idleSec)

	// graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	var wg sync.WaitGroup
	go func() {
		<-stop
		log.Println("shutting down listener...")
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// listener closed -> exit loop
			if errors.Is(err, net.ErrClosed) {
				break
			}
			// timeout-like errors (rare on Accept unless deadlines are used)
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				log.Printf("accept timeout: %v", err)
				continue
			}
			// other transient errors: back off briefly and continue
			log.Printf("accept error: %v (retrying)", err)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		wg.Add(1)
		go handleConn(conn, time.Duration(idleSec)*time.Second, greeting, &wg)
	}

	wg.Wait()
	log.Println("bye.")
}

func handleConn(c net.Conn, idle time.Duration, greeting string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func() { _ = c.Close() }()

	remote := c.RemoteAddr().String()
	log.Printf("conn %s opened", remote)
	defer log.Printf("conn %s closed", remote)

	_ = c.SetDeadline(time.Now().Add(idle))
	if greeting != "" {
		_, _ = c.Write([]byte(greeting + "\n"))
	}

	buf := make([]byte, 32*1024)
	for {
		n, err := c.Read(buf)
		if n > 0 {
			if _, werr := c.Write(buf[:n]); werr != nil {
				return
			}
			_ = c.SetDeadline(time.Now().Add(idle))
		}
		if err != nil {
			return
		}
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
