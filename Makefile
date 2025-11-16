# Makefile for gateway-homebrew-go

GO       ?= go
BIN      ?= gateway
PKG_MAIN := ./cmd/gateway
CONFIG   ?= ./cmd/config.yaml

MODULE   := github.com/fabian4/gateway-homebrew-go
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -X $(MODULE)/internal/version.Value=$(VERSION) -s -w

.PHONY: run build tidy fmt vet test clean help

## run: run the gateway with CONFIG (default: config.yaml)
run:
	$(GO) run -ldflags "$(LDFLAGS)" $(PKG_MAIN) -config $(CONFIG)

## build: build a release-like binary to ./bin/$(BIN)
build:
	@mkdir -p bin
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG_MAIN)

## tidy: go mod tidy
tidy:
	$(GO) mod tidy

## fmt: go fmt
fmt:
	$(GO) fmt ./...

## vet: go vet
vet:
	$(GO) vet ./...

## test: run unit tests
test:
	$(GO) test ./...

## clean: remove build outputs
clean:
	rm -rf bin

## upstream: run the Go HTTP echo upstream on :9001
upstream-http:
	ECHO_ADDR=:9001 go run ./examples/upstreams/http-echo

## upstream-build: build the echo binary to ./bin/http-echo
upstream-http-build:
	@mkdir -p bin
	go build -o bin/http-echo ./examples/upstreams/http-echo

## upstream-tcp: run the Go TCP echo upstream on :9002
upstream-tcp:
	TCP_ECHO_ADDR=:9002 go run ./examples/upstreams/tcp-echo

## upstream-tcp-build: build the TCP echo binary to ./bin/tcp-echo
upstream-tcp-build:
	@mkdir -p bin
	go build -o bin/tcp-echo ./examples/upstreams/tcp-echo

## docker-build: build local image (amd64) with VERSION tag
docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg GOARCH=amd64 \
		-t fabian4/gateway-homebrew-go:$(VERSION) .

## docker-run: run container and mount config.yaml (host → container)
docker-run: ## CONFIG?=/path/to/config.yaml (default: ./config.yaml)
	@[ -f "$(CONFIG)" ] || (echo "CONFIG not found: $(CONFIG)"; exit 1)
	docker run --rm -p 8080:8080 \
		-v $(CONFIG):/etc/gateway/config.yaml:ro \
		--name gateway-homebrew-go \
		fabian4/gateway-homebrew-go:$(VERSION)

## docker-buildx: multi-arch build (amd64, arm64)
docker-buildx:
	# requires: docker buildx create --use (once)
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t fabian4/gateway-homebrew-go:$(VERSION) \
		--push .

## docker-push: push local tag to Docker Hub (single arch)
docker-push:
	docker push fabian4/gateway-homebrew-go:$(VERSION)

## help: show targets
help:
	@echo "Usage: make <target>"
	@echo
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
