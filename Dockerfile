FROM golang:1.23.2-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod .
COPY go.sum .

ENV APP_VERSION=0.1.0

RUN go mod download

COPY . .

RUN go build -o ./service -v -ldflags "-s -w -X main.version=${APP_VERSION}" ./cmd/app

FROM alpine:3.20.2

RUN apk add --no-cache iputils busybox-extras curl

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/service .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip .
ENV TZ=Europe/Moscow
ENV ZONEINFO=/app/zoneinfo.zip

USER appuser

ENTRYPOINT ["/app/service"]
