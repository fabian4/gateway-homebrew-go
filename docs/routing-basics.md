# Routing Basics

## HTTP/1.1 reverse proxy

This section documents the minimal, from-scratch HTTP/1.1 reverse proxy used by the gateway. It proxies every inbound request to a single upstream (routing/LB come later).

### Overview
- **Protocol focus**: HTTP/1.1 upstream only (HTTP/2 explicitly disabled on the upstream Transport).
- **Transparency**: method, path, query and body are forwarded as-is; response headers/body are streamed back.
- **Headers**: hop-by-hop headers are removed; `X-Forwarded-*` are set/updated.
- **Timeouts**: server read/write/idle timeouts plus upstream dial/TLS timeouts are configured.

### Request flow (high level)
1. Accept inbound request on the HTTP server.
2. Build the upstream URL:
  - `scheme/host` from config,
  - path joined as `joinSlash(upstream.Path, req.URL.Path)`,
  - preserve `RawQuery`, drop fragment.
3. Clone inbound headers.
4. Remove hop-by-hop headers (and those named in `Connection` tokens).
5. Apply forwarding headers:  
   `X-Forwarded-For` (append client IP), `X-Forwarded-Host`, `X-Forwarded-Proto`.
6. Create upstream `*http.Request` with same method/body; set `Host`:
  - default: upstream host (`u.Host`);
  - optional: preserve incoming host (feature flag).
7. Send via a tuned `http.Transport` (HTTP/1.1 only).
8. On upstream response:
  - remove hop-by-hop headers,
  - copy headers to downstream,
  - write status, then stream body with `io.Copy`.

### Upstream Transport (dialing & timeouts)
- ForceAttemptHTTP2: false and TLSClientConfig.NextProtos: ["http/1.1"] to keep upstream on H1. 
- Dialer timeouts: Timeout=5s, KeepAlive=60s. 
- Pool: MaxIdleConns=200, MaxIdleConnsPerHost=100, IdleConnTimeout=90s. 
- TLS handshake timeout: 5s, ExpectContinueTimeout: 1s. 
- Optional: ResponseHeaderTimeout if you want a strict header wait bound.

### Hop-by-hop header handling

We remove both:
- Headers listed in Connection (comma-separated tokens). 
- Well-known hop-by-hop headers:
  - Connection
  - Proxy-Connection
  - Keep-Alive
  - Proxy-Authenticate
  - Proxy-Authorization
  - TE
  - Trailer
  - Transfer-Encoding
  - Upgrade

### Forwarding headers
- X-Forwarded-For: append the client IP (not replace). 
- X-Forwarded-Host: set to the inbound Host. 
- X-Forwarded-Proto: https if r.TLS != nil, otherwise http.
- (Optional) Add RFC 7239 Forwarded later if needed.

### Streaming semantics
- The proxy does not buffer whole bodies; io.Copy streams the upstream response to the client. 
- Go’s server will set Content-Length or chunked transfer encoding automatically based on headers and write pattern.

### Example: before/after headers

```http request
### Inbound (client → gateway)
GET /api/items?limit=10 HTTP/1.1
Host: app.example.com
Connection: keep-alive, X-Trace-Hop
Upgrade: websocket
X-Trace-Hop: abc123
X-Forwarded-For: 10.0.0.3
User-Agent: curl/8.5.0

### Context: r.RemoteAddr = "203.0.113.10:54321", TLS enabled on inbound (gateway terminates HTTPS).

### After clone + dropHopByHop + X-Forwarded- (gateway → upstream)
GET /api/items?limit=10 HTTP/1.1
Host: 127.0.0.1:9001
User-Agent: curl/8.5.0
X-Forwarded-For: 10.0.0.3, 203.0.113.10
X-Forwarded-Proto: https
X-Forwarded-Host: app.example.com
```
Removed: `Connection`, `Upgrade`, `X-Trace-Hop` (listed in Connection), `Keep-Alive`/`TE`/`Trailer`/`Transfer-Encoding` if present.

> If inbound already has XFF 
> 
> Inbound: `X-Forwarded-For: 172.16.0.5, 10.0.0.3`
> 
> After addXFF: `X-Forwarded-For: 172.16.0.5, 10.0.0.3, 203.0.113.10`


### Minimal config (current scope)
```yaml
listen: ":8080"
upstream: "http://127.0.0.1:9001"
```

### Quick test
```bash
# start upstream
go run ./examples/upstreams/http-echo

# run gateway
go run ./cmd/gateway -config config.yaml

# send a request
curl -i http://127.0.0.1:8080/hello -H "Host: example.local"
```

### Notes / TODO
•	Error mapping is minimal: upstream errors → 502 Bad Gateway.
•	WebSocket/Upgrade and HTTP/2 upstream are out of scope for this section and will be handled in later versions.
•	Body size limiting, request/response header rewrite, and logging middleware will be added in subsequent steps.

## Routing
> TODO: Define host and path-prefix matching, precedence rules, and examples.
