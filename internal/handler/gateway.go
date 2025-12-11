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
	"github.com/fabian4/gateway-homebrew-go/internal/lb"
	"github.com/fabian4/gateway-homebrew-go/internal/model"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
)

type Gateway struct {
	Routes     *router.Table
	Services   map[string]model.Service
	Transports fwd.Factory
	balancers  map[string]lb.Balancer
}

func NewGateway(rt *router.Table, svcs map[string]model.Service, f fwd.Factory) *Gateway {
	lbs := make(map[string]lb.Balancer)
	for name, svc := range svcs {
		lbs[name] = lb.NewSmoothWRR(svc.Endpoints)
	}
	return &Gateway{Routes: rt, Services: svcs, Transports: f, balancers: lbs}
}

var _ http.Handler = (*Gateway)(nil)

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	route := g.Routes.Match(r.Host, r.URL.Path)
	if route == nil {
		http.NotFound(w, r)
		return
	}
	svc, ok := g.Services[route.Service]
	if !ok || len(svc.Endpoints) == 0 {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	// minimal choice: first endpoint (TODO: plug LB)
	base := g.balancers[route.Service].Next()
	if base == nil {
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	tr := g.Transports.Get(svc.Proto)

	// upstream URL = base + path
	u := new(url.URL)
	*u = *base
	u.Path = joinSlash(base.Path, r.URL.Path)
	u.RawQuery = r.URL.RawQuery

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

	// Host policy
	switch {
	case route.HostRewrite != "":
		reqUp.Host = route.HostRewrite
	case route.PreserveHost:
		reqUp.Host = r.Host
	default:
		reqUp.Host = base.Host
	}

	resUp, err := tr.RoundTrip(reqUp)
	if err != nil {
		log.Printf("upstream error: %v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("error closing upstream body: %v", err)
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
