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
> TODO: Error classification, counters, temporary de-preference/skip policy.
