FROM golang:1.26-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -trimpath -ldflags="-w -s" -o /nextunnel .

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /nextunnel /nextunnel

ENTRYPOINT ["/nextunnel"]
