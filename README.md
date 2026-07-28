# Downloader bot

This bot downloads media files from Instagram and YouTube and uploads them to the telegram bot

## Environment variables

Environment variables are [available here](ENVS.md)

### Required environment variables

```bash
DOWNLOADERBOT_TELEGRAM_BOT_API_TOKEN= # your telegram bot api token
```

## Known limitations

- **Inline results are capped at 20MB, not 50MB.** An inline result can only
  reference a URL, and Telegram downloads it itself — which the Bot API limits to
  "5 MB max size for photos and 20 MB max for other types of content". That is
  much stricter than the 50MB a bot may upload directly, so a video between 20MB
  and 50MB is delivered normally in a direct message but cannot be sent inline;
  the bot offers a download link for it instead. Instagram reels run past 20MB
  routinely — a two-minute 1080x1920 reel is around 22MB.
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
