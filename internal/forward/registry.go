package forward

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"sync"
	"time"
)

// Well-known transport names.
const (
	ProtoHTTP1 = "http1" // strictly HTTP/1.1 to upstream
	ProtoAuto  = "auto"  // ALPN, allow h2 over TLS when available
	// ProtoH2C = "h2c"   // recommend registering lazily in another file if needed
)

// Options tunes the default transports.
type Options struct {
	// Dial/keepalive
	DialTimeout   time.Duration
	DialKeepAlive time.Duration

	// Pool sizing
	MaxIdleConns        int
	MaxIdleConnsPerHost int
	IdleConnTimeout     time.Duration
	MaxConnsPerHost     int // 0 = unlimited

	// Timeouts
	TLSHandshakeTimeout   time.Duration
	ExpectContinueTimeout time.Duration
	ResponseHeaderTimeout time.Duration // optional, 0 to disable

	// TLS knobs for defaults (cluster-specific/mTLS should register their own RT)
	InsecureSkipVerify bool
	RootCAs            *x509.CertPool
}

// DefaultOptions mirrors battle-tested proxy-ish settings.
func DefaultOptions() Options {
	return Options{
		DialTimeout:           5 * time.Second,
		DialKeepAlive:         60 * time.Second,
		MaxIdleConns:          512,
		MaxIdleConnsPerHost:   128,
		IdleConnTimeout:       90 * time.Second,
		MaxConnsPerHost:       0,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 0,
		InsecureSkipVerify:    false,
		RootCAs:               nil,
	}
}

// Factory returns a RoundTripper by name.
type Factory interface {
	Get(name string) http.RoundTripper
	Register(name string, rt http.RoundTripper)
	CloseIdle()
}

// Registry is a threadsafe map of named RoundTrippers.
type Registry struct {
	mu    sync.RWMutex
	store map[string]http.RoundTripper
	opts  Options
}

// NewDefaultRegistry builds a registry with DefaultOptions and pre-registers http1/auto.
func NewDefaultRegistry() *Registry { return NewRegistry(DefaultOptions()) }

// NewRegistry builds a registry with given options and pre-registers http1/auto.
func NewRegistry(opts Options) *Registry {
	r := &Registry{
		store: make(map[string]http.RoundTripper),
		opts:  opts,
	}
	r.store[ProtoHTTP1] = r.newHTTP1()
	r.store[ProtoAuto] = r.newAuto()
	// h2c/h3: register in your bootstrapping code when needed, e.g.:
	//   r.Register(ProtoH2C, newH2C(opts))
	return r
}

func (r *Registry) Get(name string) http.RoundTripper {
	r.mu.RLock()
	rt, ok := r.store[name]
	r.mu.RUnlock()
	if ok && rt != nil {
		return rt
	}
	// fallback to http1
	r.mu.RLock()
	fb := r.store[ProtoHTTP1]
	r.mu.RUnlock()
	return fb
}

func (r *Registry) Register(name string, rt http.RoundTripper) {
	if name == "" || rt == nil {
		return
	}
	r.mu.Lock()
	r.store[name] = rt
	r.mu.Unlock()
}

// CloseIdle calls CloseIdleConnections on all http.Transport in the registry.
func (r *Registry) CloseIdle() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, rt := range r.store {
		if t, ok := rt.(*http.Transport); ok {
			t.CloseIdleConnections()
		}
	}
}

// --- builders ---

func (r *Registry) newHTTP1() http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   r.opts.DialTimeout,
		KeepAlive: r.opts.DialKeepAlive,
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     false,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: r.opts.InsecureSkipVerify, RootCAs: r.opts.RootCAs, NextProtos: []string{"http/1.1"}},
		MaxIdleConns:          r.opts.MaxIdleConns,
		MaxIdleConnsPerHost:   r.opts.MaxIdleConnsPerHost,
		IdleConnTimeout:       r.opts.IdleConnTimeout,
		MaxConnsPerHost:       r.opts.MaxConnsPerHost,
		TLSHandshakeTimeout:   r.opts.TLSHandshakeTimeout,
		ExpectContinueTimeout: r.opts.ExpectContinueTimeout,
	}
	if r.opts.ResponseHeaderTimeout > 0 {
		tr.ResponseHeaderTimeout = r.opts.ResponseHeaderTimeout
	}
	return tr
}

func (r *Registry) newAuto() http.RoundTripper {
	dialer := &net.Dialer{
		Timeout:   r.opts.DialTimeout,
		KeepAlive: r.opts.DialKeepAlive,
	}
	tr := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true, // ALPN to h2 when possible; no h2c
		MaxIdleConns:          r.opts.MaxIdleConns,
		MaxIdleConnsPerHost:   r.opts.MaxIdleConnsPerHost,
		IdleConnTimeout:       r.opts.IdleConnTimeout,
		MaxConnsPerHost:       r.opts.MaxConnsPerHost,
		TLSHandshakeTimeout:   r.opts.TLSHandshakeTimeout,
		ExpectContinueTimeout: r.opts.ExpectContinueTimeout,
	}
	if r.opts.ResponseHeaderTimeout > 0 {
		tr.ResponseHeaderTimeout = r.opts.ResponseHeaderTimeout
	}
	return tr
}
