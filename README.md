# Downloader bot

This bot downloads media files from Instagram and YouTube and uploads them to the telegram bot

Try to chek it: [@mxsaverbot](https://t.me/mxsaverbot)

## Environment variables

Environment variables are [here](ENVS.md)

### Required environment variables

```bash
DOWNLOADERBOT_TELEGRAM_BOT_API_TOKEN= # your telegram bot api token
```

## TODO

- [x] GRPC api
- [x] Download photos from instagram
- [x] Download videos from instagram
- [x] Download reels from instagram
- [x] Download from youtube
- [x] Telegram bot
- [x] Telegram inline bot
- [x] Rate limiter for requests
- [x] Dockerfile
- [x] Docker compose

```bash
# start service
make start

# or you can use autoreload while developing
make watch
```
