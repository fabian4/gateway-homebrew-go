# tools/go.mk
SHELL := /bin/sh
include ./common.mk

.PHONY: run build tidy fmt vet test clean help
.DEFAULT_GOAL := run

run:
	cd "$(TOP)" && $(GO) run $(RUN_LDFLAGS) ./cmd/gateway $(RUN_CONFIG)

build:
	@$(MKDIR_BIN)
	cd "$(TOP)" && $(GO) build $(if $(strip $(GO_LDFLAGS)),-ldflags '$(GO_LDFLAGS)',) \
		-o "$(BIN_DIR)/$(BIN)" ./cmd/gateway

tidy:
	cd "$(TOP)" && $(GO) mod tidy

fmt:
	cd "$(TOP)" && $(GO) fmt ./...

vet:
	cd "$(TOP)" && $(GO) vet ./...

test:
	cd "$(TOP)" && $(GO) test ./...

clean:
	@$(RM_BIN)

help:
	@echo "Targets:"
	@echo "  run       - go run ./cmd/gateway -config $(CONFIG)"
	@echo "  build     - build $(BIN_DIR)/$(BIN)"
	@echo "  tidy|fmt|vet|test|clean"
