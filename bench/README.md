# **Gateway Benchmark (Docker-based, Single-Binary)**

This repository contains a **Docker-based benchmark suite** for comparing single-binary API gateways.

The benchmark is designed around a **non-Kubernetes, single-process deployment model**, where every gateway is evaluated **as a container image**, reflecting how gateways are commonly released and consumed in practice.

Primary target:
- **homebrew** (Go, single-binary gateway)

Comparison targets:
- Envoy (static configuration)
- NGINX
- Traefik

---
## **Goals**
This benchmark aims to answer the following questions:
1. What is the **data-plane performance cost** of common L7 gateway features?
2. How does a **Go single-binary gateway** compare to mature gateways in terms of:
  - Throughput
  - Tail latency
  - Resource usage
3. Can performance regressions be detected automatically in CI?

This project intentionally does **not** test:
- Kubernetes or Gateway API
- Control-plane behavior
- Feature completeness

The focus is strictly on **data-plane efficiency and simplicity**.

---
## **Design Principles**
- **Docker-first**: every gateway runs as a container image
- **Single-node**: no orchestration, no Kubernetes
- **Feature-equivalent configs** across gateways
- **One variable per test case**
- **CI-friendly**: results are used for trend and regression detection, not absolute numbers

---
## **Gateways Under Test**
|**Gateway**|**Mode**|**Version**|
|---|---|---|
|homebrew|Go single-binary|v0.5.0|
|Envoy|Static config|v1.30.1|
|NGINX|nginx.conf|1.26.0|
|Traefik|Static file provider|v3.0.1|

All gateways are started via Docker and exposed on the same host port for testing.

---
## **Benchmark Architecture**

```
+-------------+
| Load Client |
|  wrk / wrk2|
+------+------+
       |
+------v------+
|   Gateway   |  (one at a time)
+------+------+
       |
+------v------+
|   Backend   |  (lightweight echo server)
+-------------+
```

- Only **one gateway** is running per benchmark run
- The backend is a minimal HTTP echo server to avoid bottlenecks

---
## **Benchmark Cases**

Each case focuses on a **single capability** and is executed independently.

### **Case 0: Baseline (Pure Forwarding)**
- HTTP/1.1
- Single upstream
- No routing logic
- No header mutation
- No load balancing
- No TLS

Purpose: measure the minimum forwarding overhead.

---
### **Case 1: Path Routing**

- Prefix and exact path matching
- Multiple routes mapped to the same backend

Purpose: evaluate routing table lookup cost.

---
### **Case 2: Header Manipulation**
- Add / remove request headers
- Add response headers

Purpose: measure allocation and header processing overhead.

---
### **Case 3: Load Balancing**
- Multiple upstream backends
- Round-robin (first phase)
- Keep-alive enabled

Purpose: evaluate upstream selection and connection reuse.

---
### **Case 4: High Concurrency / Long-lived Connections**
- Large number of keep-alive connections
- Fixed request rate (wrk2)

Purpose: test connection management and tail latency stability.

---
## **Load Generation**

The benchmark uses:
- **wrk**: maximum throughput tests
- **wrk2**: fixed-rate tests for tail latency

Example:

```
wrk -t4 -c200 -d30s http://127.0.0.1:8080/
wrk -t4 -c200 -d30s -R 50000 http://127.0.0.1:8080/
```

---

## **Repository Layout**

```
bench/
├── cases/        # Case definitions
├── gateways/     # Dockerfiles and configs per gateway
├── runner/       # Benchmark orchestration scripts
├── results/      # JSON benchmark output
└── README.md
```

---
## **Result Format**

All benchmarks output a unified JSON format:

```
{
  "gateway": "homebrew",
  "case": "baseline",
  "rps": 142381,
  "latency": {
    "p50": 1.1,
    "p90": 2.3,
    "p99": 4.8
  },
  "cpu_pct": 165,
  "rss_mb": 78
}
```

This allows:
- Automated comparison
- CI regression detection
- Long-term trend tracking

---

## **CI Integration**
The benchmark is designed to run in **GitHub Actions**.

CI guidelines:
- Each case ≤ 30 seconds
- Total runtime ≤ 15 minutes
- Results are used for **relative comparison**, not absolute performance claims

Example workflow:

```
name: gateway-benchmark

on:
  pull_request:
  workflow_dispatch:

jobs:
  bench:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Run baseline benchmark
      run: ./bench/runner/run.sh baseline homebrew
```

---
## **Philosophy**
This benchmark intentionally favors:
- **Simplicity over completeness**
- **Explainability over raw numbers**
- **Performance per complexity**, not just peak throughput

A single-binary gateway should demonstrate:
- Lower memory footprint
- Predictable latency
- Minimal operational overhead

---
## **Status**
- Baseline case implemented
- Envoy adapter
- NGINX adapter
- Traefik adapter
- CI regression gate

---

Contributions and discussions are welcome.
