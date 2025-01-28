package telegram

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"mime"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/downloaderbot/pkg/retry"
	"github.com/tkcrm/modules/pkg/limiter"
	"github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/mx/logger"
	"golang.org/x/sync/errgroup"
	"gopkg.in/telebot.v3"
)

type handler struct {
	logger logger.Logger
	config *config.Config

	parserService *parser.Service
	lim           limiter.ILimiter

	bot *telebot.Bot
}

func newHandler(
	logger logger.Logger,
	config *config.Config,
	parserService *parser.Service,
	lim limiter.ILimiter,
	bot *telebot.Bot,
) *handler {
	return &handler{
		logger:        logger,
		config:        config,
		parserService: parserService,
		lim:           lim,
		bot:           bot,
	}
}

func (s *handler) Start(tgCtx telebot.Context) error {
	// Ignore channels and groups
	if tgCtx.Chat().Type != telebot.ChatPrivate {
		return nil
	}

	if err := tgCtx.Reply("Hello!"); err != nil {
		return fmt.Errorf("couldn't sent the start command response: %w", err)
	}

	return nil
}

func (s *handler) OnText(tgCtx telebot.Context) error {
	l := logger.With(
		s.logger,
		"chat_id", tgCtx.Message().Chat.ID,
	)

	l.Infof("request from user: %s", tgCtx.Message().Text)

	// check limits
	if err := s.checkLimit(context.Background(), tgCtx.Chat().ID); err != nil {
		l.Infof("user reached limits")
		return replyError(tgCtx, "you have reached your request limits. come back later")
	}

	links := util.ExtractLinksFromString(tgCtx.Message().Text)

	// Send proper error if text has no link inside
	if len(links) != 1 {
		if tgCtx.Chat().Type != telebot.ChatPrivate {
			return nil
		}

		return replyError(tgCtx, "Invalid command\nPlease send the Instagram post link")
	}

	_, err := tgCtx.Bot().Reply(
		tgCtx.Message(),
		"⏳ Please wait a moment, downloading your data...",
		telebot.ModeMarkdown,
	)
	if err != nil {
		return fmt.Errorf("couldn't reply the Error, chat_id %d: %w", tgCtx.Chat().ID, err)
	}

	link := links[0]

	if err := s.processLink(tgCtx, link); err != nil {
		if tgCtx.Chat().Type != telebot.ChatPrivate {
			return nil
		}

		l.Error(err)
		return replyError(tgCtx, err.Error())
	}

	l.Infof("successfully processed the link: %s", link)

	return nil
}

func (s *handler) OnQuery(c telebot.Context) error {
	l := logger.With(
		s.logger,
		"chat_id", c.Query().Sender.ID,
	)

	// check limits
	if err := s.checkLimit(context.Background(), c.Query().Sender.ID); err != nil {
		l.Infof("user reached limits")
		return nil
	}

	links := util.ExtractLinksFromString(c.Query().Text)

	if len(links) != 1 {
		return nil
	}

	link := links[0]

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	linkInfo, err := s.parserService.GetLinkInfo(link)
	if err != nil {
		l.Warnf("get link info error: %s", err)
		return fmt.Errorf("get link info error: %w", err)
	}

	if linkInfo.MediaSource == models.MediaSourceYoutube {
		return nil
	}

	data, err := s.parserService.GetMedia(ctx, linkInfo)
	if err != nil {
		l.Errorf("failed to get media: %v", err)
		return nil
	}

	data.Items = lo.Filter(data.Items, func(v *models.MediaItem, idx int) bool {
		return v.Url != ""
	})

	if len(data.Items) == 0 {
		return nil
	}

	results := make(telebot.Results, len(data.Items))
	for i, item := range data.Items {
		switch item.Type {
		case models.MediaTypeVideo:
			result := &telebot.VideoResult{
				Title:       fmt.Sprintf("video-%d", i+1),
				Description: data.Caption,
				MIME:        "video/mp4",
				URL:         item.Url,
				ThumbURL:    item.Url,
				Width:       item.Width,
				Height:      item.Height,
			}

			results[i] = result
		case models.MediaTypePhoto:
			result := &telebot.PhotoResult{
				URL:      item.Url,
				ThumbURL: item.Url, // required for photos
			}

			results[i] = result
		default:
			continue
		}

		// needed to set a unique string ID for each result
		results[i].SetResultID(strconv.Itoa(i))
	}

	return c.Answer(&telebot.QueryResponse{
		Results:   results,
		CacheTime: 60, // a minute
	})
}

