# Gateway Benchmark (Docker-based)

[![Benchmark CI](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml/badge.svg)](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml)

A lightweight, Docker-based benchmark suite designed to compare the data-plane performance of **Single-Binary Gateways**.

The benchmark runs in a non-Kubernetes, single-node environment to strictly evaluate data-plane efficiency. It is automatically executed via GitHub Actions to ensure reproducible results.

**🔗 Live Workflows:** [View latest benchmark runs](https://github.com/fabian4/gateway-homebrew-go/actions/workflows/bench.yml)

---

## 🏆 Benchmark Results

**Environment:** GitHub Actions (Docker) | **Tool:** wrk/wrk2

### 📊 Master Summary

| Case | Metric | **Homebrew** (Go) | Traefik (Go) | Envoy (C++) | Nginx (C)   |
| :--- | :--- |:------------------| :--- | :--- |:------------|
| **1. Baseline**<br>*(Pure Forwarding)* | **RPS** (Req/sec) | **2,517.13**      | 2,568.82 | 2,772.30 | 2,994.50    |
| | P99 Latency | **150.08 ms**     | 149.06 ms | 135.76 ms | 126.13 ms   |
| **2. Routing**<br>*(Path Matching)* | **RPS** (Req/sec) | **2,544.21**      | 2,567.54 | 2,753.08 | 2,956.38    |
| | P99 Latency | **148.90 ms**     | 147.77 ms | 136.89 ms | 127.17 ms   |
| **3. Load Balancing**<br>*(Round Robin)* | **RPS** (Req/sec) | **3,894.73**      | 3,942.64 | 3,815.17 | 4,499.72    |
| | P50 Latency | **57.31 ms**    | 69.44 ms | 63.26 ms | 52.59 ms    |
| **4. Concurrency**<br>*(Long-lived Conn)* | P99 Latency | **719.51 ms**     | 694.23 ms | 680.34 ms | 1,150.00 ms |

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

---

## 🚀 Usage

### Running Locally
The runner script handles container lifecycle and load generation automatically.

```bash
# Syntax: ./runner/run.sh <case> <gateway>

# Example: Run baseline test for homebrew
./bench/runner/run.sh baseline homebrew

# Example: Run load balancing test for nginx
./bench/runner/run.sh loadbalancing nginx
