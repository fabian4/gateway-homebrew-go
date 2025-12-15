package metrics

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// Registry holds metrics.
type Registry struct {
	mu sync.RWMutex
	// Key is "name|labels"
	counters   map[string]uint64
	gauges     map[string]int64
	histograms map[string]*Histogram
}

type Histogram struct {
	Count   uint64
	Sum     float64
	Buckets []float64
	Counts  []uint64
}

func NewRegistry() *Registry {
	return &Registry{
		counters:   make(map[string]uint64),
		gauges:     make(map[string]int64),
		histograms: make(map[string]*Histogram),
	}
}

func (r *Registry) IncRequest(service, route, method, status string) {
	key := fmt.Sprintf("requests_total|service=\"%s\",route=\"%s\",method=\"%s\",status=\"%s\"", service, route, method, status)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[key]++
}

func (r *Registry) IncActiveConns(listener, service string) {
	key := fmt.Sprintf("active_connections|listener=\"%s\",service=\"%s\"", listener, service)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[key]++
}

func (r *Registry) DecActiveConns(listener, service string) {
	key := fmt.Sprintf("active_connections|listener=\"%s\",service=\"%s\"", listener, service)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[key]--
}

func (r *Registry) ObserveLatency(service, route string, duration time.Duration) {
	key := fmt.Sprintf("upstream_latency_seconds|service=\"%s\",route=\"%s\"", service, route)
	val := duration.Seconds()

	// Default buckets: .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10
	buckets := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}

	r.mu.Lock()
	defer r.mu.Unlock()

	h, ok := r.histograms[key]
	if !ok {
		h = &Histogram{
			Buckets: buckets,
			Counts:  make([]uint64, len(buckets)),
		}
		r.histograms[key] = h
	}

	h.Count++
	h.Sum += val
	for i, b := range h.Buckets {
		if val <= b {
			h.Counts[i]++
		}
	}
}

func (r *Registry) WritePrometheus(w io.Writer) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Counters
	keys := make([]string, 0, len(r.counters))
	for k := range r.counters {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) > 0 {
		_, _ = fmt.Fprintln(w, "# HELP requests_total Total number of requests")
		_, _ = fmt.Fprintln(w, "# TYPE requests_total counter")
		for _, k := range keys {
			parts := strings.Split(k, "|")
			if len(parts) == 2 {
				name, labels := parts[0], parts[1]
				_, _ = fmt.Fprintf(w, "%s{%s} %d\n", name, labels, r.counters[k])
			}
		}
	}

	// Gauges
	keys = make([]string, 0, len(r.gauges))
	for k := range r.gauges {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) > 0 {
		_, _ = fmt.Fprintln(w, "# HELP active_connections Number of active connections")
		_, _ = fmt.Fprintln(w, "# TYPE active_connections gauge")
		for _, k := range keys {
			parts := strings.Split(k, "|")
			if len(parts) == 2 {
				name, labels := parts[0], parts[1]
				_, _ = fmt.Fprintf(w, "%s{%s} %d\n", name, labels, r.gauges[k])
			}
		}
	}

	// Histograms
	keys = make([]string, 0, len(r.histograms))
	for k := range r.histograms {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	if len(keys) > 0 {
		_, _ = fmt.Fprintln(w, "# HELP upstream_latency_seconds Upstream latency in seconds")
		_, _ = fmt.Fprintln(w, "# TYPE upstream_latency_seconds histogram")
		for _, k := range keys {
			parts := strings.Split(k, "|")
			if len(parts) == 2 {
				name, labels := parts[0], parts[1]
				h := r.histograms[k]

				for i, b := range h.Buckets {
					_, _ = fmt.Fprintf(w, "%s_bucket{%s,le=\"%g\"} %d\n", name, labels, b, h.Counts[i])
				}
				_, _ = fmt.Fprintf(w, "%s_bucket{%s,le=\"+Inf\"} %d\n", name, labels, h.Count)
				_, _ = fmt.Fprintf(w, "%s_sum{%s} %g\n", name, labels, h.Sum)
				_, _ = fmt.Fprintf(w, "%s_count{%s} %d\n", name, labels, h.Count)
			}
		}
	}
}
