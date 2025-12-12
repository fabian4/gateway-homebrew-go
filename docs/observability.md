# Observability

## Access Log
The gateway outputs structured access logs in JSON format to stdout.

### Fields
- `time`: Timestamp of the request (RFC3339)
- `method`: HTTP method (GET, POST, etc.)
- `path`: Request path
- `protocol`: HTTP protocol version
- `status`: HTTP status code
- `duration`: Request duration
- `remote_ip`: Client IP address
- `user_agent`: User-Agent header
- `referer`: Referer header
- `service`: Matched service name (if any)
- `upstream`: Upstream URL (if any)
- `bytes_written`: Number of bytes written to the response body

## Metrics
> TODO: Define RPS, 4xx/5xx rates, upstream latency, active connections, route hits.
