package youtube

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/kkdai/youtube/v2"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/tkcrm/modules/pkg/utils"
)

func GetVideoByID(ctx context.Context, id string) (*models.Media, error) {
	client := youtube.Client{}

	video, err := client.GetVideo(id)
	if err != nil {
		return nil, err
	}

	formats := utils.FilterArray(video.Formats, func(v youtube.Format) bool {
		if strings.Contains(v.MimeType, "audio") {
			return true
		}

		if v.AudioChannels > 0 {
			return true
		}

		if slices.Contains([]string{"1080p", "1440p", "2160p"}, v.QualityLabel) {
			return true
		}

		return false
	})

	if len(formats) == 0 {
		return nil, fmt.Errorf("empty formats list")
	}

	resp := &models.Media{
		Title:   video.Title,
		Caption: video.Description,
		Items:   make([]*models.MediaItem, len(formats)),
	}

	if len(video.Thumbnails) > 0 {
		resp.Url = video.Thumbnails[len(video.Thumbnails)-1].URL
	}

	for index, format := range formats {
		itemType := models.MediaTypeVideo
		if strings.Contains(format.MimeType, "audio") {
			itemType = models.MediaTypeAudio
		}

		resp.Items[index] = &models.MediaItem{
			Id:                strconv.Itoa(index),
			Type:              itemType,
			Url:               format.URL,
			Quality:           format.QualityLabel,
			ContentLength:     format.ContentLength,
			MimeType:          format.MimeType,
			VideoWithoutAudio: format.AudioChannels == 0,
		}
	}

	return resp, nil
}

func ExtractShortcodeFromLink(link string) (string, error) {
	return youtube.ExtractVideoID(link)
}
