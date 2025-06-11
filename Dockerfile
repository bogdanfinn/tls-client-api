FROM golang:1.24.4-bookworm AS base

WORKDIR /app

COPY go.mod go.sum .
RUN go mod download
RUN go mod verify

FROM base AS builder

COPY ./cmd/tls-client-api/main.go ./cmd/tls-client-api/main.go
COPY ./internal ./internal

RUN CGO_ENABLED=0 go build -o tls-client-api ./cmd/tls-client-api/main.go

FROM alpine

WORKDIR /app

RUN apk add --no-cache ca-certificates yq

COPY --from=builder /app/tls-client-api /app/tls-client-api

COPY ./cmd/tls-client-api/entrypoint.sh /app/entrypoint.sh
COPY ./cmd/tls-client-api/config.dist.yml /app/config.dist.yml

RUN chmod +x /app/tls-client-api
RUN chmod +x /app/entrypoint.sh

CMD ["/app/entrypoint.sh"]
