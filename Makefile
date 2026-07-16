.PHONY: \
	build \
	build-server \
	build-client \
	build-multi \
	build-multi-server \
	build-multi-client \
	clean

VERSION ?= $(shell cat VERSION)
LDFLAGS := -X main.version=$(VERSION)

build: build-server build-client

build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-server-"${VERSION}" ./cmd/server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-client-"${VERSION}" ./cmd/client

build-multi: build-multi-server build-multi-client

build-multi-server:
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

clean:
	rm -rf bin/
