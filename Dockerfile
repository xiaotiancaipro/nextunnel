FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
	go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags="-s -w" -o /out/nextunnel-server .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& mkdir -p /usr/local/nextunnel/bin \
    && mkdir -p /usr/local/nextunnel/config \
    && mkdir -p /usr/local/nextunnel/certs \
    && mkdir -p /usr/local/nextunnel/geoip \
    && mkdir -p /usr/local/nextunnel/logs

WORKDIR /usr/local/nextunnel

COPY --from=builder /out/nextunnel-server /usr/local/nextunnel/bin/nextunnel-server

ENTRYPOINT ["/usr/local/nextunnel/bin/nextunnel-server"]
CMD ["--config", "config/nextunnel-server.toml"]
