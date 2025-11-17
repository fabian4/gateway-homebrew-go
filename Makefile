# Root Makefile (minimal)
.DEFAULT_GOAL := help

include tools/common.mk
include tools/go.mk
include tools/docker.mk
include tools/upstream.mk
include tools/e2e.mk

.PHONY: help
help:
	@$(info Usage: make <target>)
	@$(info )
	@$(foreach L,$(HELP_LINES),$(info $(L)))
	@:
