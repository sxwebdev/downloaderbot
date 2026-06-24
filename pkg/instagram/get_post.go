// This code is borrowed from this repository with small changes
// https://github.com/omegaatt36/instagramrobot
package instagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/sxwebdev/colly/v2"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/instagram/response"
)

const (
	igUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	igBaseURL   = "https://www.instagram.com"
)

// lsdTokenRe extracts the fresh "lsd" token embedded in any Instagram web page.
var lsdTokenRe = regexp.MustCompile(`"LSD",\[\],\{"token":"([^"]+)"`)

// igSession holds a fresh anonymous Instagram web session (lsd token + cookies)
// used to sign GraphQL requests. It is cached and reused across requests, and
// only rebuilt when a request fails (see getSession).
type igSession struct {
	lsd       string
	csrftoken string
	client    *http.Client // backed by a cookie jar so csrftoken/mid/datr persist
}

var (
	sessionMu  sync.Mutex
	curSession *igSession
)

// errSessionInvalid marks a GraphQL failure caused by a rejected/expired session
// signature (as opposed to a legitimate content response). Only these failures
// trigger a session refresh + retry.
var errSessionInvalid = errors.New("instagram session invalid")

// getSession returns the cached session, building a fresh one when none exists
// or forceRefresh is set.
func getSession(ctx context.Context, forceRefresh bool) (*igSession, error) {
	sessionMu.Lock()
	defer sessionMu.Unlock()

	if curSession != nil && !forceRefresh {
		return curSession, nil
	}

	sess, err := fetchSession(ctx)
	if err != nil {
		return nil, err
	}

	curSession = sess
	return sess, nil
}

// fetchSession builds a new anonymous session by loading an Instagram web page
// and harvesting the lsd token together with the csrftoken/mid/datr cookies.
func fetchSession(ctx context.Context) (*igSession, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: util.DefaultTransport(),
		Jar:       jar,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, igBaseURL+"/", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("user-agent", igUserAgent)
	req.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("accept-language", "en-US,en;q=0.9")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, err
	}

	match := lsdTokenRe.FindSubmatch(body)
	if len(match) < 2 {
		return nil, errors.New("could not extract lsd token from instagram page")
	}

	baseURL, _ := url.Parse(igBaseURL)
	var csrftoken string
	for _, c := range jar.Cookies(baseURL) {
		if c.Name == "csrftoken" {
			csrftoken = c.Value
			break
		}
	}

	return &igSession{
		lsd:       string(match[1]),
		csrftoken: csrftoken,
		client:    client,
	}, nil
}

