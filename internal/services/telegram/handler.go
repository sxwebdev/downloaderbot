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

// maxFileSize is the Telegram Bot API limit for a file the bot uploads itself
// via multipart/form-data — the chat path, which streams the bytes.
const maxFileSize = 50 * 1024 * 1024

// maxURLFileSize is the Telegram Bot API limit for content Telegram fetches from
// a URL on the bot's behalf: "5 MB max size for photos and 20 MB max for other
// types of content". Inline results can only reference a URL, so this — not the
// 50MB upload cap — is what bounds them. Offering a larger video inline makes
// Telegram silently fail to fetch it and the result never appears for the user.
const maxURLFileSize = 20 * 1024 * 1024

// processStats captures timing/attempt metrics of handling a single link.
type processStats struct {
	FetchDuration time.Duration // time spent fetching media (extraction + retries)
	Attempts      int           // number of fetch attempts performed
}

// requestKind labels where a request originated, for structured logs.
type requestKind string

const (
	kindChat   requestKind = "chat"
	kindInline requestKind = "inline"
)

// requestLogger builds the per-request logger shared by the chat and inline
// handlers, tagging every line with the request kind and the user/chat id.
func (s *handler) requestLogger(kind requestKind, chatID int64) logger.Logger {
	return logger.With(s.logger, "type", string(kind), "chat_id", chatID)
}

// logResult emits the standard completion log for a handled link — success or
// error — with timing and attempt stats. Shared so chat and inline produce
// identical output.
func logResult(l logger.Logger, link string, start time.Time, stats processStats, err error) {
	fields := []any{
		"duration", time.Since(start).String(),
		"fetch_duration", stats.FetchDuration.String(),
		"attempts", stats.Attempts,
	}
	if err != nil {
		l.Errorw(err.Error(), fields...)
		return
	}
	l.Infow(fmt.Sprintf("successfully processed the link: %s", link), fields...)
}

type handler struct {
	logger logger.Logger
	config *config.Config

	parserService *parser.Service
	lim           *limiter.Limiter

	// loader resolves media URLs and sizes. Held as an interface rather than
	// reaching for media.Default() so tests can substitute a fake.
	loader media.Loader

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
		loader:        media.Default(),
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
	start := time.Now()

	l := s.requestLogger(kindChat, tgCtx.Message().Chat.ID)

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

	stats, err := s.processLink(tgCtx, link)
	if err != nil {
		if tgCtx.Chat().Type != telebot.ChatPrivate {
			return nil
		}

		logResult(l, link, start, stats, err)
		return replyError(tgCtx, err.Error())
	}

	logResult(l, link, start, stats, nil)

	return nil
}

func (s *handler) OnQuery(c telebot.Context) error {
	start := time.Now()

	l := s.requestLogger(kindInline, c.Query().Sender.ID)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// check limits
	if err := s.checkLimit(ctx, c.Query().Sender.ID); err != nil {
		l.Infof("user reached limits")
		return nil
	}

	links := util.ExtractLinksFromString(c.Query().Text)

	// Inline queries fire on every keystroke, so only act on a complete link.
	if len(links) != 1 {
		return nil
	}

	link := links[0]

	l.Infof("request from user: %s", link)

	linkInfo, err := s.parserService.GetLinkInfo(ctx, link)
	if err != nil {
		l.Warnf("get link info error: %s", err)
		return answerInlineError(c, "Couldn't process this link")
	}

	// YouTube inline queries are not supported due to large file sizes
	if linkInfo.MediaSource == models.MediaSourceYoutube {
		return answerInlineError(c, "YouTube is not supported in inline mode")
	}

	observeActiveUser(c)

	// Keep attempts low to stay within Telegram's inline query timeout.
	data, stats, err := s.fetchMedia(ctx, linkInfo, 3, time.Second)
	if err != nil {
		logResult(l, link, start, stats, err)
		return answerInlineError(c, "Failed to fetch media, please try again")
	}

	metrics.InlineRequests.Inc()

	description := truncateRunes(data.Caption, 1000)

	results := make(telebot.Results, 0, len(data.Items))
	for i, item := range data.Items {
		result, ok := s.inlineResultFor(ctx, item, i, description)
		if !ok {
			continue
		}

		// needed to set a unique string ID for each result
		result.SetResultID(strconv.Itoa(i))
		results = append(results, result)
	}

	// Nothing could be offered inline (e.g. TikTok items need download headers and
	// can't be referenced by URL) — tell the user instead of showing an empty list.
	if len(results) == 0 {
		l.Warnf("no inline-able results for link: %s", link)
		return answerInlineError(c, "This media can't be sent inline")
	}

	logResult(l, link, start, stats, nil)

	return c.Answer(&telebot.QueryResponse{
		Results:   results,
		CacheTime: 60, // a minute
	})
}