// Gets list of links from user message text
// and processes each one of them one by one.
func (s *handler) processLink(tgCtx telebot.Context, link string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	linkInfo, err := s.parserService.GetLinkInfo(link)
	if err != nil {
		return fmt.Errorf("get link info error: %w", err)
	}

	data, err := s.parserService.GetMedia(ctx, linkInfo)
	if err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	if len(data.Items) == 0 {
		return fmt.Errorf("empty data items")
	}

	switch data.Source {
	case models.MediaSourceYoutube:
		return s.processYoutube(tgCtx, data)
	case models.MediaSourceInstagram:
		return s.processInstagram(tgCtx, data)
	default:
		return fmt.Errorf("unsupported media source: %s", data.Source)
	}
}

func (s *handler) checkLimit(ctx context.Context, chatID int64) error {
	// get limiter
	lm, err := s.lim.GetService(ServiceName)
	if err != nil {
		return err
	}

	// get service limit stats
	lmStats, err := lm.Get(ctx, limiter.WithCacheKey(strconv.Itoa(int(chatID))))
	if err != nil {
		return err
	}

	// check limit
	if lmStats.Reached {
		return fmt.Errorf("%s rate limit is reached", ServiceName)
	}

	return nil
}

func replyError(c telebot.Context, text string) error {
	_, err := c.Bot().Reply(c.Message(), fmt.Sprintf("⚠️ *Oops, ERROR!*\n\n`%s`", text), telebot.ModeMarkdown)
	if err != nil {
		return fmt.Errorf("couldn't reply the Error, chat_id %d: %w", c.Chat().ID, err)
	}

	return nil
}

// replyText - send text message to user
func replyText(tgCtx telebot.Context, text string) error {
	// send chunked messages if length more than 4096
	if len(text) <= 4096 {
		if _, err := tgCtx.Bot().Send(tgCtx.Message().Chat, text, telebot.ModeMarkdown); err != nil {
			return fmt.Errorf("couldn't send text message: %w", err)
		}

		return nil
	}

	buf := bufio.NewScanner(strings.NewReader(text))
	writer := bytes.NewBuffer([]byte{})

	for buf.Scan() {
		newLine := buf.Text()
		if len(newLine)+writer.Len() > 4096 {
			if _, err := tgCtx.Bot().Send(tgCtx.Message().Chat, writer.String(), telebot.ModeMarkdown); err != nil {
				return fmt.Errorf("couldn't send text message: %w", err)
			}
			writer.Reset()
		}
		writer.WriteString(newLine + "\n")
	}

	if writer.Len() > 0 {
		if _, err := tgCtx.Bot().Send(tgCtx.Message().Chat, writer.String(), telebot.ModeMarkdown); err != nil {
			return fmt.Errorf("couldn't send text message: %w", err)
		}
		writer.Reset()
	}

	return nil
}

func (s *handler) processYoutube(tgCtx telebot.Context, data *models.Media) error {
	// send thumbnail
	if data.Url != "" {
		if _, err := s.bot.Send(tgCtx.Message().Chat, &telebot.Photo{
			File: telebot.FromURL(data.Url),
		}, telebot.ModeMarkdown); err != nil {
			return fmt.Errorf("couldn't send text message: %w", err)
		}
	}

	var respText string
	if data.Title != "" {
		respText += "*" + data.Title + "*\n\n"
	}

	if data.Caption != "" {
		respText += data.Caption + "\n\n"
	}

	fnVideoFormatter := func(item *models.MediaItem) {
		exts, err := mime.ExtensionsByType(item.MimeType)
		if err != nil {
			return
		}

		downloadLink := item.Url

		if s.config.ProxyHttpEnabled {
			var ext string
			if len(exts) > 0 {
				ext = exts[len(exts)-1]
			}

			downloadLink, err = url.JoinPath(
				s.config.ProxyHttpBaseUrl,
				"download",
				item.Quality+"-video"+ext,
			)
			if err != nil {
				return
			}

			downloadLink += "?redirectUrl=" + url.QueryEscape(item.Url)
		}

		noAudioStr := ""
		if item.VideoWithoutAudio {
			noAudioStr = " 🔇 "
		}

		if item.ContentLength == 0 {
			respText += fmt.Sprintf(
				"🔹 *%s*%s [Download](%s)\n`(%s)`\n\n",
				item.Quality,
				noAudioStr,
				downloadLink,
				item.MimeType,
			)
		} else {
			respText += fmt.Sprintf(
				"🔹 *%s*%s [Download %.2fMB](%s)\n`(%s)`\n\n",
				item.Quality,
				noAudioStr,
				float64(item.ContentLength)/1024/1024,
				downloadLink,
				item.MimeType,
			)
		}
	}

	fnAudioFormatter := func(item *models.MediaItem) {
		respText += fmt.Sprintf(
			"🔸 %s [Download %.2fMB](%s) `(%s)`\n",
			item.Quality,
			float64(item.ContentLength)/1024/1024,
			item.Url,
			item.MimeType,
		)
	}

	videoItems := utils.FilterArray(data.Items, func(v *models.MediaItem) bool {
		return v.Type == "video"
	})

	audioItems := utils.FilterArray(data.Items, func(v *models.MediaItem) bool {
		return v.Type == "audio"
	})

	if len(videoItems) > 0 {
		respText += "🎥 *Video*\n\n"
		for _, item := range videoItems {
			fnVideoFormatter(item)
		}
		respText += "\n"
	}

	if len(audioItems) > 0 {
		respText += "🎶 *Audio*\n\n"
		for _, item := range audioItems {
			fnAudioFormatter(item)
		}
	}

	return replyText(tgCtx, respText)
}

