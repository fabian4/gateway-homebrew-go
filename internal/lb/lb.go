package lb

import (
	"net/url"
	"sync"

	"github.com/fabian4/gateway-homebrew-go/internal/model"
)

type Balancer interface {
	Next() *url.URL
}

type smoothWRR struct {
	mu    sync.Mutex
	peers []*peer
}

type peer struct {
	url           *url.URL
	weight        int
	currentWeight int
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

func (b *smoothWRR) Next() *url.URL {
	b.mu.Lock()
	defer b.mu.Unlock()

	var best *peer
	total := 0

	for _, p := range b.peers {
		p.currentWeight += p.weight
		total += p.weight
		if best == nil || p.currentWeight > best.currentWeight {
			best = p
		}
	}

	if best == nil {
		return nil
	}

	best.currentWeight -= total
	return best.url
}
