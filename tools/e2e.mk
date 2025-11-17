# tools/e2e.mk
SHELL := /bin/sh
include ./common.mk

COMPOSE := docker compose -f $(TOP)/e2e/compose/docker-compose.yml

.PHONY: e2e-up e2e-test e2e-logs e2e-down e2e-reup

e2e-up:
	$(COMPOSE) up -d --build

e2e-test:
	# Wait a bit before tests in case gateway still warming up
	sleep 2
	cd "$(TOP)" && $(GO) test ./e2e/tests -v -count=1

e2e-logs:
	$(COMPOSE) logs --no-color --tail=200

e2e-down:
	$(COMPOSE) down -v

e2e-reup: e2e-down e2e-up
