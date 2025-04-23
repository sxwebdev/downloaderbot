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
)

func init() {
	prometheus.MustRegister(InlineRequests)
	prometheus.MustRegister(PrivateMessageRequests)
}
