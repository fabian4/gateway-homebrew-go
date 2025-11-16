package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "github.com/fabian4/gateway-homebrew-go/internal/config"
	"github.com/fabian4/gateway-homebrew-go/internal/proxy"
	"github.com/fabian4/gateway-homebrew-go/internal/version"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to YAML config")
	flag.Parse()

	c, err := cfg.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	h := proxy.NewHTTP1Proxy(c.Upstream)

	srv := &http.Server{
		Addr:              c.Listen,
		Handler:           h,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("gateway-homebrew-go %s listening on %s → %s", version.Value, c.Listen, c.Upstream)

	// start server
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	// graceful shutdown on SIGINT/SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}
	log.Println("bye.")
}
