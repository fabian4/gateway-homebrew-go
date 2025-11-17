# syntax=docker/dockerfile:1
ARG GO_VERSION=1.24.4
FROM golang:${GO_VERSION} AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG PKG=./examples/upstreams/http-echo
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -o /out/echo ${PKG}

FROM gcr.io/distroless/base-debian12:nonroot
WORKDIR /app
COPY --from=builder /out/echo /usr/local/bin/echo
USER nonroot:nonroot
EXPOSE 9001
ENTRYPOINT ["/usr/local/bin/echo"]
