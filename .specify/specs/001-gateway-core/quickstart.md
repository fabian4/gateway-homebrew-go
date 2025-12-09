# Quickstart: Gateway Core v0.1.0

## Prerequisites
- Go 1.24+
- Docker (optional)

## Build
```bash
go build -o gateway cmd/gateway/main.go
```

## Configuration
Create `config.yaml`:
```yaml
server:
  port: 8080
logging:
  level: "info"
clusters:
  - name: "httpbin"
    urls: ["https://httpbin.org"]
routes:
  - path: "/get"
    cluster: "httpbin"
    methods: ["GET"]
```

## Run
```bash
./gateway --config config.yaml
```

## Test
```bash
curl -v http://localhost:8080/get
```
Expected output: JSON response from httpbin.org.
