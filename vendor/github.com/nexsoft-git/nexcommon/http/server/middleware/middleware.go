package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	xerrors "github.com/nexsoft-git/nexcommon/error"
	srv "github.com/nexsoft-git/nexcommon/server"
	"github.com/nexsoft-git/nexcommon/server/metrics"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
)

func NewHTTPMiddleware(
	validator http_validator.HTTPController,
	basic basic_validator.BasicValidator,
	formator xerrors.Formator,
) MetricMiddleware {
	return &httpMiddleware{
		validator: validator,
		formator:  formator,
	}
}

type httpMiddleware struct {
	metrics   metrics.Metrics
	validator http_validator.HTTPController
	formator  xerrors.Formator
}

func (m *httpMiddleware) SupportMetrics(metrics metrics.Metrics) {
	m.metrics = metrics
}

func (m httpMiddleware) Serve(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()

			srv.CORSOriginHandler(&w)

			if r.Method != http.MethodOptions {
				next.ServeHTTP(w, r)
			}

			statusCode := 200

			if r.URL.Path != "/metrics" && r.Method != http.MethodOptions {
				if m.metrics != nil {
					usedPath, _ := mux.CurrentRoute(r).GetPathTemplate()
					if usedPath == "" {
						usedPath = r.URL.Path
					}

					m.metrics.GetDefaultMetric().APIHist.WithLabelValues(
						usedPath,
						r.Method,
						strconv.Itoa(statusCode),
					).Observe(float64(time.Since(now).Seconds()))
				}
			}
		},
	)
}
