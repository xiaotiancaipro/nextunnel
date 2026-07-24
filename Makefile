VERSION ?= $(shell cat VERSION)
WEB_DIR := web

.PHONY: \
	build \
	build-server \
	build-client \
	build-multi \
	build-multi-server \
	build-multi-client \
	build-server-web \
	clean

build: build-server build-client

build-server: build-server-web
	go build -o bin/nextunnel-server-"${VERSION}" ./cmd/server

build-client:
	go build -o bin/nextunnel-client-"${VERSION}" ./cmd/client

build-multi: build-multi-server build-multi-client

build-multi-server: build-server-web
	@mkdir -p bin
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building server for $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch go build -o bin/nextunnel-server-$(VERSION)-$$os-$$arch$$ext ./cmd/server; \
		done; \
	done

build-multi-client:
	@mkdir -p bin
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building client for $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch go build -o bin/nextunnel-client-$(VERSION)-$$os-$$arch$$ext ./cmd/client; \
		done; \
	done

build-server-web:
	cd "$(WEB_DIR)" && npm ci && npm run build -w nextunnel-server-web

clean:
	rm -rf bin/
