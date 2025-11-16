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

## help: show targets
help:
	@echo "Usage: make <target>"
	@echo
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} /^[a-zA-Z0-9_-]+:.*##/ { printf "  %-12s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
