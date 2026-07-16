VERSION ?= $(shell cat VERSION)
LDFLAGS := -X main.version=$(VERSION)
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
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-server-"${VERSION}" ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-client-"${VERSION}" ./cmd/client

build-multi: build-multi-server build-multi-client

build-multi-server: build-server-web
	@mkdir -p bin
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building client/server for $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-client-$(VERSION)-$$os-$$arch$$ext ./cmd/server; \
		done; \
	done

build-multi-client:
	@mkdir -p bin
	@for os in linux darwin windows; do \
		for arch in amd64 arm64; do \
			ext=""; \
			[ "$$os" = "windows" ] && ext=".exe"; \
			echo "Building client/server for $$os/$$arch"; \
			GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-client-$(VERSION)-$$os-$$arch$$ext ./cmd/client; \
		done; \
	done

build-server-web:
	cd "$(WEB_DIR)/server" && npm ci && npm run build

clean:
	rm -rf bin/
