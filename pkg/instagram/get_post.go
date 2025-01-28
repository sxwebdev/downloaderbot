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
	"regexp"
	"strings"
	"time"

	browser "github.com/EDDYCJY/fake-useragent"
	"github.com/sxwebdev/colly/v2"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/instagram/response"
)

// GetPostWithCode lets you to get information about specific Instagram post
// by providing its unique shortcode
func GetPostWithCode(ctx context.Context, code string) (*models.Media, error) {
	// validate media code
	if code == "" {
		return nil, fmt.Errorf("empty code")
	}

	URL := fmt.Sprintf("https://www.instagram.com/p/%s/embed/captioned/", code)

	var embeddedMediaImage string
	var caption string
	embedResponse := response.EmbedResponse{}

	// try to use graphql request
	if resp, err := gqlRequest(ctx, code); err == nil {
		return resp, nil
	}

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

	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	select {
	case err := <-errChan:
		if err != nil {
			return nil, err
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("failed request by context")
	}

	// If the method one which is JSON parsing didn't fail
	if !embedResponse.IsEmpty() {
		// Transform the Embed response and return
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

	// If every two methods have failed, then return an error
	return nil, errors.New("failed to fetch the post\nthe page might be \"private\", or\nthe link is completely wrong")
}

func gqlRequest(ctx context.Context, code string) (*models.Media, error) {
	url := "https://www.instagram.com/api/graphql"
	method := "POST"

	payload := strings.NewReader("av=0&__d=www&__user=0&__a=1&__req=3&__hs=19702.HYP%3Ainstagram_web_pkg.2.1..0.0&dpr=2&__ccg=UNKNOWN&__rev=1010329543&__s=ytibog%3Adaumdy%3A3lk1qh&__hsi=7311321296052053265&__dyn=7xeUjG1mxu1syUbFp60DU98nwgU29zEdEc8co2qwJw5ux609vCwjE1xoswIwuo2awlU-cw5Mx62G3i1ywOwv89k2C1Fwc60AEC7U2czXwae4UaEW2G1NwwwNwKwHw8Xxm16wUwtEvw4JwJwSyES1Twoob82ZwrUdUbGwmk1xwmo6O1FwlE6OFA6fxy4Ujw&__csr=gjhXlMxdaWXDamZbmF8ytrmBqGHXRBx2vyV4iQpGvKbCGiU-eLFoSHzqDyqzaKRKFm-ahuiqimXl7ypGjx2OeuqhuBDhHDyWDAgCGGdzEOciihElzUargG4FU01cGpE2W805eiw1S606EE25G44md40dbw1aCrc1txC0uG3VzE8Q2q0nK089w0adG&__comet_req=7&lsd=AVpQxgXKVKs&jazoest=21006&__spin_r=1010329543&__spin_b=trunk&__spin_t=1702299643&fb_api_caller_class=RelayModern&fb_api_req_friendly_name=PolarisPostActionLoadPostQueryQuery&variables=%7B%22shortcode%22%3A%22" + code + "%22%2C%22fetch_comment_count%22%3A40%2C%22fetch_related_profile_media_count%22%3A3%2C%22parent_comment_count%22%3A24%2C%22child_comment_count%22%3A3%2C%22fetch_like_count%22%3A10%2C%22fetch_tagged_user_count%22%3Anull%2C%22fetch_preview_comment_count%22%3A2%2C%22has_threaded_comments%22%3Atrue%2C%22hoisted_comment_id%22%3Anull%2C%22hoisted_reply_id%22%3Anull%7D&server_timestamps=true&doc_id=10015901848480474")

	client := &http.Client{}
	req, err := http.NewRequestWithContext(ctx, method, url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("authority", "www.instagram.com")
	req.Header.Add("accept", "*/*")
	req.Header.Add("accept-language", "ru")
	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	req.Header.Add("cookie", "csrftoken=4uEFj2FLgdivDIhwhQvWmf; mid=ZXbNoAAEAAEyW7_iU-3KhQ1e_P8O; ig_did=8447FB01-50CB-40B4-BF71-96FD60599770; datr=sAF3ZbqSNWefZVtcruJTpACc; csrftoken=FxH3VTv4mRviA8kqGgpU2B")
	req.Header.Add("dpr", "2")
	req.Header.Add("origin", "https://www.instagram.com")
	req.Header.Add("referer", "https://www.instagram.com/p/CzBjgFiISfF/")
	req.Header.Add("sec-ch-prefers-color-scheme", "dark")
	req.Header.Add("sec-ch-ua", "\"Google Chrome\";v=\"119\", \"Chromium\";v=\"119\", \"Not?A_Brand\";v=\"24\"")
	req.Header.Add("sec-ch-ua-full-version-list", "\"Google Chrome\";v=\"119.0.6045.199\", \"Chromium\";v=\"119.0.6045.199\", \"Not?A_Brand\";v=\"24.0.0.0\"")
	req.Header.Add("sec-ch-ua-mobile", "?0")
	req.Header.Add("sec-ch-ua-model", "\"\"")
	req.Header.Add("sec-ch-ua-platform", "\"macOS\"")
	req.Header.Add("sec-ch-ua-platform-version", "\"13.6.1\"")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36")
	req.Header.Add("viewport-width", "853")
	req.Header.Add("x-asbd-id", "129477")
	req.Header.Add("x-csrftoken", "4uEFj2FLgdivDIhwhQvWmf")
	req.Header.Add("x-fb-friendly-name", "PolarisPostActionLoadPostQueryQuery")
	req.Header.Add("x-fb-lsd", "AVpQxgXKVKs")
	req.Header.Add("x-ig-app-id", "936619743392459")

	res, err := client.Do(req)
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
		return nil, err
	}

	data := gqlResp.Data.XdtShortcodeMedia

	if !data.IsVideo {
		return nil, fmt.Errorf("reponse is not video")
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
