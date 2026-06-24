# Downloader bot

This bot downloads media files from Instagram and YouTube and uploads them to the telegram bot

## Environment variables

Environment variables are [available here](ENVS.md)

### Required environment variables

```bash
DOWNLOADERBOT_TELEGRAM_BOT_API_TOKEN= # your telegram bot api token
```

## Known limitations

- **TikTok is not available in inline mode.** TikTok CDN URLs only serve the
  video when the request carries the browser's cookies + a `tiktok.com` referer,
  so the bot has to download the bytes itself (which it does in direct messages).
  Telegram inline results, however, can only reference a publicly fetchable URL
  or an already-uploaded `file_id` — raw bytes cannot be attached. Making TikTok
  work inline therefore requires either pre-uploading the video to obtain a
  `file_id` (needs a storage chat) or proxying it through a public URL; this is
  planned but not implemented yet. TikTok works normally when the link is sent to
  the bot in a direct message.
- **The gRPC API returns media URLs, not bytes.** For TikTok the returned URL
  needs the same cookies/referer headers to download, which the API does not
  currently expose, so API clients cannot fetch TikTok videos directly yet.
