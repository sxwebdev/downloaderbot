package instagram

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/browser"
)

// Regexes that pull the media out of the server-rendered JSON embedded in a
// logged-out Instagram post page. The page uses the mobile "items" format:
// video items expose "video_versions" (highest quality first), photo items
// expose "image_versions2".candidates, and both carry "original_width/height".
// Carousel (sidecar) posts wrap several such items in "carousel_media".
var (
	reVideoVersion = regexp.MustCompile(`"video_versions":\[\{"type":\d+,"url":"([^"]+)"`)
	reImageVersion = regexp.MustCompile(`"image_versions2":\{"candidates":\[\{"url":"([^"]+)"`)
	// Instagram emits the pair in either order — reels arrive as
	// "original_height":H,"original_width":W — so capture each field's name and
	// number rather than assuming width-first. Assuming width-first made the
	// regex miss entirely on reels: the bot then sent 0x0 and Telegram squeezed
	// the vertical video into a square.
	reDimensions = regexp.MustCompile(`"original_(width|height)":(\d+),"original_(width|height)":(\d+)`)
	reCaption    = regexp.MustCompile(`"caption":\{(?:[^{}]*?)"text":"((?:[^"\\]|\\.)*)"`)
)

const carouselKey = `"carousel_media":[`

// igUserAgent reuses the shared browser UA so the legacy HTTP fetcher
// (get_post.go) and the browser fetcher present the same identity.
const igUserAgent = browser.UserAgent

// BrowserFetcher retrieves posts by loading the post page in a real (headless)
// browser via the shared browser.Manager. Because the requests originate from a
// genuine browser, Instagram's anti-bot layer serves the actual post instead of
// a challenge page — which is why this works where APIFetcher gets error
// 1357054. The media URLs are read from the page's embedded JSON; no
// cookies/tokens are needed.
type BrowserFetcher struct {
	mgr *browser.Manager
}

// NewBrowserFetcher creates a browser-based fetcher using the shared browser.
func NewBrowserFetcher() *BrowserFetcher {
	return &BrowserFetcher{mgr: browser.Default()}
}

// GetPost implements Fetcher by loading the post page in a real browser and
// parsing the embedded media JSON. The "/p/" path serves both single posts and
// reels (Instagram resolves the media by shortcode regardless of the path
// segment), and also carousels and photos.
func (f *BrowserFetcher) GetPost(ctx context.Context, code string) (*models.Media, error) {
	res, err := f.mgr.Load(ctx, igBaseURL+"/p/"+code+"/", browser.WithReady(hasMediaJSON))
	if err != nil {
		return nil, err
	}
	return parseMediaFromHTML(res.HTML, code)
}

// hasMediaJSON reports whether the rendered page already carries the embedded
// media JSON we parse, letting the browser stop waiting as soon as it appears.
// It matches the same patterns itemFromBlock consumes (not bare substrings), so
// an early snapshot is guaranteed to hold an extractable media URL rather than a
// partially hydrated page that merely mentions the key.
func hasMediaJSON(html string) bool {
	return reVideoVersion.MatchString(html) || reImageVersion.MatchString(html)
}

// parseMediaFromHTML extracts the media items (single, or all carousel children)
// and the caption from the embedded JSON of a post page.
func parseMediaFromHTML(html, code string) (*models.Media, error) {
	media := &models.Media{Shortcode: code}

	if items := parseCarousel(html, code); len(items) > 0 {
		media.Items = items
	} else if item := itemFromBlock(html, code); item != nil {
		media.Items = append(media.Items, item)
	}

	if len(media.Items) == 0 {
		return nil, fmt.Errorf("no media found on page for shortcode %q", code)
	}

	media.Type = string(media.Items[0].Type)

	if c := reCaption.FindStringSubmatch(html); len(c) > 1 {
		media.Caption = util.JSONUnescape(c[1])
	}

	media.Url = media.Items[0].Url
	return media, nil
}

// parseCarousel returns one item per child of a "carousel_media" array, or nil
// when the post is not a carousel.
func parseCarousel(html, code string) []*models.MediaItem {
	idx := strings.Index(html, carouselKey)
	if idx < 0 {
		return nil
	}
	// Position of the opening '[' of the array.
	arr, ok := balancedJSON(html, idx+len(carouselKey)-1)
	if !ok {
		return nil
	}

	var items []*models.MediaItem
	for _, child := range topLevelObjects(arr) {
		if item := itemFromBlock(child, code); item != nil {
			items = append(items, item)
		}
	}
	return items
}

// itemFromBlock builds a single media item from a JSON block, preferring the
// video URL when the block describes a video.
func itemFromBlock(block, code string) *models.MediaItem {
	if loc := reVideoVersion.FindStringSubmatchIndex(block); len(loc) > 0 {
		item := &models.MediaItem{
			Shortcode: code,
			Type:      models.MediaTypeVideo,
			Url:       util.JSONUnescape(block[loc[2]:loc[3]]),
		}
		item.Width, item.Height = dimensionsForItem(block, loc[0])
		return item
	}

	if loc := reImageVersion.FindStringSubmatchIndex(block); len(loc) > 0 {
		item := &models.MediaItem{
			Shortcode: code,
			Type:      models.MediaTypePhoto,
			Url:       util.JSONUnescape(block[loc[2]:loc[3]]),
		}
		item.Width, item.Height = dimensionsForItem(block, loc[0])
		return item
	}

	return nil
}

// dimensionsForItem returns the original_width/original_height pair for the media
// item whose URL was matched at urlPos. Each block — a single post or one
// carousel child — carries at most one such pair in practice, so picking the
// pair nearest the URL just guards against any stray pair elsewhere in the
// block. Bare candidate "width"/"height" (square cover thumbnails) do not match
// reDimensions, so they never distort the result. Returns (0, 0) when no pair is
// present, which leaves Telegram to detect the size from the file.
func dimensionsForItem(block string, urlPos int) (width, height int) {
	matches := reDimensions.FindAllStringSubmatchIndex(block, -1)
	if len(matches) == 0 {
		return 0, 0
	}

	chosen := matches[0]
	bestDist := -1
	for _, m := range matches {
		dist := urlPos - m[0]
		if dist < 0 {
			dist = -dist
		}
		if bestDist < 0 || dist < bestDist {
			chosen, bestDist = m, dist
		}
	}

	return dimsFromMatch(block, chosen)
}

// dimsFromMatch reads a width/height pair from a reDimensions submatch, honoring
// whichever field ("width" or "height") came first.
func dimsFromMatch(block string, m []int) (width, height int) {
	firstVal, _ := strconv.Atoi(block[m[4]:m[5]])
	secondVal, _ := strconv.Atoi(block[m[8]:m[9]])
	if block[m[2]:m[3]] == "width" {
		return firstVal, secondVal
	}
	return secondVal, firstVal
}

// balancedJSON returns the balanced bracket span of s starting at index start
// (which must point at a '[' or '{'), honoring JSON string literals and escapes.
func balancedJSON(s string, start int) (string, bool) {
	if start < 0 || start >= len(s) {
		return "", false
	}
	open := s[start]
	var close byte
	switch open {
	case '[':
		close = ']'
	case '{':
		close = '}'
	default:
		return "", false
	}

	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}

// topLevelObjects splits a JSON array span into its top-level "{...}" object
// substrings.
func topLevelObjects(arr string) []string {
	var out []string
	for i := 0; i < len(arr); {
		if arr[i] == '{' {
			obj, ok := balancedJSON(arr, i)
			if !ok {
				break
			}
			out = append(out, obj)
			i += len(obj)
			continue
		}
		i++
	}
	return out
}
