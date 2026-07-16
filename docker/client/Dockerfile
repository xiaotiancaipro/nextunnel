FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder

WORKDIR /src

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG VERSION=v0.0.0

RUN --mount=type=cache,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
	go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /out/nextunnel-client ./cmd/client

FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
	&& mkdir -p /etc/nextunnel/certs \
    && mkdir -p /var/log/nextunnel

WORKDIR /root

COPY --from=builder /out/nextunnel-client /usr/local/bin/nextunnel-client

USER root

ENTRYPOINT ["/usr/local/bin/nextunnel-client"]
CMD ["--config", "/etc/nextunnel/nextunnel-client.toml"]
