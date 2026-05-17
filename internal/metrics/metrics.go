package metrics

import "github.com/prometheus/client_golang/prometheus"

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
)

func init() {
	prometheus.MustRegister(
		InlineRequests,
		PrivateMessageRequests,
		ExtractDuration,
		ExtractErrors,
		TelegramSendErrors,
		MediaSizeBytes,
	)
}
