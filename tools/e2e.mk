# mk/e2e.mk
.PHONY: e2e-up e2e-test e2e-down

e2e-up:
	@echo "TODO: docker compose up (gateway + http-echo)"

e2e-test:
	@echo "TODO: go test ./e2e/tests -v"

e2e-down:
	@echo "TODO: docker compose down -v"

HELP_LINES += "e2e:   e2e-up | e2e-test | e2e-down"
HELP_LINES += "  e2e-up            - (TODO) compose up gateway + upstreams"
HELP_LINES += "  e2e-test          - (TODO) end-to-end tests"
