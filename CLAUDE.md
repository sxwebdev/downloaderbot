# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A Telegram bot (`gopkg.in/telebot.v3`, long polling) that takes a media link, extracts the
downloadable media, and sends it back — plus a gRPC API exposing the same extraction. Instagram
and TikTok are scraped through a headless Chromium; YouTube and ~40 other sites go through
libraries.

## Commands

```bash
make start            # go run ./cmd/app start
make build            # -> ./.build/app
make test             # go test -race -count=1 ./...
make test-cover       # coverage + open HTML report
make fmt              # go fix ./... && gofumpt -l -w .
make vuln             # govulncheck
make genproto         # regenerate pb/ from proto/bot.proto
make genenvs          # regenerate config.template.yaml + ENVS.md from the Config struct
make grpcui           # grpcui against localhost:9000
make docker-run       # build local image and run it (needs .env with TELEGRAM_BOT_API_TOKEN)
make release-preview  # goreleaser dry run into ./dist
```

Running one test / package:

```bash
go test -race -run TestTruncateRunes ./internal/services/telegram/
go test -race -short ./...     # skips the live-network tests (see below)
```

`pkg/instagram` and `pkg/tiktok` contain **integration tests that hit the real sites and launch a
browser** (`instagram_test.go`, `browser_test.go`, `tiktok_test.go`). They are gated by
`testing.Short()`, so use `-short` when offline or when a failure would only mean "Instagram
changed its markup again". Everything else is pure unit tests.

`make genenvs` must be re-run after any change to `internal/config.Config` — `ENVS.md` and
`config.template.yaml` are generated artifacts, never edit them by hand.

Commits follow Conventional Commits with a scope, e.g. `fix(instagram): anchor caption to the
requested post`.

## Configuration

`internal/config.Load` (via `sxwebdev/xconfig`) merges, in order: `config.yaml`, `.env`, then
`DOWNLOADERBOT_*` environment variables. Unknown fields are rejected
(`WithDisallowUnknownFields`) and the struct is validated. Only
`DOWNLOADERBOT_TELEGRAM_BOT_API_TOKEN` is required. `BROWSER_BIN` (read directly by
`pkg/browser`, not through the config struct) points go-rod at a system Chromium — required in
the Alpine image, which sets it to `/usr/bin/chromium-browser`.

## Architecture

Startup (`cmd/app/start.go`) builds a `tkcrm/mx` launcher and registers three services: pingpong,
the gRPC transport, and the Telegram service. The parser service is a plain dependency of both
entry points, not a runnable service.

```
Telegram handler ─┐
                  ├─> parser.Service ─> extractor.Registry ─> Extractor ─> models.Media
gRPC BotService  ─┘                                                             │
                                                            internal/media.Loader (download)
```

### Extractor registry (`pkg/extractor`)

Every source implements `Extractor{ Name(); Hosts(); Extract(ctx, url) }`. `init.go` registers
them into the process-wide `DefaultRegistry` at package init; the registry indexes by host and
resolves a link with `GetByURL`. **Hosts are normalized** — `www.`, `m.` and `mobile.` prefixes
are stripped — so extractors list only the canonical host (`instagram.com`), but _must_ list
short-link domains explicitly (`vt.tiktok.com`, `vm.tiktok.com`, `b23.tv`, ...).

Four registration groups:

- `instagram` — custom, browser-based.
- `youtube` — `kkdai/youtube`; returns one item per quality/format rather than a single file.
- `tiktok` — custom, browser-based. **Deliberately not mapped in the lux host table** — lux's
  TikTok extractor is blocked by anti-bot.
- `lux` — one `lux.Extractor` per site, built from the `hostToSite` map. Adding a lux-backed site
  = add the blank import, the `hostToSite` entry, and the `siteToSource` entry.

Registration panics on duplicate name or host, so a collision fails at startup, not at runtime.

### Headless browser (`pkg/browser`)

One lazily-launched, process-wide `Manager` (`browser.Default()`), pre-warmed in a goroutine at
startup and closed on shutdown. `Load` opens a page, **blocks image/media/font/stylesheet
requests** (the extractors only need the server-rendered JSON) and snapshots the HTML.

The latency-critical part is `WithReady(pred)`: instead of waiting for the load event plus a
settle delay, `Load` polls the rendered HTML and returns the moment the predicate matches. Ready
predicates must match a _value-bearing_ pattern (reuse the extraction regex), not a bare key —
TikTok short links render an interstitial containing `"playAddr":""`, and Instagram pages mention
the keys before hydration. Both would otherwise yield an unparsable snapshot.

Extraction itself is regex-over-embedded-JSON (`pkg/instagram/browser.go`, `pkg/tiktok`), not DOM
queries. `pkg/instagram/browser.go` has non-obvious invariants worth preserving: a post page also
embeds _other_ posts (suggested reels), so the caption is anchored to the requested shortcode
rather than taken from the first `"caption"` on the page; and dimension pairs are parsed
order-agnostically because reels emit `original_height` before `original_width` (getting this
wrong makes Telegram render portrait video as a square).

### Media download (`internal/media`)

`Loader` is the single place that knows source-specific download requirements. Its distinction
drives behavior across all three consumers:

- `DirectURL(item)` returns `("", false)` when the item carries `DownloadHeaders` (TikTok needs
  visit cookies + a `tiktok.com` referer). Such items **cannot** be offered as Telegram inline
  results or fetched by gRPC clients — see "Known limitations" in [README.md](README.md).
- `Open(ctx, item)` streams the body with those headers applied — the direct-message path.
- `ContentLength(ctx, item)` does a HEAD so the 50MB Telegram cap can be checked without
  downloading.

### Telegram handlers (`internal/services/telegram/handler.go`)

Three handlers, all wrapped in `handler.recover` so a panic doesn't kill the polling loop.
Chat (`OnText`) and inline (`OnQuery`) share `fetchMedia`, differing only in budget: chat gets a
5-minute context and 2s retry delay, inline gets 10s and 1s because Telegram times inline queries
out. YouTube takes a separate path (`processYoutube`) that posts a formatted list of
quality/download links instead of uploading bytes, and is rejected outright in inline mode.

Two recurring gotchas encoded here: media over `maxFileSize` (50MB) falls back to a download-link
message in both chat and inline; and captions are arbitrary user text sent **without**
`ModeMarkdown` — an unbalanced `*` or `[` makes Telegram reject the whole message with a 400.

Rate limiting is in-memory, 10 requests/minute per chat ID (`internal/limiter`, `ulule/limiter`).

## Notable decisions

Instagram Stories support was evaluated and **declined** — see
[docs/instagram-stories.md](docs/instagram-stories.md). Stories need an authenticated session that
inevitably hits human-only challenges, which conflicts with the project's "runs unattended"
requirement. Don't re-add it without re-reading that doc.
