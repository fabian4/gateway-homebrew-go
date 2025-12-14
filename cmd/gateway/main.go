package main

import (
	"context"
	"crypto/tls"
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

	srv := &http.Server{
		Addr:              c.Listen,
		Handler:           gw,
		ReadTimeout:       c.Timeouts.Read,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      c.Timeouts.Write,
		IdleTimeout:       60 * time.Second,
	}

	if c.TLS.Enabled {
		tlsConfig := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		for _, cert := range c.TLS.Certificates {
			c, err := tls.LoadX509KeyPair(cert.CertFile, cert.KeyFile)
			if err != nil {
				log.Fatalf("load cert %s: %v", cert.CertFile, err)
			}
			tlsConfig.Certificates = append(tlsConfig.Certificates, c)
		}
		srv.TLSConfig = tlsConfig
	}

	log.Printf("gateway-homebrew-go %s listening on %s (routes=%d services=%d tls=%v)",
		version.Value, c.Listen, len(c.Routes), len(c.Services), c.TLS.Enabled)

	go func() {
		if c.TLS.Enabled {
			if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen tls: %v", err)
			}
		} else {
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("listen: %v", err)
			}
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