func (s *handler) processInstagram(tgCtx telebot.Context, data *models.Media) error {
	// filter items
	data.Items = lo.Filter(data.Items, func(v *models.MediaItem, idx int) bool {
		return v.Url != ""
	})

	if len(data.Items) == 0 {
		return fmt.Errorf("empty data items")
	}

	if err := s.sendContentToInstagram(tgCtx, data); err != nil {
		return fmt.Errorf("couldn't send the content to Instagram: %w", err)
	}

	if data.Caption != "" {
		if err := retry.New().Do(func() error {
			_, err := s.bot.Reply(tgCtx.Message(), data.Caption, telebot.ModeHTML)
			return err
		}); err != nil {
			s.logger.Warnf("send caption with params error: %v", err)
		}
	}

	return nil
}

func (s *handler) sendContentToInstagram(tgCtx telebot.Context, data *models.Media) error {
	if len(data.Items) == 1 {
		mediaItem := data.Items[0]
		if mediaItem.ContentLength > 50*1024*1024 {
			text := fmt.Sprintf("the size of your media file is more than 50MB.\ntelegram allows you to send files via bot up to 50 MB\ntry to download it from [here](%s)", mediaItem.Url)
			if err := retry.New().Do(func() error {
				_, err := s.bot.Reply(tgCtx.Message(), text, telebot.ModeMarkdown)
				return err
			}); err != nil {
				s.logger.Error(err)
			}
			return nil
		}

		mediaData, err := mediaItem.GetMediaDataByURL()
		if err != nil {
			return err
		}

		// handle video
		if mediaItem.Type.IsVideo() {
			if err := retry.New().Do(func() error {
				_, err = s.bot.Send(tgCtx.Message().Chat, &telebot.Video{
					File:   telebot.FromReader(mediaData),
					Width:  mediaItem.Width,
					Height: mediaItem.Height,
					MIME:   mediaItem.MimeType,
				})
				return err
			}); err != nil {
				return fmt.Errorf("couldn't send the single video: %w", err)
			}
		}

		// handle photo
		if mediaItem.Type.IsPhoto() {
			if err := retry.New().Do(func() error {
				_, err := s.bot.Send(tgCtx.Message().Chat, &telebot.Photo{
					File:   telebot.FromReader(mediaData),
					Width:  mediaItem.Width,
					Height: mediaItem.Height,
				})
				return err
			}); err != nil {
				return fmt.Errorf("couldn't send the single photo: %w", err)
			}
		}

		return nil
	}

	for chunk := range slices.Chunk(data.Items, 10) {
		album, err := generateAlbumFromMedia(chunk)
		if err != nil {
			return fmt.Errorf("couldn't generate the album: %w", err)
		}

		if _, err := s.bot.SendAlbum(tgCtx.Message().Chat, album); err != nil {
			return err
		}
	}

	return nil
}

func generateAlbumFromMedia(items []*models.MediaItem) (telebot.Album, error) {
	album := util.NewSliceWithLength[telebot.Inputtable](len(items))

	eg := errgroup.Group{}

	for idx, media := range items {
		eg.Go(func() error {
			mediaData, err := media.GetMediaDataByURL()
			if err != nil {
				return err
			}

			if media.Type.IsVideo() {
				album.AddToIndex(idx, &telebot.Video{
					File:   telebot.FromReader(mediaData),
					Width:  media.Width,
					Height: media.Height,
					MIME:   media.MimeType,
				})
			} else {
				album.AddToIndex(idx, &telebot.Photo{
					File:   telebot.FromReader(mediaData),
					Width:  media.Width,
					Height: media.Height,
				})
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return album.GetAll(), nil
}
