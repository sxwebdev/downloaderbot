# Instagram Stories Support — Declined

**Status:** declined
**Date:** 2026-05-18

## Context

A request came in to add support for downloading Instagram Stories, e.g.:

```
https://www.instagram.com/stories/{username}/{story_id}
```

The bot already handles posts, reels, IGTV and `/reels/videos/` URLs via the
public, unauthenticated endpoints used in
[`pkg/instagram/get_post.go`](../pkg/instagram/get_post.go). Stories would be a
natural addition.

## Why it is hard

1. **Stories require an authenticated session.** Unlike posts/reels there is no
   public endpoint. Fetching the story tray requires `sessionid` and
   `ds_user_id` cookies from a logged-in Instagram account.
2. **Sessions expire.** Web sessions typically last from a few days to a couple
   of months. Mobile-API sessions live longer (months), but still expire.
3. **Instagram throws challenges that cannot be solved in code.** Sooner or
   later any account hits a checkpoint: 2FA code, email confirmation, CAPTCHA,
   or "was this you?" prompts. These are designed specifically to require a
   human, and no amount of clever code gets around them.

## Options considered

| Approach                                           | Engineering cost | Realistic uptime without manual intervention |
| -------------------------------------------------- | ---------------- | -------------------------------------------- |
| Manual cookie import + file hot-reload + 401 alert | Low              | 1–3 months between manual refreshes          |
| Username/password auto-login + cookie fallback     | Medium           | Days to weeks (challenges break the loop)    |
| Mobile-API login + persist device fingerprint      | High             | Weeks to months, periodic challenges remain  |
| Headless browser (chromedp / playwright)           | High             | No better than mobile-API; harder to ship    |
| Paid third-party API provider                      | $$$ per month    | Genuinely "lives on its own" (their problem) |

## Decision

The hard requirement for this project is **"the system must live on its own
without intervention"**. None of the self-hosted approaches above satisfy that
requirement honestly — Instagram's anti-automation makes the "no manual
intervention" guarantee impossible without paying a third party.

Rather than ship a feature that quietly breaks every few weeks and forces the
operator to fight CAPTCHAs, we are not building Instagram Stories support.

Posts, reels and IGTV continue to work as before, because their endpoints are
public and require no authentication.

## If we revisit this

Two paths are worth considering if the priorities change:

1. **Pay a third-party scraping API** (RapidAPI marketplace, Apify, etc.).
   Minimal code, a real "set and forget" SLA, but recurring cost and an
   external dependency.
2. **Mobile-API auto-login with Telegram alerting.** The bot logs in with
   username/password, persists a stable `device_id` / `phone_id` / `uuid`
   between restarts, re-logs on 401, and posts an alert to the bot owner's
   Telegram chat when it hits a challenge that needs human action. This needs
   a dedicated "technical" Instagram account without 2FA and accepts that the
   operator will resolve a challenge every 1–3 months at best.

Either path should be re-scoped as its own task, not bolted onto an existing
PR.
