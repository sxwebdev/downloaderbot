package instagram_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/sxwebdev/downloaderbot/pkg/instagram"
)

func TestGetPostWithCode(t *testing.T) {
	tests := []struct {
		name string
		link string
	}{
		{
			name: "reels",
			link: "https://www.instagram.com/reel/CzBjgFiISfF/?igshid=MzRlODBiNWFlZA==",
		},
		{
			name: "photos",
			link: "https://www.instagram.com/p/Cy3lCHntbF7/?igshid=MzRlODBiNWFlZA==",
		},
		{
			name: "videos",
			link: "https://www.instagram.com/p/C0GixQTodKU/?igshid=MzRlODBiNWFlZA==",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			code, err := instagram.ExtractShortcodeFromLink(tc.link)
			if err != nil {
				t.Fatal(err)
			}

			resp, err := instagram.GetPostWithCode(code)
			if err != nil {
				t.Fatal(err)
			}

			spew.Dump(resp)

			// t.Log(resp.Url)

			for _, item := range resp.Items {
				if item.IsVideo {
					httpClient := &http.Client{Timeout: 5 * time.Second}
					response, err := httpClient.Head(item.Url)
					if err != nil {
						t.Fatal(err)
					}

					fmt.Printf("Content-Length: %d \n", response.ContentLength)
				}
				//t.Log(item.Url)
			}
		})
	}
}
