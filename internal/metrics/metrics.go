package metrics

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	OutcomeSuccess = "success"
	OutcomeFailure = "failure"

	ReasonNone       = "none"
	ReasonTimeout    = "timeout"
	ReasonCanceled   = "canceled"
	ReasonOpen       = "open"
	ReasonRead       = "read"
	ReasonIncomplete = "incomplete"
	ReasonSizeLimit  = "size_limit"
	ReasonTelegram   = "telegram"
	ReasonOther      = "other"
)

var (
	InlineRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "inline_requests_total",
		Help: "Total number of requests",
	})

	PrivateMessageRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "private_message_requests_total",
		Help: "Total number of private message requests",
	})

	ExtractDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "extract_duration_seconds",
		Help:    "Duration of media extraction by source.",
		Buckets: prometheus.DefBuckets,
	}, []string{"source"})

	ExtractErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "extract_errors_total",
		Help: "Number of media extraction errors by source and kind.",
	}, []string{"source", "kind"})

	TelegramSendErrors = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "telegram_send_errors_total",
		Help: "Number of Telegram send errors by kind.",
	}, []string{"kind"})

	MediaSizeBytes = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "media_size_bytes",
		Help:    "Sizes of downloaded media items in bytes.",
		Buckets: prometheus.ExponentialBuckets(64*1024, 2, 12),
	})

	MediaExtractionAttempts = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_extraction_attempts_total",
		Help: "Number of media extraction attempts, including retries.",
	}, []string{"source", "outcome", "reason"})

	MediaExtractionRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_extraction_requests_total",
		Help: "Final outcomes of logical media extraction requests after retries.",
	}, []string{"source", "outcome", "reason"})

	MediaDownloads = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_downloads_total",
		Help: "Final outcomes of media objects downloaded by the bot.",
	}, []string{"source", "outcome", "reason"})

	MediaDownloadBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_download_bytes_total",
		Help: "Bytes actually read by the bot from upstream media responses, including partial downloads.",
	}, []string{"source"})

	MediaDownloadCompletedBytes = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "media_download_completed_bytes_total",
		Help: "Bytes read for media objects that the bot downloaded completely.",
	}, []string{"source"})

	TelegramDeliveries = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "telegram_deliveries_total",
		Help: "Final outcomes of media delivery operations to Telegram after retries.",
	}, []string{"source", "kind", "outcome", "reason"})
)

func init() {
	prometheus.MustRegister(
		InlineRequests,
		PrivateMessageRequests,
		ExtractDuration,
		ExtractErrors,
		TelegramSendErrors,
		MediaSizeBytes,
		MediaExtractionAttempts,
		MediaExtractionRequests,
		MediaDownloads,
		MediaDownloadBytes,
		MediaDownloadCompletedBytes,
		TelegramDeliveries,
		processActiveUsers,
	)
}

// ObserveExtractionAttempt records one extractor call. Calls made by retries
// are deliberately counted separately from the final logical request result.
func ObserveExtractionAttempt(source string, started time.Time, err error) {
	source = normalizeSource(source)
	ExtractDuration.WithLabelValues(source).Observe(time.Since(started).Seconds())

	outcome, reason := outcomeAndReason(err, ReasonOther)
	MediaExtractionAttempts.WithLabelValues(source, outcome, reason).Inc()
	if err != nil {
		ExtractErrors.WithLabelValues(source, reason).Inc()
	}
}

// ObserveExtractionRequest records the final result after all extraction
// retries for one logical caller request.
func ObserveExtractionRequest(source string, err error) {
	outcome, reason := outcomeAndReason(err, ReasonOther)
	MediaExtractionRequests.WithLabelValues(normalizeSource(source), outcome, reason).Inc()
}

// ObserveDownloadFailure records a media object that the bot could not or did
// not download. reason is normalized to a bounded enum before becoming a label.
func ObserveDownloadFailure(source, reason string) {
	MediaDownloads.WithLabelValues(
		normalizeSource(source),
		OutcomeFailure,
		normalizeReason(reason),
	).Inc()
}

// ObserveTelegramDelivery records the final Telegram API result after retries.
// Existing telegram_send_errors_total is maintained for compatibility.
func ObserveTelegramDelivery(source, kind string, err error) {
	outcome, reason := outcomeAndReason(err, ReasonTelegram)
	TelegramDeliveries.WithLabelValues(normalizeSource(source), kind, outcome, reason).Inc()
	if err != nil {
		TelegramSendErrors.WithLabelValues(kind).Inc()
	}
}

// TrackDownload wraps an upstream media body. It counts bytes as they are
// actually read and records success only after EOF. Closing before EOF is an
// incomplete download, even when the underlying Close succeeds.
func TrackDownload(source string, body io.ReadCloser) io.ReadCloser {
	return &downloadReadCloser{
		source: normalizeSource(source),
		body:   body,
	}
}

type downloadReadCloser struct {
	source string
	body   io.ReadCloser

	mu       sync.Mutex
	bytes    int64
	finished bool
}

func (r *downloadReadCloser) Read(p []byte) (int, error) {
	n, err := r.body.Read(p)
	if n > 0 {
		MediaDownloadBytes.WithLabelValues(r.source).Add(float64(n))
		r.mu.Lock()
		r.bytes += int64(n)
		r.mu.Unlock()
	}

	switch {
	case errors.Is(err, io.EOF):
		r.finish(OutcomeSuccess, ReasonNone)
	case err != nil:
		r.finish(OutcomeFailure, ReasonRead)
	}

	return n, err
}

func (r *downloadReadCloser) Close() error {
	err := r.body.Close()
	if err != nil {
		r.finish(OutcomeFailure, ReasonOther)
	} else {
		r.finish(OutcomeFailure, ReasonIncomplete)
	}
	return err
}

func (r *downloadReadCloser) finish(outcome, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.finished {
		return
	}
	r.finished = true

	MediaDownloads.WithLabelValues(r.source, outcome, reason).Inc()
	if outcome == OutcomeSuccess {
		MediaDownloadCompletedBytes.WithLabelValues(r.source).Add(float64(r.bytes))
	}
}

func outcomeAndReason(err error, fallbackReason string) (string, string) {
	if err == nil {
		return OutcomeSuccess, ReasonNone
	}

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return OutcomeFailure, ReasonTimeout
	case errors.Is(err, context.Canceled):
		return OutcomeFailure, ReasonCanceled
	default:
		return OutcomeFailure, normalizeReason(fallbackReason)
	}
}

func normalizeSource(source string) string {
	if source == "" {
		return "unknown"
	}
	return source
}

func normalizeReason(reason string) string {
	switch reason {
	case ReasonNone,
		ReasonTimeout,
		ReasonCanceled,
		ReasonOpen,
		ReasonRead,
		ReasonIncomplete,
		ReasonSizeLimit,
		ReasonTelegram,
		ReasonOther:
		return reason
	default:
		return ReasonOther
	}
}
