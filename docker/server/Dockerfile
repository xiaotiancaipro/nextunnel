FROM --platform=$BUILDPLATFORM node:22-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
COPY web/shared/package.json ./shared/
COPY web/server/package.json ./server/
RUN --mount=type=cache,target=/root/.npm npm ci

COPY web/shared ./shared
COPY web/server ./server
RUN npm run build -w nextunnel-server-web

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .
COPY --from=web-builder /src/internal/server/controllers/dist ./internal/server/controllers

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/nextunnel-server ./cmd/server

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& mkdir -p /etc/nextunnel/certs \
	&& mkdir -p /var/log/nextunnel

WORKDIR /root

COPY --from=builder /out/nextunnel-server /usr/local/bin/nextunnel-server

ENTRYPOINT ["/usr/local/bin/nextunnel-server"]
CMD ["--config", "/etc/nextunnel/nextunnel-server.toml"]
