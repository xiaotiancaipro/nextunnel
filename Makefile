VERSION ?= $(shell cat VERSION)
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build build-client build-server clean

build: build-client build-server

build-client:
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-client ./cmd/client

build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/nextunnel-server ./cmd/server

clean:
	rm -rf bin/
