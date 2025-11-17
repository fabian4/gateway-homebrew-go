package forward

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

// Factory returns a RoundTripper by name.
type Factory interface {
	Get(name string) http.RoundTripper
}

type Registry struct {
	store map[string]http.RoundTripper
}

func NewRegistry() *Registry {
	r := &Registry{store: make(map[string]http.RoundTripper)}
	// Strict HTTP/1.1
	r.store["http1"] = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 60 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     false,
		TLSClientConfig:       &tls.Config{NextProtos: []string{"http/1.1"}},
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	// Auto (allow ALPN upgrade to HTTP/2 over TLS when available)
	//r.store["auto"] = &http.Transport{
	//    Proxy: http.ProxyFromEnvironment,
	//    DialContext: (&net.Dialer{
	//        Timeout:   5 * time.Second,
	//        KeepAlive: 60 * time.Second,
	//    }).DialContext,
	//    ForceAttemptHTTP2:   true,
	//    MaxIdleConns:        200,
	//    MaxIdleConnsPerHost: 100,
	//    IdleConnTimeout:     90 * time.Second,
	//    TLSHandshakeTimeout: 5 * time.Second,
	//    ExpectContinueTimeout: 1 * time.Second,
	//}
	// h2c (HTTP/2 cleartext, no TLS). Use only for trusted internal upstreams.
	//r.store["h2c"] = &http2.Transport{
	//    AllowHTTP: true,
	//    DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
	//        return net.DialTimeout(network, addr, 5*time.Second)
	//    },
	//}
	return r
}

func (r *Registry) Get(name string) http.RoundTripper {
	if v, ok := r.store[name]; ok {
		return v
	}
	return r.store["http1"]
}
