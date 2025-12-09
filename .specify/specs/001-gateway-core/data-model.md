# Data Model: Gateway Core v0.1.0

## Configuration Schema (YAML)

The configuration file defines the gateway's behavior.

```yaml
server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: "10s"
  write_timeout: "10s"

logging:
  level: "info" # debug, info, warn, error
  format: "json" # json, text

clusters:
  - name: "backend-api"
    urls:
      - "http://localhost:8081"
      - "http://localhost:8082"
    load_balancing: "round-robin" # round-robin (v0.1.0)

routes:
  - path: "/api/v1/"
    cluster: "backend-api"
    methods: ["GET", "POST"]
    strip_prefix: true
```

## Internal Structures

### Config
```go
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Logging  LoggingConfig  `yaml:"logging"`
    Clusters []ClusterConfig `yaml:"clusters"`
    Routes   []RouteConfig   `yaml:"routes"`
}
```

### Route
```go
type Route struct {
    PathPrefix  string
    ClusterName string
    Methods     []string
    StripPrefix bool
    Handler     http.Handler // Constructed ReverseProxy
}
```

### Cluster
```go
type Cluster struct {
    Name      string
    Targets   []*url.URL
    LBPolicy  LoadBalancer
}
```
