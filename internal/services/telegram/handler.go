package telegram

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/tkcrm/modules/pkg/limiter"
	"github.com/tkcrm/mx/logger"
	"gopkg.in/telebot.v3"
)

type handler struct {
	logger logger.Logger

	parserService *parser.Service
	lim           limiter.ILimiter

	bot *telebot.Bot
}

func newHandler(
	logger logger.Logger,
	parserService *parser.Service,
	lim limiter.ILimiter,
	bot *telebot.Bot,
) *handler {
	return &handler{
		logger:        logger,
		parserService: parserService,
		lim:           lim,
		bot:           bot,
	}
}

func (s *handler) Start(c telebot.Context) error {
	// Ignore channels and groups
	if c.Chat().Type != telebot.ChatPrivate {
		return nil
	}

	if err := c.Reply("Hello!"); err != nil {
		return fmt.Errorf("couldn't sent the start command response: %w", err)
	}

	return nil
}

func (s *handler) OnText(c telebot.Context) error {
	l := logger.With(
		s.logger,
		"chat_id", c.Message().Chat.ID,
	)

	l.Infof("user reached limits")

	// check limits
	if err := s.checkLimit(context.Background(), c.Chat().ID); err != nil {
		return replyError(c, "you have reached your request limits. come back later")
	}

	links := util.ExtractLinksFromString(c.Message().Text)

	// Send proper error if text has no link inside
	if len(links) != 1 {
		if c.Chat().Type != telebot.ChatPrivate {
			return nil
		}

		l.Error("Invalid command,\nPlease send the Instagram post link.")
		return replyError(c, "Invalid command,\nPlease send the Instagram post link.")
	}

	_, err := c.Bot().Reply(
		c.Message(),
		"⏳ Please wait a moment, downloading your data...",
		telebot.ModeMarkdown,
	)
	if err != nil {
		return fmt.Errorf("couldn't reply the Error, chat_id %d: %w", c.Chat().ID, err)
	}

	link := links[0]

	if err := s.processLink(link, c.Message()); err != nil {
		if c.Chat().Type != telebot.ChatPrivate {
			return nil
		}

		l.Error(err)
		return replyError(c, err.Error())
	}

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

	data, err := s.parserService.GetMedia(ctx, link)
	if err != nil {
		l.Errorf("failed to get media: %v", err)
		return nil
	}

	if len(data.Items) == 0 {
		return nil
	}

	results := make(telebot.Results, len(data.Items))
	for i, item := range data.Items {
		if item.IsVideo {
			result := &telebot.VideoResult{
				Title:       fmt.Sprintf("video-%d", i),
				Description: data.Caption,
				MIME:        "video/mp4",
				URL:         item.Url,
				ThumbURL:    item.Url,
			}

			results[i] = result
		} else {
			result := &telebot.PhotoResult{
				URL:      item.Url,
				ThumbURL: item.Url, // required for photos
			}

			results[i] = result
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
func (s *handler) processLink(link string, msg *telebot.Message) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	data, err := s.parserService.GetMedia(ctx, link)
	if err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	if len(data.Items) == 0 {
		return fmt.Errorf("empty data items")
	}

	if len(data.Items) == 1 {
		mediaItem := data.Items[0]
		if data.IsVideo {
			if _, err := s.bot.Send(msg.Chat, &telebot.Video{
				File: telebot.FromURL(mediaItem.Url),
			}); err != nil {
				return fmt.Errorf("couldn't send the single video: %w", err)
			}

			s.logger.Debugf("sent single video with short code [%v]", mediaItem.Shortcode)
		} else {
			if _, err := s.bot.Send(msg.Chat, &telebot.Photo{
				File: telebot.FromURL(mediaItem.Url),
			}); err != nil {
				return fmt.Errorf("couldn't send the single photo: %w", err)
			}

			s.logger.Debugf("sent single photo with short code [%v]", mediaItem.Shortcode)
		}

		return nil
	}

	_, err = s.bot.SendAlbum(msg.Chat, generateAlbumFromMedia(data.Items))
	if err != nil {
		return fmt.Errorf("couldn't send the nested media: %w", err)
	}

	return nil
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

func generateAlbumFromMedia(items []*models.MediaItem) telebot.Album {
	var album telebot.Album

	for _, media := range items {
		if media.IsVideo {
			album = append(album, &telebot.Video{
				File: telebot.FromURL(media.Url),
			})
		} else {
			album = append(album, &telebot.Photo{
				File: telebot.FromURL(media.Url),
			})
		}
	}

	return album
}

func replyError(c telebot.Context, text string) error {
	_, err := c.Bot().Reply(c.Message(), fmt.Sprintf("⚠️ *Oops, ERROR!*\n\n`%s`", text), telebot.ModeMarkdown)
	if err != nil {
		return fmt.Errorf("couldn't reply the Error, chat_id %d: %w", c.Chat().ID, err)
	}

	return nil
}
