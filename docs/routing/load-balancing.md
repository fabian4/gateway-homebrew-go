# Load Balancing

## WRR
The gateway implements Nginx-style Smooth Weighted Round Robin (WRR).
Each upstream endpoint can be assigned a weight (default 1).
The algorithm ensures a smooth distribution of requests according to weights, avoiding burstiness.

Configuration example:
```yaml
services:
  - name: backend
    endpoints:
      - { url: "http://srv1:8080", weight: 5 }
      - { url: "http://srv2:8080", weight: 1 }
      - "http://srv3:8080" # default weight 1
```

## Least-Conn
> TODO: (Unreleased) Define algorithm sketch and tie-ins to connection stats.

## Consistent-Hash
> TODO: (Unreleased) Key selection (header/IP), hash ring, and affinity notes.