// inlineResultFor builds the inline result offered for a single media item.
// ok is false when the item cannot be offered inline at all, in which case the
// caller skips it.
func (s *handler) inlineResultFor(ctx context.Context, item *models.MediaItem, index int, description string) (telebot.Result, bool) {
	// Inline results can only reference a publicly fetchable URL (Telegram
	// downloads it itself). Items that require download headers (e.g. TikTok)
	// can't be offered inline — skip them. See README "Known limitations".
	directURL, ok := s.loader.DirectURL(item)
	if !ok {
		return nil, false
	}

	switch item.Type {
	case models.MediaTypeVideo:
		// Telegram fetches the URL itself and gives up above maxURLFileSize,
		// leaving a result that silently never sends. Offer a download link
		// instead — same fallback as the chat handler. An unknown size (the CDN
		// answered no HEAD) is not treated as too large: offering the video is
		// still the better guess.
		if size, err := s.loader.ContentLength(ctx, item); err == nil && size > maxURLFileSize {
			return tooLargeResult(directURL, maxURLFileSize), true
		}
		return &telebot.VideoResult{
			Title:       fmt.Sprintf("video-%d", index+1),
			Description: description,
			MIME:        "video/mp4",
			URL:         directURL,
			ThumbURL:    inlineThumbURL(item, directURL),
			Width:       item.Width,
			Height:      item.Height,
			// Without this Telegram shows the result with no length until the
			// client has downloaded the whole file.
			Duration: item.Duration,
		}, true
	case models.MediaTypePhoto:
		return &telebot.PhotoResult{
			URL:      directURL,
			ThumbURL: directURL, // required for photos
		}, true
	default:
		return nil, false
	}
}

// Gets list of links from user message text
// and processes each one of them one by one.
func (s *handler) processLink(tgCtx telebot.Context, link string) (processStats, error) {
	var stats processStats

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	linkInfo, err := s.parserService.GetLinkInfo(ctx, link)
	if err != nil {
		return stats, fmt.Errorf("get link info error: %w", err)
	}

	observeActiveUser(tgCtx)

	data, stats, err := s.fetchMedia(ctx, linkInfo, 3, 2*time.Second)
	if err != nil {
		return stats, err
	}

	// YouTube has special handling with quality options
	if data.Source == models.MediaSourceYoutube {
		return stats, s.processYoutube(tgCtx, data)
	}

	// All other sources use the generic media handler (like Instagram)
	return stats, s.processGenericMedia(ctx, tgCtx, data)
}

