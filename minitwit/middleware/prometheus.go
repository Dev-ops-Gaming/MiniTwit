package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"path", "method", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"path", "method"},
	)

	httpResponseMessages = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_response_messages_total",
			Help: "Total number of HTTP responses by message type",
		},
		[]string{"status", "message_type"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpResponseMessages)
}

func RecordResponseMessage(status int, messageType string) {
	statusCode := strconv.Itoa(status)
	httpResponseMessages.WithLabelValues(statusCode, messageType).Inc()
}

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// wrap response writer
		ww := &responseWriterWrapper{ResponseWriter: w, statusCode: 200}

		next.ServeHTTP(ww, r)

		// Get path from route
		var path string
		route := mux.CurrentRoute(r)
		if route != nil {
			var err error
			path, err = route.GetPathTemplate()
			if err != nil {
				path = "unknown"
			}
		} else {
			path = "unknown"
		}

		duration := time.Since(start).Seconds()
		statusCode := strconv.Itoa(ww.statusCode)

		httpRequestsTotal.WithLabelValues(path, r.Method, statusCode).Inc()
		httpRequestDuration.WithLabelValues(path, r.Method).Observe(duration)
	})
}

type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (rww *responseWriterWrapper) WriteHeader(code int) {
	rww.statusCode = code
	rww.ResponseWriter.WriteHeader(code)
}
