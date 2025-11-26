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

### Match model
- Host: exact hostname (case-insensitive). If omitted or empty the route is a wildcard (matches any host).
- PathPrefix: leading path segment prefix (must start with "/"). Simple byte prefix match; 
  - no regex, no glob, no normalization beyond what net/http already does for now.

### Algorithm (per inbound request)
1. Normalize the inbound Host header: lowercase and strip optional port ("example.com:443" -> "example.com").
2. Look up the route list for that exact host. If none match, fall back to the wildcard list (routes with no host defined).
3. Route lists are pre-sorted by descending PathPrefix length. The first route whose PathPrefix is a prefix of the request URL.Path wins.
4. If no route matches, the request is currently unhandled (future: default / 404 behavior).

### Precedence rules
1. Exact host routes always take precedence over wildcard routes.
2. Longer PathPrefix beats shorter (e.g. "/api/v1" before "/api").
3. Ties (same host presence and same PathPrefix length) preserve the order declared in the config (stable sort) – first wins.

### Path prefix semantics
- Prefix "/api" matches "/api", "/api/", and "/api/v1/items".
- For stricter separation, use a trailing slash: "/api/" will NOT match "/apix" but will match all under "/api/...".
- There is no automatic slash insertion; choose the prefix explicitly.

### Host semantics
- Exact only. No wildcards like "*.example.com" and no pattern matching yet.
- Single host per route keeps config simple; omit host for wildcard.

### Host rewrite options
- PreserveHost=true: upstream request Host header set to the original inbound host.
- HostRewrite: if non-empty overrides PreserveHost and sets upstream Host to the provided value.
- Default (neither option): upstream Host is the selected service endpoint host.

### Minimal example config
```yaml
services:
  - name: echo
    proto: http1
    endpoints:
      - http://127.0.0.1:9001
  - name: api
    proto: http1
    endpoints:
      - http://127.0.0.1:9002
routes:
  - name: api-v1
    match:
      host: "app.example.com"
      path_prefix: "/api/v1"
    service: api
  - name: api-root
    match:
      host: "app.example.com"
      path_prefix: "/api"
    service: api
  - name: echo-wildcard
    match:
      path_prefix: "/"
    service: echo
```

### Match examples
| Inbound Host | Path                | Selected Route   | Reason |
|--------------|---------------------|------------------|--------|
| app.example.com | /api/v1/users      | api-v1           | Exact host; longest prefix (/api/v1) |
| app.example.com | /api/status       | api-root         | Exact host; /api matches; /api/v1 does not |
| other.example.org | /api/v1/users   | echo-wildcard    | No exact host routes; wildcard fallback |
| other.example.org | /               | echo-wildcard    | Wildcard root prefix |

### Future extensions (not yet implemented)
- Wildcard / suffix host matching ("*.example.com").
- Regex or segment-aware routing.
- Priority weights and conditional predicates (method, header, query).
- Default / catch-all configurable response when no route matches.

