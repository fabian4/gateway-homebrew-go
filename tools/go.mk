# mk/go.mk
.PHONY: run build tidy fmt vet test clean

run:
	$(GO) run -ldflags "$(LDFLAGS)" $(PKG_MAIN) -config $(CONFIG)

build:
	@$(MKDIR_BIN)
	$(GO) build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG_MAIN)

tidy:
	$(GO) mod tidy

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

clean:
	@$(RM_BIN)

HELP_LINES += "go:    run | build | tidy | fmt | vet | test | clean"
HELP_LINES += "  run                - run gateway with CONFIG (default: ./config.yaml)"
HELP_LINES += "  build              - build ./bin/$(BIN)"
HELP_LINES += "  test               - go test ./..."
