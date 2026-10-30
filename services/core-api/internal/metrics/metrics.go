package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	HttpRequestsTotal   *prometheus.CounterVec
	HttpRequestDuration *prometheus.HistogramVec
}

func NewMetrics(serviceName string) *Metrics {
	return &Metrics{
		HttpRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: serviceName,
				Name:      "http_requests_total",
				Help:      "Total number of HTTP requests.",
			},
			[]string{"method", "path", "status"},
		),
		HttpRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: serviceName,
				Name:      "http_request_duration_seconds",
				Help:      "Duration of HTTP requests in seconds.",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"method", "path"},
		),
	}
}
