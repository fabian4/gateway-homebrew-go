# Research: Gateway Core v0.1.0

## Unknowns & Decisions

### 1. Routing Implementation
**Decision**: Use Go 1.22+ `net/http.ServeMux`.
**Rationale**:
- Supports method matching (`GET /path`) and wildcards (`/path/{id}`).
- Standard library (no external dependencies).
- Sufficient for v0.1.0 requirements (Host/Path-prefix).
**Alternatives Considered**:
- `gorilla/mux`: Mature but in maintenance mode; heavier.
- `go-chi/chi`: Lightweight and fast, but stdlib is preferred if sufficient.

### 2. Structured Logging
**Decision**: Use `log/slog` (Standard Library).
**Rationale**:
- Native structured logging support in Go.
- JSON handler built-in.
- Performance is adequate for 2k RPS target.
**Alternatives Considered**:
- `uber-go/zap`: Faster, but adds dependency. Can migrate if performance becomes a bottleneck.
- `sirupsen/logrus`: Older, slower than zap/slog.

### 3. Configuration Loading
**Decision**: Use `gopkg.in/yaml.v3` with custom environment variable and flag precedence.
**Rationale**:
- `yaml.v3` is robust for YAML parsing.
- Custom logic allows strict control over precedence (Flag > Env > Config File).
- Avoids `viper` complexity and size for a "minimal" gateway.
**Alternatives Considered**:
- `spf13/viper`: Standard for Go apps but can be heavy and opinionated.
- `knadh/koanf`: Lighter than Viper, but custom implementation is simpler for v0.1.0.

### 4. Reverse Proxy
**Decision**: Use `net/http/httputil.ReverseProxy`.
**Rationale**:
- Battle-tested standard library component.
- Supports HTTP/1.1 and HTTP/2.
- Extensible via `Director` and `Transport`.
**Alternatives Considered**:
- Custom implementation: High risk, reinventing the wheel.
- `fasthttp`: Incompatible with `net/http` ecosystem.
