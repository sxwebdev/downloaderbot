FROM golang:1.26.4-alpine AS builder

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

# chromium + fonts are needed for the headless-browser Instagram fetcher (go-rod).
RUN apk add --no-cache \
	iputils busybox-extras curl \
	chromium nss freetype harfbuzz ca-certificates ttf-freefont font-noto-emoji

# Point the browser fetcher at the system Chromium instead of letting go-rod
# download its own (which would not match Alpine/musl).
ENV BROWSER_BIN=/usr/bin/chromium-browser

RUN adduser -D -g '' appuser

WORKDIR /app

COPY --from=builder --chown=appuser:appuser /app/bin/downloaderbot .

COPY --from=builder /usr/local/go/lib/time/zoneinfo.zip .
ENV ZONEINFO=/app/zoneinfo.zip

USER appuser

ENTRYPOINT ["/app/downloaderbot"]
