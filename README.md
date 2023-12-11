# Downloader bot

This bot downloads media files from Instagram and YouTube and uploads them to the telegram bot

## Environment variables

Environment variables are [here](https://github.com/sxwebdev/downloaderbot/blob/master/ENVS.md)

### Required environment variables

```bash
ENV_CI=local # dev / stage / prod
REDIS_ADDR=localhost:6379 # redis server address
TELEGRAM_BOT_API_TOKEN= # your telegram bot api token
S3_ACCESS_ID=
S3_SECRET_KEY=
S3_REGION=en-1
S3_ENDPOINT=http://localhost:9050
S3_BUCKET_NAME=downloaderbot
S3_BASE_URL=
```

## TODO

- [x] GRPC api
- [x] Download photos from instagram
- [x] Download videos from instagram
- [x] Download reels from instagram
- [x] Download from youtube
- [x] Telegram bot
- [x] Telegram inline bot
- [x] Task manager for scheduling jobs
- [x] Rate limiter for requests
- [x] Dockerfile
- [x] Docker compose
- [x] S3 File storage (used for inline bot)

```bash
# start service
make start

# or you can use autoreload while developing
make watch
```
