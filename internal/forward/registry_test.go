package forward

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"testing"
	"time"
)

func TestDefaultOptions(t *testing.T) {
	opts := DefaultOptions()

	if opts.DialTimeout != 5*time.Second {
		t.Errorf("DialTimeout: got %v, want %v", opts.DialTimeout, 5*time.Second)
	}
	if opts.DialKeepAlive != 60*time.Second {
		t.Errorf("DialKeepAlive: got %v, want %v", opts.DialKeepAlive, 60*time.Second)
	}
	if opts.MaxIdleConns != 512 {
		t.Errorf("MaxIdleConns: got %d, want %d", opts.MaxIdleConns, 512)
	}
	if opts.MaxIdleConnsPerHost != 128 {
		t.Errorf("MaxIdleConnsPerHost: got %d, want %d", opts.MaxIdleConnsPerHost, 128)
	}
	if opts.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout: got %v, want %v", opts.IdleConnTimeout, 90*time.Second)
	}
	if opts.MaxConnsPerHost != 0 {
		t.Errorf("MaxConnsPerHost: got %d, want %d", opts.MaxConnsPerHost, 0)
	}
	if opts.TLSHandshakeTimeout != 5*time.Second {
		t.Errorf("TLSHandshakeTimeout: got %v, want %v", opts.TLSHandshakeTimeout, 5*time.Second)
	}
	if opts.ExpectContinueTimeout != 1*time.Second {
		t.Errorf("ExpectContinueTimeout: got %v, want %v", opts.ExpectContinueTimeout, 1*time.Second)
	}
	if opts.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be false by default")
	}
}

func TestNewDefaultRegistry(t *testing.T) {
	reg := NewDefaultRegistry()

	if reg == nil {
		t.Fatal("NewDefaultRegistry returned nil")
	}
	if reg.store == nil {
		t.Fatal("registry store is nil")
	}

	// Check pre-registered transports
	if _, ok := reg.store[ProtoHTTP1]; !ok {
		t.Error("http1 transport not pre-registered")
	}
	if _, ok := reg.store[ProtoAuto]; !ok {
		t.Error("auto transport not pre-registered")
	}
}

