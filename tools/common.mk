# tools/common.mk

SHELL := /bin/sh

# Resolve repo root from the location of this file (../ from tools/)
ifndef TOP
TOP := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
endif

# Go/tooling (robust defaults even if env vars are empty)
GO ?= go
ifeq ($(strip $(GO)),)
  GO := go
endif

MODULE   ?= github.com/fabian4/gateway-homebrew-go
BIN      ?= gateway

# Default config under repo root
CONFIG ?= $(TOP)/cmd/config.yaml

# Version and ldflags (avoid colliding with system LDFLAGS)
VERSION    ?= $(shell cd "$(TOP)" && git describe --tags --always --dirty 2>/dev/null || echo dev)
GO_LDFLAGS ?= -X $(MODULE)/internal/version.Value=$(VERSION) -s -w

# CLI fragments (append only when non-empty)
RUN_LDFLAGS := $(if $(strip $(GO_LDFLAGS)),-ldflags '$(GO_LDFLAGS)',)
RUN_CONFIG  := $(if $(strip $(CONFIG)),-config $(CONFIG),)

# Artifacts
BIN_DIR := $(TOP)/bin

# Cross-platform mkdir/rm
ifeq ($(OS),Windows_NT)
  MKDIR_BIN := powershell -NoProfile -Command "New-Item -ItemType Directory -Force '$(BIN_DIR)' | Out-Null"
  RM_BIN    := powershell -NoProfile -Command "If (Test-Path '$(BIN_DIR)') { Remove-Item -Recurse -Force '$(BIN_DIR)' }"
else
  MKDIR_BIN := mkdir -p '$(BIN_DIR)'
  RM_BIN    := rm -rf '$(BIN_DIR)'
endif
