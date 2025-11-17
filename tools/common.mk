# mk/common.mk

GO       ?= go
BIN      ?= gateway
PKG_MAIN ?= ./cmd/gateway
CONFIG   ?= ./config.yaml

MODULE   ?= github.com/fabian4/gateway-homebrew-go
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  ?= -X $(MODULE)/internal/version.Value=$(VERSION) -s -w

REGISTRY ?= ghcr.io/fabian4
IMAGE    ?= $(REGISTRY)/gateway-homebrew-go

# Cross-platform mkdir/rm
ifeq ($(OS),Windows_NT)
  MKDIR_BIN := powershell -NoProfile -Command "New-Item -ItemType Directory -Force bin | Out-Null"
  RM_BIN    := powershell -NoProfile -Command "If (Test-Path bin) { Remove-Item -Recurse -Force bin }"
else
  MKDIR_BIN := mkdir -p bin
  RM_BIN    := rm -rf bin
endif
