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
	Balancer          lb.Balancer
	IdleTimeout       time.Duration
	ConnectionTimeout time.Duration
}

func NewTCPProxy(balancer lb.Balancer, idleTimeout, connectionTimeout time.Duration) *TCPProxy {
	return &TCPProxy{
		Balancer:          balancer,
		IdleTimeout:       idleTimeout,
		ConnectionTimeout: connectionTimeout,
	}
}

func (p *TCPProxy) Handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Overall connection timeout
	if p.ConnectionTimeout > 0 {
		timer := time.AfterFunc(p.ConnectionTimeout, func() {
			_ = conn.Close()
		})
		defer timer.Stop()
	}

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

	// Wrap connections for idle timeout
	var clientConn, upstreamConn = conn, upstream
	if p.IdleTimeout > 0 {
		clientConn = &idleTimeoutConn{Conn: conn, timeout: p.IdleTimeout}
		upstreamConn = &idleTimeoutConn{Conn: upstream, timeout: p.IdleTimeout}
	}

	done := make(chan struct{})
	go func() {
		_, _ = io.Copy(upstreamConn, clientConn)
		if c, ok := upstream.(*net.TCPConn); ok {
			_ = c.CloseWrite()
		}
		close(done)
	}()

	_, _ = io.Copy(clientConn, upstreamConn)
	if c, ok := conn.(*net.TCPConn); ok {
		_ = c.CloseWrite()
	}
	<-done
}

type idleTimeoutConn struct {
	net.Conn
	timeout time.Duration
}

func (c *idleTimeoutConn) Read(b []byte) (n int, err error) {
	_ = c.SetDeadline(time.Now().Add(c.timeout))
	return c.Conn.Read(b)
}

func (c *idleTimeoutConn) Write(b []byte) (n int, err error) {
	_ = c.SetDeadline(time.Now().Add(c.timeout))
	return c.Conn.Write(b)
}
