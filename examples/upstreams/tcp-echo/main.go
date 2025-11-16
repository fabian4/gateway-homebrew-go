package main

import (
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
	addr := getEnv("TCP_ECHO_ADDR", ":9002")
	idleSec := getEnvInt("TCP_ECHO_IDLE", 300) // idle timeout in seconds
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
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("accept temp err: %v", err)
				time.Sleep(100 * time.Millisecond)
				continue
			}
			// most likely listener closed
			break
		}
		wg.Add(1)
		go handleConn(conn, time.Duration(idleSec)*time.Second, greeting, &wg)
	}

	wg.Wait()
	log.Println("bye.")
}

func handleConn(c net.Conn, idle time.Duration, greeting string, wg *sync.WaitGroup) {
	defer wg.Done()
	defer func(c net.Conn) {
		err := c.Close()
		if err != nil {

		}
	}(c)

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
			// client closed / timeout / network error
			return
		}
	}
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
