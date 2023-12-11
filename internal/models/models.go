package models

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/sxwebdev/downloaderbot/pkg/instagram/response"
)

// Owner is a single Instagram user who owns the Media
type Owner struct {
	Id                string `json:"id"`
	ProfilePictureURL string `json:"profile_pic_url"`
	Username          string `json:"username"`
	Followers         uint64 `json:"followers_count"`
	IsPrivate         bool   `json:"is_private"`
	IsVerified        bool   `json:"is_verified"`
}

// MediaItem contains information about the Instagram post
// which is similar to the Instagram Media struct
type MediaItem struct {
	Id                string    `json:"id"`
	Shortcode         string    `json:"shortcode"`
	Type              MediaType `json:"type"`
	VideoWithoutAudio bool      `json:"video_without_audio"`
	Url               string    `json:"url"`
	Quality           string    `json:"quality"`
	ContentLength     int64     `json:"content_length"`
	MimeType          string    `json:"mime_type"`
	Width             int       `json:"width"`
	Height            int       `json:"height"`
}

func (s MediaItem) GetMediaDataByURL() (io.Reader, error) {
	if s.Url == "" {
		return nil, fmt.Errorf("empty url")
	}

	resp, err := http.Get(s.Url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(data), nil
}

// Media which contains a single Instagram post
type Media struct {
	Source     MediaSource  `json:"source"`
	RequestUrl string       `json:"request_url"`
	Id         string       `json:"id"`
	Shortcode  string       `json:"shortcode"`
	Title      string       `json:"title"`
	Author     string       `json:"author"`
	Type       string       `json:"type"`
	Comments   uint64       `json:"comments_count"`
	Likes      uint64       `json:"likes_count"`
	Caption    string       `json:"caption"`
	Url        string       `json:"url"`
	Items      []*MediaItem `json:"items"`
	TakenAt    int64        `json:"taken_at"` // Timestamp
}

// FromEmbedResponse will automatically transforms the EmbedResponse to the Media
func FromEmbedResponse(embed response.EmbedResponse) Media {
	media := Media{
		Id:        embed.Media.Id,
		Shortcode: embed.Media.Shortcode,
		Type:      embed.Media.Type,
		Comments:  embed.Media.Comments.Count,
		Likes:     embed.Media.Likes.Count,
		Url:       embed.ExtractMediaURL(),
		TakenAt:   embed.Media.TakenAt.Unix(),
		Caption:   embed.GetCaption(),
	}

	for _, item := range embed.Media.SliderItems.Edges {
		mediaType := MediaTypePhoto
		if item.Node.IsVideo {
			mediaType = MediaTypeVideo
		}

		media.Items = append(media.Items, &MediaItem{
			Id:        item.Node.Id,
			Shortcode: item.Node.Shortcode,
			Type:      mediaType,
			Url:       item.Node.ExtractMediaURL(),
		})
	}

	return media
}
