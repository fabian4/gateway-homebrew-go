# mk/upstreams.mk
.PHONY: upstream-http upstream-http-build upstream-tcp upstream-tcp-build

upstream-http:
	ECHO_ADDR=:9001 $(GO) run ./examples/upstreams/http-echo

upstream-http-build:
	@$(MKDIR_BIN)
	$(GO) build -o bin/http-echo ./examples/upstreams/http-echo

upstream-tcp:
	TCP_ECHO_ADDR=:9002 $(GO) run ./examples/upstreams/tcp-echo

upstream-tcp-build:
	@$(MKDIR_BIN)
	$(GO) build -o bin/tcp-echo ./examples/upstreams/tcp-echo

HELP_LINES += "upstream: upstream-http | upstream-tcp"
HELP_LINES += "  upstream-http      - run HTTP echo on :9001"
HELP_LINES += "  upstream-tcp       - run TCP echo on :9002"