func TestNewRegistry(t *testing.T) {
	customOpts := Options{
		DialTimeout:           10 * time.Second,
		MaxIdleConns:          100,
		InsecureSkipVerify:    true,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	reg := NewRegistry(customOpts)

	if reg == nil {
		t.Fatal("NewRegistry returned nil")
	}
	if reg.opts.DialTimeout != 10*time.Second {
		t.Errorf("opts not preserved: got %v, want %v", reg.opts.DialTimeout, 10*time.Second)
	}
}

func TestRegistry_Get(t *testing.T) {
	reg := NewDefaultRegistry()

	t.Run("existing transport", func(t *testing.T) {
		rt := reg.Get(ProtoHTTP1)
		if rt == nil {
			t.Fatal("Get(http1) returned nil")
		}
		if _, ok := rt.(*http.Transport); !ok {
			t.Error("expected *http.Transport")
		}
	})

	t.Run("non-existing falls back to http1", func(t *testing.T) {
		rt := reg.Get("non-existent")
		if rt == nil {
			t.Fatal("Get(non-existent) returned nil, expected fallback")
		}
		// Should be same as http1
		http1 := reg.Get(ProtoHTTP1)
		if rt != http1 {
			t.Error("expected fallback to http1 transport")
		}
	})

	t.Run("auto transport", func(t *testing.T) {
		rt := reg.Get(ProtoAuto)
		if rt == nil {
			t.Fatal("Get(auto) returned nil")
		}
	})
}

func TestRegistry_Register(t *testing.T) {
	reg := NewDefaultRegistry()

	t.Run("successful registration", func(t *testing.T) {
		customRT := &http.Transport{}
		reg.Register("custom", customRT)

		rt := reg.Get("custom")
		if rt != customRT {
			t.Error("registered transport not returned by Get")
		}
	})

	t.Run("ignore empty name", func(t *testing.T) {
		customRT := &http.Transport{}
		reg.Register("", customRT)

		// Should not panic or error
	})

	t.Run("ignore nil transport", func(t *testing.T) {
		reg.Register("nil-test", nil)

		// Should not panic or error
	})
}

func TestRegistry_CloseIdle(t *testing.T) {
	reg := NewDefaultRegistry()

	// Register a custom transport
	customRT := &http.Transport{}
	reg.Register("custom", customRT)

	// Should not panic
	reg.CloseIdle()
}

func TestRegistry_HTTP1Transport(t *testing.T) {
	opts := Options{
		DialTimeout:           3 * time.Second,
		DialKeepAlive:         30 * time.Second,
		MaxIdleConns:          50,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       60 * time.Second,
		TLSHandshakeTimeout:   3 * time.Second,
		ExpectContinueTimeout: 500 * time.Millisecond,
		InsecureSkipVerify:    true,
	}

	reg := NewRegistry(opts)
	rt := reg.Get(ProtoHTTP1)

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if tr.MaxIdleConns != 50 {
		t.Errorf("MaxIdleConns: got %d, want 50", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost != 10 {
		t.Errorf("MaxIdleConnsPerHost: got %d, want 10", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 60*time.Second {
		t.Errorf("IdleConnTimeout: got %v, want %v", tr.IdleConnTimeout, 60*time.Second)
	}
	if tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be false for http1")
	}
	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
}

func TestRegistry_AutoTransport(t *testing.T) {
	reg := NewDefaultRegistry()
	rt := reg.Get(ProtoAuto)

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if !tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true for auto")
	}
}

func TestRegistry_WithRootCAs(t *testing.T) {
	pool := x509.NewCertPool()
	opts := Options{
		RootCAs: pool,
	}

	reg := NewRegistry(opts)
	rt := reg.Get(ProtoHTTP1)

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if tr.TLSClientConfig.RootCAs != pool {
		t.Error("RootCAs not set correctly")
	}
}

func TestRegistry_WithResponseHeaderTimeout(t *testing.T) {
	opts := Options{
		ResponseHeaderTimeout: 10 * time.Second,
	}

	reg := NewRegistry(opts)
	rt := reg.Get(ProtoHTTP1)

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if tr.ResponseHeaderTimeout != 10*time.Second {
		t.Errorf("ResponseHeaderTimeout: got %v, want %v", tr.ResponseHeaderTimeout, 10*time.Second)
	}
}

func TestRegistry_RegisterCustom(t *testing.T) {
	reg := NewDefaultRegistry()

	// Register custom transport with insecure skip verify
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	reg.RegisterCustom("custom-tls", tlsConfig, ProtoHTTP1)

	rt := reg.Get("custom-tls")
	if rt == nil {
		t.Fatal("Get(custom-tls) returned nil")
	}

	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}

	if !tr.TLSClientConfig.InsecureSkipVerify {
		t.Error("InsecureSkipVerify should be true")
	}
	if tr.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be false for http1")
	}

	// Register custom transport with auto proto (h2)
	reg.RegisterCustom("custom-h2", nil, ProtoAuto)
	rt2 := reg.Get("custom-h2")
	tr2, ok := rt2.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if !tr2.ForceAttemptHTTP2 {
		t.Error("ForceAttemptHTTP2 should be true for auto")
	}
}

func TestRegistry_CustomOptions(t *testing.T) {
	opts := Options{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     5 * time.Minute,
	}
	reg := NewRegistry(opts)

	// Check http1
	rt := reg.Get(ProtoHTTP1)
	tr, ok := rt.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.MaxIdleConns != 1000 {
		t.Errorf("MaxIdleConns: got %d, want 1000", tr.MaxIdleConns)
	}
	if tr.MaxIdleConnsPerHost != 100 {
		t.Errorf("MaxIdleConnsPerHost: got %d, want 100", tr.MaxIdleConnsPerHost)
	}
	if tr.IdleConnTimeout != 5*time.Minute {
		t.Errorf("IdleConnTimeout: got %v, want %v", tr.IdleConnTimeout, 5*time.Minute)
	}

	// Check RegisterCustom inherits options
	reg.RegisterCustom("custom", nil, ProtoHTTP1)
	rtCustom := reg.Get("custom")
	trCustom, ok := rtCustom.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if trCustom.MaxIdleConns != 1000 {
		t.Errorf("custom MaxIdleConns: got %d, want 1000", trCustom.MaxIdleConns)
	}
}
