# tools/upstream.mk
SHELL := /bin/sh
include ./common.mk

.PHONY: upstream-http upstream-http-build upstream-tcp upstream-tcp-build help
.DEFAULT_GOAL := upstream-http

upstream-http:
	cd "$(TOP)" && env ECHO_ADDR=:9001 $(GO) run ./examples/upstreams/http-echo

upstream-http-build:
	@$(MKDIR_BIN)
	cd "$(TOP)" && $(GO) build -o "$(BIN_DIR)/http-echo" ./examples/upstreams/http-echo

upstream-tcp:
	cd "$(TOP)" && env TCP_ECHO_ADDR=:9002 $(GO) run ./examples/upstreams/tcp-echo

upstream-tcp-build:
	@$(MKDIR_BIN)
	cd "$(TOP)" && $(GO) build -o "$(BIN_DIR)/tcp-echo" ./examples/upstreams/tcp-echo

help:
	@echo "Targets:"
	@echo "  upstream-http       – run HTTP echo on :9001"
	@echo "  upstream-http-build – build bin/http-echo"
	@echo "  upstream-tcp        – run TCP echo on :9002"
	@echo "  upstream-tcp-build  – build bin/tcp-echo"
