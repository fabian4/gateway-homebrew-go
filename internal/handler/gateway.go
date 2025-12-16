package handler

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"

	"sync"
	"time"

	"github.com/fabian4/gateway-homebrew-go/internal/config"
	fwd "github.com/fabian4/gateway-homebrew-go/internal/forward"
	"github.com/fabian4/gateway-homebrew-go/internal/lb"
	"github.com/fabian4/gateway-homebrew-go/internal/metrics"
	"github.com/fabian4/gateway-homebrew-go/internal/model"
	"github.com/fabian4/gateway-homebrew-go/internal/router"
)

type GatewayState struct {
	Routes          *router.Table
	Services        map[string]model.Service
	balancers       map[string]lb.Balancer
	UpstreamTimeout time.Duration
	AccessLogConfig config.AccessLogConfig
}

type Gateway struct {
	stateMu    sync.RWMutex
	state      *GatewayState
	Transports fwd.Factory
	AccessLog  io.Writer
	Metrics    *metrics.Registry
}

func NewGateway(rt *router.Table, svcs map[string]model.Service, f fwd.Factory, upstreamTimeout time.Duration, accessLog io.Writer, alc config.AccessLogConfig, m *metrics.Registry) *Gateway {
	lbs := make(map[string]lb.Balancer)
	for name, svc := range svcs {
		lbs[name] = lb.NewSmoothWRR(svc.Endpoints)
	}
	if accessLog == nil {
		accessLog = io.Discard
	}
	state := &GatewayState{
		Routes:          rt,
		Services:        svcs,
		balancers:       lbs,
		UpstreamTimeout: upstreamTimeout,
		AccessLogConfig: alc,
	}
	return &Gateway{state: state, Transports: f, AccessLog: accessLog, Metrics: m}
}

func (g *Gateway) UpdateState(rt *router.Table, svcs map[string]model.Service, upstreamTimeout time.Duration, alc config.AccessLogConfig) {
	lbs := make(map[string]lb.Balancer)
	for name, svc := range svcs {
		lbs[name] = lb.NewSmoothWRR(svc.Endpoints)
	}
	newState := &GatewayState{
		Routes:          rt,
		Services:        svcs,
		balancers:       lbs,
		UpstreamTimeout: upstreamTimeout,
		AccessLogConfig: alc,
	}
	g.stateMu.Lock()
	g.state = newState
	g.stateMu.Unlock()
}

