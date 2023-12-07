// This code is borrowed from this repository with small changes
// https://github.com/omegaatt36/instagramrobot
package instagram

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	var embedResponse = response.EmbedResponse{}

	errChan := make(chan error, 1)
	go func() {
		collector := colly.NewCollector()
		collector.SetClient(util.DefaultHttpClient())

		collector.OnHTML("img.EmbeddedMediaImage", func(e *colly.HTMLElement) {
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
			Url: embeddedMediaImage,
			Items: []*models.MediaItem{
				{
					Url: embeddedMediaImage,
				},
			},
		}, nil
	}

	// If every two methods have failed, then return an error
	return nil, errors.New("failed to fetch the post\nthe page might be \"private\", or\nthe link is completely wrong")
}

// ExtractShortcodeFromLink will extract the media shortcode from a URL link or path
func ExtractShortcodeFromLink(link string) (string, error) {
	values := regexp.MustCompile(`(p|tv|reel|reels\/videos)\/([A-Za-z0-9-_]+)`).FindStringSubmatch(link)
	if len(values) != 3 {
		return "", errors.New("couldn't extract the media shortcode from the link")
	}
	// return shortcode
	return values[2], nil
}
