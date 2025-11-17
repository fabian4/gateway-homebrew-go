# mk/docker.mk
.PHONY: docker-build docker-run docker-buildx docker-push

docker-build:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(VERSION) .

docker-run:
	docker run --rm -p 8080:8080 -v $(CONFIG):/etc/gateway/config.yaml:ro --name gateway-homebrew-go $(IMAGE):$(VERSION)

docker-buildx:
	docker buildx build --platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t $(IMAGE):$(VERSION) -t $(IMAGE):latest --push .

docker-push:
	docker push $(IMAGE):$(VERSION)

HELP_LINES += "docker: docker-build | docker-run | docker-buildx | docker-push"
HELP_LINES += "  docker-build       - build image $(IMAGE):$(VERSION)"
HELP_LINES += "  docker-run         - run image mounting $(CONFIG)"
HELP_LINES += "  docker-buildx      - multi-arch build & push (amd64,arm64)"
