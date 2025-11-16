package proxy

import (
	"crypto/tls"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// HTTP1Proxy is a minimal, hand-rolled HTTP/1.1 reverse proxy (no httputil.ReverseProxy).
type HTTP1Proxy struct {
	Upstream             *url.URL
	Transport            *http.Transport
	PreserveIncomingHost bool
}

// compile-time interface check
var _ http.Handler = (*HTTP1Proxy)(nil)

func NewHTTP1Proxy(upstream *url.URL) *HTTP1Proxy {
	tr := &http.Transport{
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
	return &HTTP1Proxy{Upstream: upstream, Transport: tr}
}

func (p *HTTP1Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	up := new(url.URL)
	*up = *p.Upstream
	up.Path = joinSlash(p.Upstream.Path, r.URL.Path)
	up.RawQuery = r.URL.RawQuery
	up.Fragment = ""

	hdr := cloneHeader(r.Header)
	dropHopByHop(hdr)
	addXFF(hdr, r.RemoteAddr)
	setXFProto(hdr, r)
	setXFHost(hdr, r.Host)

	reqUp, err := http.NewRequestWithContext(r.Context(), r.Method, up.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	reqUp.Header = hdr
	if p.PreserveIncomingHost {
		reqUp.Host = r.Host
	} else {
		reqUp.Host = p.Upstream.Host
	}

	resUp, err := p.Transport.RoundTrip(reqUp)
	if err != nil {
		log.Printf("upstream error: %v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resUp.Body)

	dropHopByHop(resUp.Header)
	copyHeaders(w.Header(), resUp.Header)
	w.WriteHeader(resUp.StatusCode)
	_, _ = io.Copy(w, resUp.Body)
}

// --- helpers ---

func cloneHeader(h http.Header) http.Header {
	out := make(http.Header, len(h))
	for k, vv := range h {
		cc := make([]string, len(vv))
		copy(cc, vv)
		out[k] = cc
	}
	return out
}

func copyHeaders(dst, src http.Header) {
	for k, vv := range src {
		dst.Del(k)
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func joinSlash(a, b string) string {
	as := strings.HasSuffix(a, "/")
	bs := strings.HasPrefix(b, "/")
	switch {
	case as && bs:
		return a + b[1:]
	case !as && !bs:
		return a + "/" + b
	default:
		return a + b
	}
}

var hopByHop = map[string]struct{}{
	"Connection":          {},
	"Proxy-Connection":    {},
	"Keep-Alive":          {},
	"Proxy-Authenticate":  {},
	"Proxy-Authorization": {},
	"TE":                  {},
	"Trailer":             {},
	"Transfer-Encoding":   {},
	"Upgrade":             {},
}

func dropHopByHop(h http.Header) {
	for _, f := range h.Values("Connection") {
		for _, k := range strings.Split(f, ",") {
			k = textproto.TrimString(k)
			if k != "" {
				h.Del(k)
			}
		}
	}
	for k := range hopByHop {
		h.Del(k)
	}
}

func addXFF(h http.Header, remoteAddr string) {
	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil || ip == "" {
		return
	}
	const key = "X-Forwarded-For"
	if prior := h.Get(key); prior != "" {
		h.Set(key, prior+", "+ip)
	} else {
		h.Set(key, ip)
	}
}

func setXFHost(h http.Header, host string) {
	h.Set("X-Forwarded-Host", host)
}

func setXFProto(h http.Header, r *http.Request) {
	if r.TLS != nil {
		h.Set("X-Forwarded-Proto", "https")
	} else {
		h.Set("X-Forwarded-Proto", "http")
	}
}
