package telegram

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/samber/lo"
	"github.com/sxwebdev/downloaderbot/internal/config"
	"github.com/sxwebdev/downloaderbot/internal/limiter"
	"github.com/sxwebdev/downloaderbot/internal/media"
	"github.com/sxwebdev/downloaderbot/internal/metrics"
	"github.com/sxwebdev/downloaderbot/internal/models"
	"github.com/sxwebdev/downloaderbot/internal/services/parser"
	"github.com/sxwebdev/downloaderbot/internal/util"
	"github.com/sxwebdev/xutils/retry"
	"github.com/tkcrm/modules/pkg/utils"
	"github.com/tkcrm/mx/logger"
	"golang.org/x/sync/errgroup"
	"gopkg.in/telebot.v3"
)

// maxFileSize is the Telegram Bot API upload limit for files sent by a bot.
const maxFileSize = 50 * 1024 * 1024

type handler struct {
	logger logger.Logger
	config *config.Config

	parserService *parser.Service
	lim           *limiter.Limiter

	bot *telebot.Bot
}

func newHandler(
	logger logger.Logger,
	config *config.Config,
	parserService *parser.Service,
	lim *limiter.Limiter,
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

// recover wraps a telebot handler with panic recovery so a panic in user code
// doesn't crash the long-polling loop.
func (s *handler) recover(name string, fn telebot.HandlerFunc) telebot.HandlerFunc {
	return func(tgCtx telebot.Context) (err error) {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Errorf("panic in handler %s: %v", name, r)
				err = fmt.Errorf("handler %s panicked: %v", name, r)
			}
		}()
		return fn(tgCtx)
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

	metrics.PrivateMessageRequests.Inc()

	l.Infof("request from user: %s", tgCtx.Message().Text)

	limCtx, limCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer limCancel()

	// check limits
	if err := s.checkLimit(limCtx, tgCtx.Chat().ID); err != nil {
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

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// check limits
	if err := s.checkLimit(ctx, c.Query().Sender.ID); err != nil {
		l.Infof("user reached limits")
		return nil
	}

	links := util.ExtractLinksFromString(c.Query().Text)

	if len(links) != 1 {
		return nil
	}

	link := links[0]

	linkInfo, err := s.parserService.GetLinkInfo(ctx, link)
	if err != nil {
		l.Warnf("get link info error: %s", err)
		return fmt.Errorf("get link info error: %w", err)
	}

	// YouTube inline queries are not supported due to large file sizes
	if linkInfo.MediaSource == models.MediaSourceYoutube {
		return nil
	}

	// Some sources transiently return empty or URL-less items, so retry the
	// fetch. Keep attempts low to stay within Telegram's inline query timeout.
	var data *models.Media
	if err := retry.New(
		retry.WithContext(ctx),
		retry.WithPolicy(retry.PolicyLinear),
		retry.WithMaxAttempts(3),
		retry.WithDelay(time.Second),
	).Do(func() error {
		data, err = s.parserService.GetMedia(ctx, linkInfo)
		if err != nil {
			return err
		}

		// keep only items with valid URLs
		data.Items = lo.Filter(data.Items, func(v *models.MediaItem, _ int) bool {
			return v.Url != ""
		})

		if len(data.Items) == 0 {
			return fmt.Errorf("empty data items")
		}

		return nil
	}); err != nil {
		l.Errorf("failed to get media: %v", err)
		return nil
	}

	metrics.InlineRequests.Inc()

	description := truncateRunes(data.Caption, 1000)

	results := make(telebot.Results, 0, len(data.Items))
	for i, item := range data.Items {
		// Inline results can only reference a publicly fetchable URL (Telegram
		// downloads it itself). Items that require download headers (e.g. TikTok)
		// can't be offered inline — skip them. See README "Known limitations".
		directURL, ok := media.Default().DirectURL(item)
		if !ok {
			continue
		}

		var result telebot.Result
		switch item.Type {
		case models.MediaTypeVideo:
			result = &telebot.VideoResult{
				Title:       fmt.Sprintf("video-%d", i+1),
				Description: description,
				MIME:        "video/mp4",
				URL:         directURL,
				ThumbURL:    directURL,
				Width:       item.Width,
				Height:      item.Height,
			}
		case models.MediaTypePhoto:
			result = &telebot.PhotoResult{
				URL:      directURL,
				ThumbURL: directURL, // required for photos
			}
		default:
			continue
		}

		// needed to set a unique string ID for each result
		result.SetResultID(strconv.Itoa(i))
		results = append(results, result)
	}

	return c.Answer(&telebot.QueryResponse{
		Results:   results,
		CacheTime: 60, // a minute
	})
}

// Gets list of links from user message text
// and processes each one of them one by one.
func (s *handler) processLink(tgCtx telebot.Context, link string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	linkInfo, err := s.parserService.GetLinkInfo(ctx, link)
	if err != nil {
		return fmt.Errorf("get link info error: %w", err)
	}

	// Some sources transiently return empty or URL-less items, so retry the
	// fetch until we get usable media items.
	var data *models.Media
	if err := retry.New(
		retry.WithContext(ctx),
		retry.WithPolicy(retry.PolicyLinear),
		retry.WithMaxAttempts(10),
		retry.WithDelay(2*time.Second),
	).Do(func() error {
		data, err = s.parserService.GetMedia(ctx, linkInfo)
		if err != nil {
			return err
		}

		// keep only items with valid URLs
		data.Items = lo.Filter(data.Items, func(v *models.MediaItem, _ int) bool {
			return v.Url != ""
		})

		if len(data.Items) == 0 {
			return fmt.Errorf("empty data items")
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to get media: %w", err)
	}

	// YouTube has special handling with quality options
	if data.Source == models.MediaSourceYoutube {
		return s.processYoutube(tgCtx, data)
	}

	// All other sources use the generic media handler (like Instagram)
	return s.processGenericMedia(ctx, tgCtx, data)
}

func (s *handler) checkLimit(ctx context.Context, chatID int64) error {
	return s.lim.Allow(ctx, strconv.Itoa(int(chatID)))
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
	if err := buf.Err(); err != nil {
		return fmt.Errorf("scan text: %w", err)
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
		downloadLink := item.Url

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

// processGenericMedia handles media from all sources (Instagram, TikTok, Twitter, etc.)
func (s *handler) processGenericMedia(ctx context.Context, tgCtx telebot.Context, data *models.Media) error {
	if err := s.sendMediaContent(ctx, tgCtx, data); err != nil {
		return fmt.Errorf("couldn't send the content: %w", err)
	}

	// Send title and caption if available
	var captionText string
	if data.Title != "" {
		captionText = "*" + escapeMarkdown(data.Title) + "*\n\n"
	}
	if data.Caption != "" {
		captionText += data.Caption
	}

	if captionText != "" {
		if err := retry.New().Do(func() error {
			_, err := s.bot.Reply(tgCtx.Message(), captionText, telebot.ModeMarkdown)
			return err
		}); err != nil {
			return fmt.Errorf("send caption error: %w", err)
		}
	}

	return nil
}

// escapeMarkdown escapes special Markdown characters
func escapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(text)
}

// truncateRunes returns text limited to maxRunes runes, appending an ellipsis if it was cut.
func truncateRunes(text string, maxRunes int) string {
	runes := []rune(text)
	if len(runes) <= maxRunes {
		return text
	}
	if maxRunes <= 1 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-1]) + "…"
}

func (s *handler) replyTooLarge(tgCtx telebot.Context, sourceURL string) error {
	text := fmt.Sprintf("the size of your media file is more than 50MB.\ntelegram allows you to send files via bot up to 50 MB\ntry to download it from [here](%s)", sourceURL)
	if err := retry.New().Do(func() error {
		_, err := s.bot.Reply(tgCtx.Message(), text, telebot.ModeMarkdown)
		return err
	}); err != nil {
		s.logger.Warnf("reply too-large markdown failed, falling back to plain reply: %v", err)
		if _, fallbackErr := s.bot.Reply(tgCtx.Message(), "file is larger than 50MB, telegram bots can't send it"); fallbackErr != nil {
			return fmt.Errorf("reply too-large failed: %w (after markdown error: %v)", fallbackErr, err)
		}
	}
	return nil
}

func (s *handler) sendMediaContent(ctx context.Context, tgCtx telebot.Context, data *models.Media) error {
	if len(data.Items) == 1 {
		mediaItem := data.Items[0]

		if mediaItem.ContentLength > maxFileSize {
			return s.replyTooLarge(tgCtx, mediaItem.Url)
		}

		content, err := media.Default().Open(ctx, mediaItem)
		if err != nil {
			return err
		}
		body := content.Body
		defer body.Close()

		// Open reports the real size from the response header — recheck before streaming
		if content.ContentLength > maxFileSize {
			return s.replyTooLarge(tgCtx, mediaItem.Url)
		}

		if content.ContentLength > 0 {
			metrics.MediaSizeBytes.Observe(float64(content.ContentLength))
		}

		// handle video
		if mediaItem.Type.IsVideo() {
			if err := retry.New().Do(func() error {
				_, err = s.bot.Send(tgCtx.Message().Chat, &telebot.Video{
					File:   telebot.FromReader(body),
					Width:  mediaItem.Width,
					Height: mediaItem.Height,
					MIME:   mediaItem.MimeType,
				})
				return err
			}); err != nil {
				metrics.TelegramSendErrors.WithLabelValues("video").Inc()
				return fmt.Errorf("couldn't send the single video: %w", err)
			}
		}

		// handle photo
		if mediaItem.Type.IsPhoto() {
			if err := retry.New().Do(func() error {
				_, err := s.bot.Send(tgCtx.Message().Chat, &telebot.Photo{
					File:   telebot.FromReader(body),
					Width:  mediaItem.Width,
					Height: mediaItem.Height,
				})
				return err
			}); err != nil {
				metrics.TelegramSendErrors.WithLabelValues("photo").Inc()
				return fmt.Errorf("couldn't send the single photo: %w", err)
			}
		}

		return nil
	}

	for chunk := range slices.Chunk(data.Items, 10) {
		album, err := generateAlbumFromMedia(ctx, chunk)
		if err != nil {
			return fmt.Errorf("couldn't generate the album: %w", err)
		}

		if err := retry.New().Do(func() error {
			_, err := s.bot.SendAlbum(tgCtx.Message().Chat, album)
			return err
		}); err != nil {
			metrics.TelegramSendErrors.WithLabelValues("album").Inc()
			return fmt.Errorf("couldn't send the album: %w", err)
		}
	}

	return nil
}

func generateAlbumFromMedia(ctx context.Context, items []*models.MediaItem) (telebot.Album, error) {
	album := util.NewSliceWithLength[telebot.Inputtable](len(items))

	eg := errgroup.Group{}
	eg.SetLimit(5)

	for idx, item := range items {
		eg.Go(func() error {
			content, err := media.Default().Open(ctx, item)
			if err != nil {
				return err
			}
			defer content.Body.Close()

			// Guard before buffering the whole item into memory.
			if content.ContentLength > maxFileSize {
				return fmt.Errorf("media item exceeds %d bytes", int64(maxFileSize))
			}

			data, err := io.ReadAll(content.Body)
			if err != nil {
				return err
			}
			buf := bytes.NewReader(data)

			if item.Type.IsVideo() {
				album.AddToIndex(idx, &telebot.Video{
					File:   telebot.FromReader(buf),
					Width:  item.Width,
					Height: item.Height,
					MIME:   item.MimeType,
				})
			} else {
				album.AddToIndex(idx, &telebot.Photo{
					File:   telebot.FromReader(buf),
					Width:  item.Width,
					Height: item.Height,
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
