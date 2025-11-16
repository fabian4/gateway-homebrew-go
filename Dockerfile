# syntax=docker/dockerfile:1

########################
# Build stage
########################
FROM golang:1.24.4 AS builder
WORKDIR /src

# deps cache
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# source
COPY . .

ARG VERSION=v0.0.1
ARG GOARCH=amd64
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=${GOARCH} \
    go build -trimpath \
      -ldflags="-s -w -X github.com/fabian4/gateway-homebrew-go/internal/version.Value=${VERSION}" \
      -o /out/gateway ./cmd/gateway

########################
# Runtime stage
########################
FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/gateway /usr/local/bin/gateway

USER nonroot:nonroot
EXPOSE 8080

# read config from /etc/gateway/config.yaml
ENTRYPOINT ["/usr/local/bin/gateway"]
CMD ["-config", "/etc/gateway/config.yaml"]
