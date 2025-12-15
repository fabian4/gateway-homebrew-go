package handler

import (
	"io"
	"log"
	"net"
	"time"

	"github.com/fabian4/gateway-homebrew-go/internal/lb"
)

// TCPProxy handles L4 TCP proxying.
type TCPProxy struct {
	Balancer lb.Balancer
}

func NewTCPProxy(balancer lb.Balancer) *TCPProxy {
	return &TCPProxy{Balancer: balancer}
}

func (p *TCPProxy) Handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	ep := p.Balancer.Next()
	if ep == nil {
		log.Printf("tcp proxy: no healthy upstream")
		return
	}

	// Dial upstream
	// We use the Host from the URL (e.g. "127.0.0.1:8080")
	u := ep.URL()
	upstream, err := net.DialTimeout("tcp", u.Host, 5*time.Second)
	if err != nil {
		log.Printf("tcp proxy: dial upstream %s: %v", u.Host, err)
		ep.Feedback(false)
		return
	}
	defer func() { _ = upstream.Close() }()

	ep.Feedback(true)

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(upstream, conn)
		if c, ok := upstream.(*net.TCPConn); ok {
			_ = c.CloseWrite()
		}
		close(done)
	}()

	_, _ = io.Copy(conn, upstream)
	if c, ok := conn.(*net.TCPConn); ok {
		_ = c.CloseWrite()
	}
	<-done
}
