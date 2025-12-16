# Gateway Benchmark (Docker-based)

[![Benchmark CI](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml/badge.svg)](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml)

A lightweight, Docker-based benchmark suite designed to compare the data-plane performance of **Single-Binary Gateways**.

The benchmark runs in a non-Kubernetes, single-node environment to strictly evaluate data-plane efficiency. It is automatically executed via GitHub Actions to ensure reproducible results.

**🔗 Live Workflows:** [View latest benchmark runs](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml)

---

## 🏆 Benchmark Results

**Environment:** GitHub Actions (Docker) | **Tool:** wrk/wrk2

### 📊 Summary

| Case | Metric | homebrew | envoy | nginx | traefik |
| --- | --- | --- | --- | --- | --- |
| **baseline** | RPS | 2519.35 | 2798.24 | 2889.37 | 2514.50 |
|  | P50 (ms) | 73.59 | 66.09 | 64.10 | 74.29 |
|  | P90 (ms) | 93.39 | 86.67 | 80.79 | 89.84 |
|  | P99 (ms) | 150.96 | 135.45 | 131.28 | 151.45 |
| **concurrency** | RPS | 2489.03 | 2724.63 | 2876.22 | 2534.95 |
|  | P50 (ms) | 384.19 | 352.71 | 334.50 | 378.76 |
|  | P90 (ms) | 410.19 | 381.71 | 351.09 | 399.85 |
|  | P99 (ms) | 771.77 | 701.41 | 1230.00 | 719.98 |
| **loadbalancing** | RPS | 3889.47 | 3995.24 | 4519.87 | 3839.26 |
|  | P50 (ms) | 52.36 | 61.15 | 56.91 | 67.52 |
|  | P90 (ms) | 90.83 | 98.61 | 88.17 | 105.05 |
|  | P99 (ms) | 149.59 | 159.57 | 143.36 | 171.14 |
| **routing** | RPS | 2479.93 | 2737.31 | 2817.05 | 2530.19 |
|  | P50 (ms) | 74.91 | 67.76 | 65.83 | 73.92 |
|  | P90 (ms) | 92.77 | 84.63 | 81.63 | 89.43 |
|  | P99 (ms) | 151.93 | 138.53 | 134.10 | 148.93 |
## 🎯 Targets

| Gateway | Version | Type | Configuration |
|---|---|---|---|
| **homebrew** | 0.5.0 | **Go (Target)** | Single Binary |
| **Traefik** | v3.0.1 | Go | Static File Provider |
| **Envoy** | v1.30.1 | C++ | Static Config |
| **NGINX** | 1.26.0 | C | nginx.conf |

---

## 🧪 Test Cases

Each case evaluates a specific capability using `wrk` (throughput) and `wrk2` (latency).

1.  **Baseline**: Single upstream, no logic. Measures minimum forwarding overhead.
2.  **Routing**: Path routing (prefix/exact) with multiple route rules. Measures lookup cost.
3.  **Load Balancing**: Multiple upstreams with Round-Robin strategy. Measures connection reuse and selection logic.
4.  **Concurrency**: High number of keep-alive connections. Measures resource stability under pressure.