var _ http.Handler = (*Gateway)(nil)

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	g.stateMu.RLock()
	state := g.state
	g.stateMu.RUnlock()

	start := time.Now()
	lw := &loggingResponseWriter{ResponseWriter: w}
	var serviceName, upstreamAddr, routeName string
	defer func() {
		status := lw.statusCode
		if status == 0 {
			status = http.StatusOK
		}
		duration := time.Since(start)

		// Sampling
		if state.AccessLogConfig.Sampling < 1.0 && rand.Float64() > state.AccessLogConfig.Sampling {
			// skip logging
		} else {
			entry := AccessLog{
				Time:         start,
				Method:       r.Method,
				Path:         r.URL.Path,
				Protocol:     r.Proto,
				Status:       status,
				Duration:     duration.Milliseconds(),
				RemoteIP:     r.RemoteAddr,
				UserAgent:    r.UserAgent(),
				Referer:      r.Referer(),
				Service:      serviceName,
				Upstream:     upstreamAddr,
				BytesWritten: lw.bytes,
			}

			var logOutput any = entry
			if len(state.AccessLogConfig.Fields) > 0 {
				// Filter fields
				m := make(map[string]any)
				// We need to map struct fields to json tags manually or use reflection.
				// Manual is faster and safer here since we know the struct.
				// Or we can marshal to JSON and unmarshal to map, then filter? Too slow.
				// Let's build the map manually based on requested fields.
				allowed := make(map[string]bool)
				for _, f := range state.AccessLogConfig.Fields {
					allowed[f] = true
				}

				if allowed["time"] {
					m["time"] = entry.Time
				}
				if allowed["method"] {
					m["method"] = entry.Method
				}
				if allowed["path"] {
					m["path"] = entry.Path
				}
				if allowed["protocol"] {
					m["protocol"] = entry.Protocol
				}
				if allowed["status"] {
					m["status"] = entry.Status
				}
				if allowed["duration_ms"] {
					m["duration_ms"] = entry.Duration
				}
				if allowed["remote_ip"] {
					m["remote_ip"] = entry.RemoteIP
				}
				if allowed["user_agent"] {
					m["user_agent"] = entry.UserAgent
				}
				if allowed["referer"] {
					m["referer"] = entry.Referer
				}
				if allowed["service"] {
					m["service"] = entry.Service
				}
				if allowed["upstream"] {
					m["upstream"] = entry.Upstream
				}
				if allowed["bytes_written"] {
					m["bytes_written"] = entry.BytesWritten
				}

				logOutput = m
			}

			if err := json.NewEncoder(g.AccessLog).Encode(logOutput); err != nil {
				log.Printf("access log: %v", err)
			}
		}

		if g.Metrics != nil {
			g.Metrics.IncRequest(serviceName, routeName, r.Method, strconv.Itoa(status))
			g.Metrics.ObserveLatency(serviceName, routeName, duration)
		}
	}()

	route := state.Routes.Match(r.Host, r.URL.Path)
	if route == nil {
		http.NotFound(lw, r)
		return
	}
	serviceName = route.Service
	routeName = route.Name
	svc, ok := state.Services[route.Service]
	if !ok || len(svc.Endpoints) == 0 {
		http.Error(lw, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	// minimal choice: first endpoint (TODO: plug LB)
	ep := state.balancers[route.Service].Next()
	if ep == nil {
		http.Error(lw, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	base := ep.URL()
	tr := g.Transports.Get(svc.Name)

	// upstream URL = base + path
	u := new(url.URL)
	*u = *base
	u.Path = joinSlash(base.Path, r.URL.Path)
	u.RawQuery = r.URL.RawQuery
	upstreamAddr = u.String()

	hdr := cloneHeader(r.Header)
	dropHopByHop(hdr)
	addXFF(hdr, r.RemoteAddr)
	setXFProto(hdr, r)
	setXFHost(hdr, r.Host)

	ctx := r.Context()
	if state.UpstreamTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, state.UpstreamTimeout)
		defer cancel()
	}

	reqUp, err := http.NewRequestWithContext(ctx, r.Method, u.String(), r.Body)
	if err != nil {
		http.Error(lw, "bad request", http.StatusBadRequest)
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
		ep.Feedback(false)
		http.Error(lw, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			log.Printf("error closing upstream body: %v", err)
		}
	}(resUp.Body)

	if resUp.StatusCode >= 500 {
		ep.Feedback(false)
	} else {
		ep.Feedback(true)
	}

	dropHopByHop(resUp.Header)
	copyHeaders(lw.Header(), resUp.Header)

	// Announce trailers if any
	if len(resUp.Trailer) > 0 {
		trailerKeys := make([]string, 0, len(resUp.Trailer))
		for k := range resUp.Trailer {
			trailerKeys = append(trailerKeys, k)
		}
		lw.Header().Set("Trailer", strings.Join(trailerKeys, ","))
	}

	lw.WriteHeader(resUp.StatusCode)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	_, _ = io.Copy(lw, resUp.Body)

	// Copy trailer values
	if len(resUp.Trailer) > 0 {
		for k, vv := range resUp.Trailer {
			for _, v := range vv {
				lw.Header().Add(k, v)
			}
		}
	}
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
		if k == "TE" && h.Get("TE") == "trailers" {
			continue
		}
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

type AccessLog struct {
	Time         time.Time `json:"time"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	Protocol     string    `json:"protocol"`
	Status       int       `json:"status"`
	Duration     int64     `json:"duration_ms"`
	RemoteIP     string    `json:"remote_ip"`
	UserAgent    string    `json:"user_agent"`
	Referer      string    `json:"referer"`
	Service      string    `json:"service,omitempty"`
	Upstream     string    `json:"upstream,omitempty"`
	BytesWritten int64     `json:"bytes_written"`
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int64
}

func (w *loggingResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += int64(n)
	return n, err
}

func (w *loggingResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}
