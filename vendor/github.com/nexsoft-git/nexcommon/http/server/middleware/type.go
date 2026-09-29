package middleware

import (
	"net/http"

	"github.com/nexsoft-git/nexcommon/server/metrics"
)

type MetricMiddleware interface {
	SupportMetrics(metrics metrics.Metrics)

	Serve(next http.Handler) http.Handler
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewLoggingResponseWriter(w http.ResponseWriter) *loggingResponseWriter {
	return &loggingResponseWriter{w, http.StatusOK}
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
