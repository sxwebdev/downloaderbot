FROM golang:1.22.6-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod .
COPY go.sum .

ENV APP_VERSION=0.1.0

RUN go mod download

COPY . .

RUN go build -o ./ -v -ldflags "-s -w -X main.version=${APP_VERSION}" ./cmd/app

FROM alpine:3.20.2

RUN apk add --no-cache iputils busybox-extras curl

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/app .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip .
ENV TZ=Europe/Moscow
ENV ZONEINFO=/app/zoneinfo.zip

USER appuser

ENTRYPOINT ["/app/app"]
