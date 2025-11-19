# tools/docker.mk
SHELL := /bin/sh
include ./common.mk

# Registry/image defaults (override via env if needed)
REGISTRY ?= ghcr.io/fabian4
IMAGE    ?= $(REGISTRY)/gateway-homebrew-go
DOCKERFILE ?= $(TOP)/Dockerfile
CONTEXT    ?= $(TOP)

# Build args
GO_VERSION ?= 1.24.4
PLATFORMS  ?= linux/amd64,linux/arm64
BUILDER    ?= ghx

.PHONY: docker-build docker-build-latest docker-run docker-push docker-buildx \
        docker-buildx-create docker-login-ghcr docker-img-inspect docker-clean

## docker-build: single-arch local build (tags :$(VERSION))
docker-build:
	docker build \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg GO_VERSION=$(GO_VERSION) \
	  -t $(IMAGE):$(VERSION) \
	  -f $(DOCKERFILE) $(CONTEXT)

## docker-build-latest: tag local image as :latest
docker-build-latest: docker-build
	docker tag $(IMAGE):$(VERSION) $(IMAGE):latest

## docker-run: run container mapping CONFIG to /etc/gateway/config.yaml
docker-run:
	@[ -f "$(CONFIG)" ] || (echo "CONFIG not found: $(CONFIG)"; exit 1)
	docker run --rm -p 8080:8080 \
	  -v "$(CONFIG)":/etc/gateway/config.yaml:ro \
	  --name gateway-homebrew-go \
	  $(IMAGE):$(VERSION)

## docker-push: push :$(VERSION) (and :latest if present locally)
docker-push:
	docker push $(IMAGE):$(VERSION)
	- docker push $(IMAGE):latest

## docker-buildx-create: create/use a buildx builder (idempotent)
docker-buildx-create:
	- docker buildx create --name $(BUILDER) --use
	- docker buildx use $(BUILDER)

## docker-buildx: multi-arch build & push (:$(VERSION) and :latest)
docker-buildx: docker-buildx-create
	docker buildx build \
	  --platform $(PLATFORMS) \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg GO_VERSION=$(GO_VERSION) \
	  -t $(IMAGE):$(VERSION) \
	  -t $(IMAGE):latest \
	  -f $(DOCKERFILE) $(CONTEXT) \
	  --push

## docker-login-ghcr: login using GHCR_PAT (or fail)
docker-login-ghcr:
	@test -n "$$GHCR_PAT" || (echo "GHCR_PAT is not set"; exit 1)
	printf %s "$$GHCR_PAT" | docker login ghcr.io -u fabian4 --password-stdin

## docker-img-inspect: show manifest (architectures)
docker-img-inspect:
	docker buildx imagetools inspect $(IMAGE):$(VERSION)

## docker-clean: remove local images for current VERSION
docker-clean:
	- docker rmi $(IMAGE):$(VERSION)
	- docker rmi $(IMAGE):latest
