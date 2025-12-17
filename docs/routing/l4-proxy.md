# L4 TCP Proxy

## Port-to-Cluster
You can map a specific listener port directly to a service cluster for L4 TCP proxying.

Example configuration:

```yaml
entrypoint:
  - name: web
    address: ":8080"
  - name: mysql-proxy
    address: ":3306"
    service: mysql-cluster

services:
  - name: mysql-cluster
    proto: tcp
    endpoints:
      - "tcp://10.0.0.1:3306"
      - "tcp://10.0.0.2:3306"
```

In this example:
- Traffic on port 8080 is handled by the L7 HTTP proxy (default behavior).
- Traffic on port 3306 is forwarded via TCP to `mysql-cluster`.

## Timeouts
You can configure idle and overall connection timeouts for L4 proxies in the `timeouts` section of the config.

```yaml
timeouts:
  tcp_idle: 5m        # Close connection if no data transferred for 5 minutes (default)
  tcp_connection: 1h  # Hard limit on connection duration (default: 0/unlimited)
```

- `tcp_idle`: Resets on every read/write operation.
- `tcp_connection`: Absolute duration from connection acceptance until forced closure.

## SNI-Mapping
> TODO: (Unreleased) SNI to cluster hints and limitations.
