# Observability

## Access Log
The gateway outputs structured access logs in JSON format to stdout.

### Fields
- `time`: Timestamp of the request (RFC3339)
- `method`: HTTP method (GET, POST, etc.)
- `path`: Request path
- `protocol`: HTTP protocol version
- `status`: HTTP status code
- `duration_ms`: Request duration in milliseconds
- `remote_ip`: Client IP address
- `user_agent`: User-Agent header
- `referer`: Referer header
- `service`: Matched service name (if any)
- `upstream`: Upstream URL (if any)
- `bytes_written`: Number of bytes written to the response body

## Metrics
The gateway exposes Prometheus-compatible metrics on a configured address (e.g. `:9090`).

### Configuration
```yaml
metrics:
  address: ":9090"
```

### Exposed Metrics
- `requests_total`: Counter of HTTP requests (labels: `service`, `route`, `method`, `status`).
- `upstream_latency_seconds`: Histogram of upstream response latency (labels: `service`, `route`).
- `active_connections`: Gauge of active L4 TCP connections (labels: `listener`, `service`).
