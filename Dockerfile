FROM golang:1.25.6-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod .
COPY go.sum .

# Define build arguments for version, commit, and date.
ARG VERSION="unknown"
ARG COMMIT_HASH="unknown"
ARG BUILD_DATE="unknown"

RUN go mod download

COPY . .

RUN go build -trimpath -ldflags="-w -s -X 'main.version=${VERSION}' -X 'main.commitHash=${COMMIT_HASH}' -X 'main.buildDate=${BUILD_DATE}'" -o bin/downloaderbot ./cmd/app

FROM alpine:latest

RUN apk add --no-cache iputils busybox-extras curl

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/bin/downloaderbot .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip .
ENV TZ=Europe/Moscow
ENV ZONEINFO=/app/zoneinfo.zip

USER appuser

ENTRYPOINT ["/app/downloaderbot"]
