package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "github.com/fabian4/gateway-homebrew-go/internal/config"
	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/handler"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
	"github.com/fabian4/gateway-homebrew-go/internal/version"
)

func main() {
	configPath := flag.String("config", "./cmd/config.yaml", "path to YAML config")
	flag.Parse()

	c, err := cfg.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	rt := router.New(c.Routes)
	reg := fwd.NewDefaultRegistry()
	gw := handler.NewGateway(rt, c.Services, reg, c.Timeouts.Upstream, os.Stdout)

	readTimeout := 15 * time.Second
	if c.Timeouts.Read > 0 {
		readTimeout = c.Timeouts.Read
	}
	writeTimeout := 30 * time.Second
	if c.Timeouts.Write > 0 {
		writeTimeout = c.Timeouts.Write
	}

	srv := &http.Server{
		Addr:              c.Listen,
		Handler:           gw,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       60 * time.Second,
	}
	log.Printf("gateway-homebrew-go %s listening on %s (routes=%d services=%d)",
		version.Value, c.Listen, len(c.Routes), len(c.Services))

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
