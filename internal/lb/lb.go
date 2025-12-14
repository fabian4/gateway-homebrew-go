package lb

import (
	"net/url"
	"sync"
	"time"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type Balancer interface {
	Next() Endpoint
}

type Endpoint interface {
	URL() *url.URL
	Feedback(success bool)
}

type smoothWRR struct {
	mu    sync.Mutex
	peers []*peer
}

type peer struct {
	url           *url.URL
	weight        int
	currentWeight int

	// Passive health
	fails     int
	skipUntil time.Time
}

func NewSmoothWRR(endpoints []model.Endpoint) Balancer {
	peers := make([]*peer, len(endpoints))
	for i, e := range endpoints {
		w := e.Weight
		if w <= 0 {
			w = 1
		}
		peers[i] = &peer{
			url:    e.URL,
			weight: w,
		}
	}
	return &smoothWRR{peers: peers}
}

func (b *smoothWRR) Next() Endpoint {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	var best *peer
	total := 0

	for _, p := range b.peers {
		// Skip unhealthy peers
		if !p.skipUntil.IsZero() && now.Before(p.skipUntil) {
			continue
		}
		// If probe time passed, treat as candidate (maybe reset fails? or just try once)

		p.currentWeight += p.weight
		total += p.weight
		if best == nil || p.currentWeight > best.currentWeight {
			best = p
		}
	}

	if best == nil {
		// All skipped? Fallback to random or just return nil?
		// If all are skipped, we might want to return one to probe, or fail.
		// Let's try to find *any* peer if all are skipped, to avoid complete outage?
		// Or just return nil and let handler return 502.
		// Let's return nil for now.
		return nil
	}

	best.currentWeight -= total
	return &peerEndpoint{p: best, b: b}
}

type peerEndpoint struct {
	p *peer
	b *smoothWRR
}

func (e *peerEndpoint) URL() *url.URL {
	return e.p.url
}

func (e *peerEndpoint) Feedback(success bool) {
	e.b.mu.Lock()
	defer e.b.mu.Unlock()

	if success {
		e.p.fails = 0
		e.p.skipUntil = time.Time{}
	} else {
		e.p.fails++
		if e.p.fails >= 3 { // TODO: Configurable
			e.p.skipUntil = time.Now().Add(10 * time.Second) // TODO: Configurable
		}
	}
}
