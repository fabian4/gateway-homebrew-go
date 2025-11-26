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
- **Host**
  - Exact hostname (case-insensitive), e.g. `"app.example.com"`.
  - Wildcard hostname of the form `"*.example.com"` which matches any subdomain of `example.com`:
    - `"api.example.com"`, `"deep.api.example.com"` match.
    - Bare `"example.com"` does **not** match `"*.example.com"`.
  - If omitted or empty, the route is **global** (matches any host).
- **PathPrefix**
  - Leading path segment prefix (must start with `/`).
  - Matching is **segment-aware**, not just raw `HasPrefix`:
    - `"/api"` matches `"/api"`, `"/api/"`, `"/api/v1/items"`.
    - `"/api"` does **not** match `"/apiary"`.
    - `"/"` matches everything.

### Algorithm (per inbound request)
1. Normalize the inbound Host header: lowercase and strip optional port (`"example.com:443"` → `"example.com"`).
2. Try **exact host routes** for that host.
3. If none match, try **wildcard host routes**, ordered from most-specific suffix to least (e.g. `"*.api.example.com"` before `"*.example.com"`).
4. If still none match, try **global routes** (routes with no host).
5. Within each host bucket, routes are pre-sorted by descending `PathPrefix` length; the first route whose `PathPrefix` matches the request `URL.Path` (using segment-aware matching) wins.

### Precedence rules
1. **Exact host** routes always take precedence over wildcard and global routes.
2. Within wildcard hosts, **more specific suffix** wins (e.g. `"*.api.example.com"` before `"*.example.com"` for `foo.api.example.com`).
3. Longer `PathPrefix` beats shorter (e.g. `"/api/v1"` before `"/api"`).
4. Ties (same host bucket and same `PathPrefix` length) preserve the order declared in the config (stable sort) – first wins.

### Default route semantics
- A route with `path_prefix: "/"` for a given host acts as that host’s **default route**:
  - If no more specific prefix matches for that host, its `/` route is used.
- A route with **empty host** and `path_prefix: "/"` acts as a **global default**:
  - Used only when no exact host or wildcard host routes match.

### Host rewrite options
- **PreserveHost=true**: upstream request `Host` header set to the original inbound host.
- **HostRewrite**: if non-empty overrides `PreserveHost` and sets upstream `Host` to the provided value.
- **Default** (neither option): upstream `Host` is the selected service endpoint host.

### Example config: exact host, wildcard host, and global default
```yaml
services:
  - name: api-v1
    proto: http1
    endpoints:
      - "http://127.0.0.1:19001"
  - name: api-root
    proto: http1
    endpoints:
      - "http://127.0.0.1:19002"
  - name: wildcard-subdomains
    proto: http1
    endpoints:
      - "http://127.0.0.1:19003"
  - name: global-default
    proto: http1
    endpoints:
      - "http://127.0.0.1:19004"

routes:
  # Exact host, longest prefix wins
  - name: api-v1
    match:
      host: "app.example.com"
      path_prefix: "/api/v1"
    service: api-v1

  - name: api-root
    match:
      host: "app.example.com"
      path_prefix: "/api"
    service: api-root

  # Per-host default for app.example.com
  - name: app-default
    match:
      host: "app.example.com"
      path_prefix: "/"
    service: api-root

  # Wildcard host: any subdomain of example.com (not example.com itself)
  - name: subdomains-example
    match:
      host: "*.example.com"
      path_prefix: "/"
    service: wildcard-subdomains

  # Global default for all other hosts
  - name: global-default
    match:
      host: ""
      path_prefix: "/"
    service: global-default
```

### Match examples
| Inbound Host        | Path          | Selected Route       | Reason |
|---------------------|---------------|----------------------|--------|
| `app.example.com`   | `/api/v1/ping` | `api-v1`             | Exact host; longest prefix `/api/v1` |
| `app.example.com`   | `/api/ping`    | `api-root`           | Exact host; `/api` matches, `/api/v1` does not |
| `app.example.com`   | `/unknown`     | `app-default`        | Per-host default `/` for `app.example.com` |
| `foo.example.com`   | `/healthz`     | `subdomains-example` | Matches `*.example.com` wildcard |
| `other.local`       | `/anything`    | `global-default`     | No exact or wildcard host; global default `/` |

### Future extensions (not yet implemented)
- Regex or more advanced segment-aware routing predicates (method, header, query).
- Priority weights and conditional match logic.
- Configurable default / catch-all responses when no route matches.