// GetPostWithCode lets you to get information about specific Instagram post
// by providing its unique shortcode
func GetPostWithCode(ctx context.Context, code string) (*models.Media, error) {
	if code == "" {
		return nil, fmt.Errorf("empty code")
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if media, err := gqlRequest(ctx, code); err == nil && media != nil {
		return media, nil
	}

	if media, err := embedRequest(ctx, code); err == nil && media != nil {
		return media, nil
	} else if err != nil {
		return nil, err
	}

	return nil, errors.New("failed to fetch the post\nthe page might be \"private\", or\nthe link is completely wrong")
}

func embedRequest(ctx context.Context, code string) (*models.Media, error) {
	URL := fmt.Sprintf("https://www.instagram.com/p/%s/embed/captioned/", code)

	var embeddedMediaImage string
	var caption string
	embedResponse := response.EmbedResponse{}

	errChan := make(chan error, 1)
	go func() {
		collector := colly.NewCollector()
		collector.SetClient(util.DefaultHttpClient())

		collector.OnHTML("img.EmbeddedMediaImage", func(e *colly.HTMLElement) {
			embeddedMediaImage = e.Attr("src")
		})

		collector.OnHTML("div[class=Caption]", func(e *colly.HTMLElement) {
			r := regexp.MustCompile(`.*</a><br/><br/>(.*)<div class="CaptionComments">.*`)
			match := r.FindStringSubmatch(fmt.Sprint(e.DOM.Html()))
			if len(match) > 0 {
				caption = match[1]
			}
		})

		collector.OnHTML("video", func(e *colly.HTMLElement) {
			embeddedMediaImage = e.Attr("src")
		})

		collector.OnHTML("script", func(e *colly.HTMLElement) {
			r := regexp.MustCompile(`\\\"gql_data\\\":([\s\S]*)\}\"\}\]\]\,\[\"NavigationMetrics`)
			match := r.FindStringSubmatch(e.Text)

			if len(match) < 2 {
				return
			}

			s := strings.ReplaceAll(match[1], `\"`, `"`)
			s = strings.ReplaceAll(s, `\\/`, `/`)
			s = strings.ReplaceAll(s, `\\`, `\`)

			errChan <- json.Unmarshal([]byte(s), &embedResponse)
		})

		collector.OnRequest(func(r *colly.Request) {
			r.Headers.Set("Accept", "*/*")
			r.Headers.Set("Host", "www.instagram.com")
			r.Headers.Set("Referer", "https://www.instagram.com/")
			r.Headers.Set("DNT", "1")
			r.Headers.Set("Sec-Fetch-Dest", "document")
			r.Headers.Set("Sec-Fetch-Mode", "navigate")
			r.Headers.Set("Sec-Fetch-Site", "same-origin")
			r.Headers.Set("User-Agent", browser.Chrome())
		})

		errChan <- collector.Visit(URL)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("failed request by context")
	}

	if !embedResponse.IsEmpty() {
		resp := models.FromEmbedResponse(embedResponse)
		return &resp, nil
	}

	if embeddedMediaImage != "" {
		return &models.Media{
			Caption: caption,
			Url:     embeddedMediaImage,
			Items: []*models.MediaItem{
				{
					Type: models.MediaTypePhoto,
					Url:  embeddedMediaImage,
				},
			},
		}, nil
	}

	return nil, errors.New("embed response empty")
}

func gqlRequest(ctx context.Context, code string) (*models.Media, error) {
	sess, err := getSession(ctx, false)
	if err != nil {
		return nil, err
	}

	media, err := doGQLRequest(ctx, sess, code)
	if err == nil || !errors.Is(err, errSessionInvalid) {
		return media, err
	}

	// The session was rejected (expired/rotated lsd token or stale cookies).
	// Rebuild it once and retry before giving up to the embed fallback.
	sess, refreshErr := getSession(ctx, true)
	if refreshErr != nil {
		return nil, refreshErr
	}

	return doGQLRequest(ctx, sess, code)
}

// doGQLRequest performs a single GraphQL post for the given shortcode using the
// provided session.
func doGQLRequest(ctx context.Context, sess *igSession, code string) (*models.Media, error) {
	url := "https://www.instagram.com/api/graphql"
	method := "POST"

	payload := strings.NewReader("av=0&__d=www&__user=0&__a=1&__req=3&__comet_req=7&dpr=2&__ccg=UNKNOWN&lsd=" + sess.lsd + "&fb_api_caller_class=RelayModern&fb_api_req_friendly_name=PolarisPostActionLoadPostQueryQuery&variables=%7B%22shortcode%22%3A%22" + code + "%22%2C%22fetch_comment_count%22%3A40%2C%22fetch_related_profile_media_count%22%3A3%2C%22parent_comment_count%22%3A24%2C%22child_comment_count%22%3A3%2C%22fetch_like_count%22%3A10%2C%22fetch_tagged_user_count%22%3Anull%2C%22fetch_preview_comment_count%22%3A2%2C%22has_threaded_comments%22%3Atrue%2C%22hoisted_comment_id%22%3Anull%2C%22hoisted_reply_id%22%3Anull%7D&server_timestamps=true&doc_id=10015901848480474")

	req, err := http.NewRequestWithContext(ctx, method, url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("authority", "www.instagram.com")
	req.Header.Add("accept", "*/*")
	req.Header.Add("accept-language", "en-US,en;q=0.9")
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("origin", "https://www.instagram.com")
	req.Header.Add("referer", "https://www.instagram.com/reel/"+code+"/")
	req.Header.Add("sec-ch-ua", "\"Google Chrome\";v=\"131\", \"Chromium\";v=\"131\", \"Not_A Brand\";v=\"24\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-platform", "\"macOS\"")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("user-agent", igUserAgent)
	req.Header.Add("x-asbd-id", "129477")
	req.Header.Add("x-csrftoken", sess.csrftoken)
	req.Header.Add("x-fb-friendly-name", "PolarisPostActionLoadPostQueryQuery")
	req.Header.Add("x-fb-lsd", sess.lsd)
	req.Header.Add("x-ig-app-id", "936619743392459")
	req.Header.Add("x-requested-with", "XMLHttpRequest")

	res, err := sess.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	gqlResp := new(response.GrpahSQLResponse)
	if err := json.Unmarshal(body, gqlResp); err != nil {
		// Instagram returns a non-JSON error envelope (prefixed with "for (;;);")
		// when the request signature is rejected, e.g. error 1357054.
		return nil, fmt.Errorf("%w: unexpected graphql response: %w", errSessionInvalid, err)
	}

	data := gqlResp.Data.XdtShortcodeMedia

	if data.Shortcode == "" {
		return nil, fmt.Errorf("%w: empty graphql response for shortcode %q", errSessionInvalid, code)
	}

	if !data.IsVideo {
		return nil, fmt.Errorf("response is not video")
	}

	resp := &models.Media{
		Title: data.Title,
		Items: []*models.MediaItem{
			{
				Shortcode: data.Shortcode,
				Type:      models.MediaTypeVideo,
				Url:       data.VideoURL,
				Width:     data.Dimensions.Width,
				Height:    data.Dimensions.Height,
			},
		},
	}

	if len(data.EdgeMediaToCaption.Edges) > 0 {
		edge := data.EdgeMediaToCaption.Edges[0]
		resp.Caption = edge.Node.Text
	}

	return resp, nil
}

// ExtractShortcodeFromLink will extract the media shortcode from a URL link or path
func ExtractShortcodeFromLink(link string) (string, error) {
	values := regexp.MustCompile(`(p|tv|reel|reels\/videos)\/([A-Za-z0-9-_]+)`).FindStringSubmatch(link)
	if len(values) != 3 {
		return "", errors.New("couldn't extract the media shortcode from the link")
	}
	return values[2], nil
}
