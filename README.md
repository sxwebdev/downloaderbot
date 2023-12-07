# Downloader bot

This bot downloads media files from Instagram and YouTube and uploads them to the telegram bot

## Environment variables

Environment variables are [here](https://github.com/sxwebdev/downloaderbot/blob/master/ENVS.md)

### Required environment variables

```bash
ENV_CI=local # dev / stage / prod
REDIS_ADDR=localhost:6379 # redis server address
TELEGRAM_BOT_API_TOKEN= # your telegram bot api token
```

## TODO

- [x] GRPC api
- [x] Download photos from instagram
- [x] Download videos from instagram
- [ ] Download reels from instagram
- [ ] Download from youtube
- [x] Telegram bot
- [ ] Telegram inline bot
- [ ] Task manager for scheduling jobs
- [ ] Rate limiter for requests
- [ ] Docker

```bash
# start service
make start

# or you can use autoreload while developing
make watch
```
