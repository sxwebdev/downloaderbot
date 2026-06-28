// Package tiktok extracts downloadable media from TikTok video links by loading
// the page in a real (headless) browser, which bypasses TikTok's anti-bot.
package tiktok

import (
	"context"
	"fmt"
	"regexp"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/browser"
)

// The playable URL is embedded in the page's __UNIVERSAL_DATA_FOR_REHYDRATION__
// JSON. playAddr is the primary playback URL; downloadAddr is a fallback.
var (
	rePlayAddr     = regexp.MustCompile(`"playAddr":"([^"]+)"`)
	reDownloadAddr = regexp.MustCompile(`"downloadAddr":"([^"]+)"`)
	reDesc         = regexp.MustCompile(`"desc":"((?:[^"\\]|\\.)*)"`)
)

// GetVideo loads a TikTok video page (short vt.tiktok.com / vm.tiktok.com links
// are followed automatically) and returns the playable media.
//
// TikTok CDN URLs return 403 unless the request carries the visit cookies plus a
// tiktok.com referer, so those are attached to the item as DownloadHeaders for
// internal/media.Loader.Open to use.
func GetVideo(ctx context.Context, link string) (*models.Media, error) {
	res, err := browser.Default().Load(ctx, link, browser.WithReady(hasPlaybackData))
	if err != nil {
		return nil, err
	}

	var url string
	if m := rePlayAddr.FindStringSubmatch(res.HTML); len(m) > 1 {
		url = util.JSONUnescape(m[1])
	} else if m := reDownloadAddr.FindStringSubmatch(res.HTML); len(m) > 1 {
		url = util.JSONUnescape(m[1])
	}
	if url == "" {
		return nil, fmt.Errorf("no video url found on tiktok page")
	}

	headers := map[string]string{
		"Referer":    "https://www.tiktok.com/",
		"User-Agent": browser.UserAgent,
	}
	if ck := res.CookieHeader(); ck != "" {
		headers["Cookie"] = ck
	}

	media := &models.Media{
		Source: models.MediaSourceTikTok,
		Type:   string(models.MediaTypeVideo),
		Url:    url,
		Items: []*models.MediaItem{
			{
				Type:            models.MediaTypeVideo,
				Url:             url,
				DownloadHeaders: headers,
			},
		},
	}

	if m := reDesc.FindStringSubmatch(res.HTML); len(m) > 1 {
		media.Caption = util.JSONUnescape(m[1])
	}

	return media, nil
}

// hasPlaybackData reports whether the page already carries a non-empty playback
// URL, so the browser can stop waiting early. It reuses the extraction regexes
// (which require a value) rather than matching the bare key: short links
// (vt.tiktok.com) first render a redirect interstitial that has the rehydration
// container — and can carry an empty "playAddr":"" — but not yet the real URL,
// and stopping there would yield a page the extractor can't parse.
func hasPlaybackData(html string) bool {
	return rePlayAddr.MatchString(html) || reDownloadAddr.MatchString(html)
}
