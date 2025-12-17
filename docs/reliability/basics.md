# Reliability Basics

## Timeouts
You can configure timeouts in the `timeouts` section of your `config.yaml`.

```yaml
timeouts:
  read: 15s      # Max time to read the request body
  write: 30s     # Max time to write the response
  upstream: 10s  # Max time for the upstream request (round-trip)
```

Defaults if not specified:
- `read`: 15s
- `write`: 30s
- `upstream`: 0 (no timeout, but `dial_timeout` applies)

## Passive-Health

The gateway implements a basic passive health check mechanism (circuit breaker) for upstream endpoints.

- **Failure Detection**: If an upstream request fails (network error or 5xx status code), the failure count for that endpoint is incremented.
- **Threshold**: If an endpoint fails **3 consecutive times**, it is marked as unhealthy.
- **Skip Policy**: Unhealthy endpoints are skipped (de-preferenced) for **10 seconds**. After this period, they are eligible for selection again (probing).
- **Success Reset**: A successful response (status < 500) resets the failure count and clears the unhealthy status.

This behavior is currently hardcoded but ensures that failing upstreams do not impact overall service availability if healthy endpoints are available.
