package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	cfg "github.com/fabian4/gateway-homebrew-go/internal/config"
	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/handler"
	"github.com/fabian4/gateway-homebrew-go/internal/lb"
	"github.com/fabian4/gateway-homebrew-go/internal/metrics"
	"github.com/fabian4/gateway-homebrew-go/internal/model"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
	"github.com/fabian4/gateway-homebrew-go/internal/version"
)

func updateRegistry(reg *fwd.Registry, services map[string]model.Service) {
	for _, svc := range services {
		if svc.TLS != nil {
			tlsConf := &tls.Config{
				InsecureSkipVerify: svc.TLS.InsecureSkipVerify,
			}
			if svc.TLS.CAFile != "" {
				caCert, err := os.ReadFile(svc.TLS.CAFile)
				if err != nil {
					log.Printf("service %s: read ca_file: %v", svc.Name, err)
					continue
				}
				caPool := x509.NewCertPool()
				if ok := caPool.AppendCertsFromPEM(caCert); !ok {
					log.Printf("service %s: failed to parse ca_file", svc.Name)
					continue
				}
				tlsConf.RootCAs = caPool
			}
			if svc.TLS.CertFile != "" && svc.TLS.KeyFile != "" {
				cert, err := tls.LoadX509KeyPair(svc.TLS.CertFile, svc.TLS.KeyFile)
				if err != nil {
					log.Printf("service %s: load cert/key: %v", svc.Name, err)
					continue
				}
				tlsConf.Certificates = []tls.Certificate{cert}
			}
			reg.RegisterCustom(svc.Name, tlsConf, svc.Proto)
		} else {
			// Use shared transport
			reg.Register(svc.Name, reg.Get(svc.Proto))
		}
	}
}

func watchConfig(path string, interval time.Duration, onChange func(*cfg.Config)) {
	var lastMod time.Time
	if info, err := os.Stat(path); err == nil {
		lastMod = info.ModTime()
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		if !info.ModTime().Equal(lastMod) {
			lastMod = info.ModTime()
			log.Printf("config change detected, reloading...")

			newCfg, err := cfg.Load(path)
			if err != nil {
				log.Printf("config reload failed (validation error): %v", err)
				continue
			}

			onChange(newCfg)
			log.Printf("config reloaded successfully")
		}
	}
}

func main() {
	configPath := flag.String("config", "./cmd/config.yaml", "path to YAML config")
	flag.Parse()

	c, err := cfg.Load(*configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	// Metrics
	m := metrics.NewRegistry()
	if c.Metrics.Address != "" {
		go func() {
			mux := http.NewServeMux()
			mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
				m.WritePrometheus(w)
			})
			log.Printf("metrics listening on %s/metrics", c.Metrics.Address)
			if err := http.ListenAndServe(c.Metrics.Address, mux); err != nil {
				log.Printf("metrics server error: %v", err)
			}
		}()
	}

	rt := router.New(c.Routes)

	// Transport options
	fwdOpts := fwd.DefaultOptions()
	fwdOpts.MaxIdleConns = c.Transport.MaxIdleConns
	fwdOpts.MaxIdleConnsPerHost = c.Transport.MaxIdleConnsPerHost
	fwdOpts.IdleConnTimeout = c.Transport.IdleConnTimeout
	fwdOpts.DialTimeout = c.Transport.DialTimeout
	fwdOpts.DialKeepAlive = c.Transport.DialKeepAlive

	reg := fwd.NewRegistry(fwdOpts)

	// Register transports for each service
	updateRegistry(reg, c.Services)

	gw := handler.NewGateway(rt, c.Services, reg, c.Timeouts.Upstream, os.Stdout, c.AccessLog, m)

	if !c.Benchmark.Enabled {
		go watchConfig(*configPath, 5*time.Second, func(newC *cfg.Config) {
			updateRegistry(reg, newC.Services)
			rt := router.New(newC.Routes)
			gw.UpdateState(rt, newC.Services, newC.Timeouts.Upstream, newC.AccessLog)
		})
	}

	var httpServers []*http.Server
	var tcpListeners []net.Listener

	log.Printf("gateway-homebrew-go %s starting...", version.Value)
	if c.Benchmark.Enabled {
		log.Printf("benchmark mode enabled (background tasks disabled)")
	}

	for _, l := range c.Listeners {
		if l.Service != "" {
			// L4 TCP Proxy
			svc, ok := c.Services[l.Service]
			if !ok {
				log.Fatalf("listener %s: service %s not found", l.Name, l.Service)
			}
			balancer := lb.NewSmoothWRR(svc.Endpoints)
			proxy := handler.NewTCPProxy(balancer, c.Timeouts.TCPIdle, c.Timeouts.TCPConnection, m, l.Name, l.Service)

			ln, err := net.Listen("tcp", l.Address)
			if err != nil {
				log.Fatalf("listener %s: listen tcp %s: %v", l.Name, l.Address, err)
			}
			tcpListeners = append(tcpListeners, ln)

			log.Printf("L4 listener %s on %s forwarding to %s", l.Name, l.Address, l.Service)

			go func(ln net.Listener) {
				for {
					conn, err := ln.Accept()
					if err != nil {
						return
					}
					go proxy.Handle(conn)
				}
			}(ln)

		} else {
			// L7 HTTP Proxy
			srv := &http.Server{
				Addr:              l.Address,
				Handler:           gw,
				ReadTimeout:       c.Timeouts.Read,
				ReadHeaderTimeout: 10 * time.Second,
				WriteTimeout:      c.Timeouts.Write,
				IdleTimeout:       60 * time.Second,
			}

			if c.TLS.Enabled {
				tlsConfig := &tls.Config{
					MinVersion: tls.VersionTLS12,
					NextProtos: []string{"h2", "http/1.1"},
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

			httpServers = append(httpServers, srv)

			log.Printf("L7 listener %s on %s (routes=%d services=%d tls=%v)",
				l.Name, l.Address, len(c.Routes), len(c.Services), c.TLS.Enabled)

			go func(srv *http.Server) {
				if c.TLS.Enabled {
					if err := srv.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
						log.Fatalf("listen tls: %v", err)
					}
				} else {
					if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
						log.Fatalf("listen: %v", err)
					}
				}
			}(srv)
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, srv := range httpServers {
		_ = srv.Shutdown(shutdownCtx)
	}
	for _, ln := range tcpListeners {
		_ = ln.Close()
	}
}
