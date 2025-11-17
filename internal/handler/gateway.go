package handler

import (
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"

	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
)

type Gateway struct {
	Routes               *router.Table
	Transports           fwd.Factory
	PreserveIncomingHost bool
}

func NewGateway(rt *router.Table, f fwd.Factory) *Gateway {
	return &Gateway{Routes: rt, Transports: f}
}

var _ http.Handler = (*Gateway)(nil)

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route := g.Routes.Match(r.Host, r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}
	tr := g.Transports.Get(route.Proto)

	// Build upstream URL
	up := route.URL
	u := new(url.URL)
	*u = *up
	u.Path = joinSlash(up.Path, r.URL.Path)
	u.RawQuery = r.URL.RawQuery
	u.Fragment = ""

	// Prepare headers
	hdr := cloneHeader(r.Header)
	dropHopByHop(hdr)
	addXFF(hdr, r.RemoteAddr)
	setXFProto(hdr, r)
	setXFHost(hdr, r.Host)

	reqUp, err := http.NewRequestWithContext(r.Context(), r.Method, u.String(), r.Body)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	reqUp.Header = hdr
	if g.PreserveIncomingHost {
		reqUp.Host = r.Host
	} else {
		reqUp.Host = up.Host
	}

	resUp, err := tr.RoundTrip(reqUp)
	if err != nil {
		log.Printf("upstream error: %v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer resUp.Body.Close()

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