// fetchMedia fetches usable media for linkInfo, retrying transient empty or
// URL-less results, and reports timing/attempt stats. Shared by the chat and
// inline handlers, which differ only in how aggressively they may retry within
// their respective timeouts.
func (s *handler) fetchMedia(ctx context.Context, linkInfo parser.GetLinkInfoResponse, maxAttempts int, delay time.Duration) (*models.Media, processStats, error) {
	var stats processStats
	var data *models.Media

	fetchStart := time.Now()
	err := retry.New(
		retry.WithContext(ctx),
		retry.WithPolicy(retry.PolicyLinear),
		retry.WithMaxAttempts(maxAttempts),
		retry.WithDelay(delay),
	).Do(func() error {
		stats.Attempts++

		var err error
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
	})
	stats.FetchDuration = time.Since(fetchStart)
	metrics.ObserveExtractionRequest(string(linkInfo.MediaSource), err)
	if err != nil {
		return nil, stats, fmt.Errorf("failed to get media: %w", err)
	}

	return data, stats, nil
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

func observeActiveUser(c telebot.Context) {
	sender := c.Sender()
	if sender == nil || sender.IsBot {
		return
	}
	metrics.ObserveActiveUser(sender.ID)
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

	// Send title and caption if available. The caption is arbitrary user text
	// and must NOT be parsed as Markdown — stray/unbalanced markup (a lone `*`,
	// `_`, `[`, ...) makes Telegram reject the message with a 400 entity-parse
	// error, so it is sent as plain text.
	var captionText string
	if data.Title != "" {
		captionText = data.Title + "\n\n"
	}
	if data.Caption != "" {
		captionText += data.Caption
	}

	if captionText != "" {
		if err := retry.New().Do(func() error {
			_, err := s.bot.Reply(tgCtx.Message(), captionText)
			return err
		}); err != nil {
			return fmt.Errorf("send caption error: %w", err)
		}
	}

	return nil
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

// tooLargeText is the message shown when media exceeds the Telegram size limit
// that applies to the delivery path, pointing the user to the original download
// URL. Shared by the chat reply and the inline download-link result so the
// wording stays in sync; the limit differs between them (see maxURLFileSize).
func tooLargeText(sourceURL string, limit int64) string {
	mb := limit / 1024 / 1024
	return fmt.Sprintf("the size of your media file is more than %dMB.\ntelegram allows you to send files via bot up to %d MB\ntry to download it from [here](%s)", mb, mb, sourceURL)
}

// inlineThumbURL picks the JPEG cover Telegram requires for an inline video
// result. Telegram documents thumbnail_url as "JPEG only", so the media URL is
// only a last resort for sources that expose no cover frame.
func inlineThumbURL(item *models.MediaItem, directURL string) string {
	if item.ThumbnailUrl != "" {
		return item.ThumbnailUrl
	}
	return directURL
}

// answerInlineError shows the user that the inline request failed instead of
// returning nothing, which looks like the bot is stuck. The query is answered
// with a single article describing the error; a short cache time keeps a later
// retry from being blocked.
func answerInlineError(c telebot.Context, message string) error {
	result := &telebot.ArticleResult{
		Title:       "Error",
		Description: message,
	}
	result.SetContent(&telebot.InputTextMessageContent{
		Text: fmt.Sprintf("⚠️ %s", message),
	})
	result.SetResultID("error")

	return c.Answer(&telebot.QueryResponse{
		Results:   telebot.Results{result},
		CacheTime: 5,
	})
}

// tooLargeResult builds an inline result that, when picked, sends the download
// link for a video too large for Telegram to fetch from a URL.
func tooLargeResult(sourceURL string, limit int64) telebot.Result {
	result := &telebot.ArticleResult{
		Title:       fmt.Sprintf("File is larger than %dMB", limit/1024/1024),
		Description: "Tap to get a download link",
	}
	result.SetContent(&telebot.InputTextMessageContent{
		Text:      tooLargeText(sourceURL, limit),
		ParseMode: telebot.ModeMarkdown,
	})
	return result
}

func (s *handler) replyTooLarge(tgCtx telebot.Context, sourceURL string) error {
	text := tooLargeText(sourceURL, maxFileSize)
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

// videoFromItem builds the video upload for a media item, shared by the single
// video and the album paths so both describe the file the same way.
//
// Telegram never probes a file a bot uploads: whatever it is not told stays
// unknown to the clients. Without Duration the video renders with no length
// until the user has downloaded the whole thing, and without Streaming it cannot
// start playing before that either.
func videoFromItem(item *models.MediaItem, body io.Reader) *telebot.Video {
	return &telebot.Video{
		File:      telebot.FromReader(body),
		Width:     item.Width,
		Height:    item.Height,
		MIME:      item.MimeType,
		Duration:  item.Duration,
		Streaming: true,
	}
}

func (s *handler) sendMediaContent(ctx context.Context, tgCtx telebot.Context, data *models.Media) error {
	source := string(data.Source)
	if len(data.Items) == 1 {
		mediaItem := data.Items[0]

		if mediaItem.ContentLength > maxFileSize {
			metrics.ObserveDownloadFailure(source, metrics.ReasonSizeLimit)
			return s.replyTooLarge(tgCtx, mediaItem.Url)
		}

		content, err := s.loader.Open(ctx, mediaItem)
		if err != nil {
			metrics.ObserveDownloadFailure(source, metrics.ReasonOpen)
			return err
		}

		// Open reports the real size from the response header — recheck before streaming
		if content.ContentLength > maxFileSize {
			_ = content.Body.Close()
			metrics.ObserveDownloadFailure(source, metrics.ReasonSizeLimit)
			return s.replyTooLarge(tgCtx, mediaItem.Url)
		}

		body := metrics.TrackDownload(source, content.Body)
		defer body.Close()

		if content.ContentLength > 0 {
			metrics.MediaSizeBytes.Observe(float64(content.ContentLength))
		}

		// handle video
		if mediaItem.Type.IsVideo() {
			sendErr := retry.New().Do(func() error {
				_, err := s.bot.Send(tgCtx.Message().Chat, videoFromItem(mediaItem, body))
				return err
			})
			metrics.ObserveTelegramDelivery(source, "video", sendErr)
			if sendErr != nil {
				return fmt.Errorf("couldn't send the single video: %w", sendErr)
			}
		}

		// handle photo
		if mediaItem.Type.IsPhoto() {
			sendErr := retry.New().Do(func() error {
				_, err := s.bot.Send(tgCtx.Message().Chat, &telebot.Photo{
					File:   telebot.FromReader(body),
					Width:  mediaItem.Width,
					Height: mediaItem.Height,
				})
				return err
			})
			metrics.ObserveTelegramDelivery(source, "photo", sendErr)
			if sendErr != nil {
				return fmt.Errorf("couldn't send the single photo: %w", sendErr)
			}
		}

		return nil
	}

	for chunk := range slices.Chunk(data.Items, 10) {
		album, err := generateAlbumFromMedia(ctx, s.loader, source, chunk)
		if err != nil {
			return fmt.Errorf("couldn't generate the album: %w", err)
		}

		sendErr := retry.New().Do(func() error {
			_, err := s.bot.SendAlbum(tgCtx.Message().Chat, album)
			return err
		})
		metrics.ObserveTelegramDelivery(source, "album", sendErr)
		if sendErr != nil {
			return fmt.Errorf("couldn't send the album: %w", sendErr)
		}
	}

	return nil
}

func generateAlbumFromMedia(ctx context.Context, loader media.Loader, source string, items []*models.MediaItem) (telebot.Album, error) {
	album := util.NewSliceWithLength[telebot.Inputtable](len(items))

	eg := errgroup.Group{}
	eg.SetLimit(5)

	for idx, item := range items {
		eg.Go(func() error {
			content, err := loader.Open(ctx, item)
			if err != nil {
				metrics.ObserveDownloadFailure(source, metrics.ReasonOpen)
				return err
			}

			// Guard before buffering the whole item into memory.
			if content.ContentLength > maxFileSize {
				_ = content.Body.Close()
				metrics.ObserveDownloadFailure(source, metrics.ReasonSizeLimit)
				return fmt.Errorf("media item exceeds %d bytes", int64(maxFileSize))
			}

			body := metrics.TrackDownload(source, content.Body)
			defer body.Close()
			data, err := io.ReadAll(body)
			if err != nil {
				return err
			}
			buf := bytes.NewReader(data)

			if item.Type.IsVideo() {
				album.AddToIndex(idx, videoFromItem(item, buf))
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
