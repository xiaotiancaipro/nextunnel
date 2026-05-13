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
	go build -trimpath -ldflags="-s -w" -o /out/nextunnel-client .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& addgroup -S nextunnel \
	&& adduser -S -G nextunnel -h /usr/local/nextunnel nextunnel \
	&& mkdir -p /usr/local/nextunnel/bin /usr/local/nextunnel/config \
	&& chown -R nextunnel:nextunnel /usr/local/nextunnel

WORKDIR /usr/local/nextunnel

COPY --from=builder --chown=nextunnel:nextunnel /out/nextunnel-client /usr/local/nextunnel/bin/nextunnel-client

USER nextunnel

EXPOSE 30985/tcp

ENTRYPOINT ["/usr/local/nextunnel/bin/nextunnel-client"]
CMD ["--config", "config/nextunnel-client.toml"]
