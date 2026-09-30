// Package router owns the gorilla/mux router: one instance, created once, with
// the controller's routes hung off it.
package router

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	mdl "github.com/nexsoft-git/nexcommon/http/server/middleware"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/config"
)

var router *mux.Router

// InitHttpService creates the router, hands it to the controller, and installs
// the middleware.
//
// The controller has to be given the router before InitEndpoint runs, because
// HandleFunc registers against it - and the middleware has to be installed
// before any route is served, which is why both happen here rather than inside
// the controller.
func InitHttpService(
	config config.Configuration,
	metricMiddleware mdl.MetricMiddleware,
	httpValidator *http_validator.HTTPController,
	healthChecker health.HealthEndpoint,
) {
	router = mux.NewRouter()
	httpValidator.Router(router)

	router.HandleFunc("/health", healthChecker.CheckHealthConnection).Methods(http.MethodGet)

	router.Use(metricMiddleware.Serve)
}

// StartService generates the swagger document and blocks serving.
//
// GenerateSwaggerDocs runs here and not in InitHttpService because it reads the
// swagger metadata that InitEndpoint's WrapService calls accumulate - calling
// it earlier would write a document of zero routes.
func StartService(
	config config.Configuration,
	httpValidator *http_validator.HTTPController,
) {
	httpValidator.GenerateSwaggerDocs(
		"/inspect",
		"SDMS Teacher Management Documentation",
		true,
	)

	address := fmt.Sprintf("0.0.0.0:%d", config.Server.Port)

	log.Info().
		Str("action", "server.start").
		Int("port", config.Server.Port).
		Msg("HTTP Server Start.")

	// Not http.ListenAndServe, which builds a Server with every timeout left at
	// its zero value - and a zero ReadTimeout/WriteTimeout means no timeout at
	// all, not a default. A client that opens a connection and then sends
	// nothing but a header byte an hour holds a goroutine and a socket for as
	// long as it likes, which is the whole of a slowloris.
	err := newServer(address, limitRequestBody(router)).ListenAndServe()
	if err != nil {
		log.Fatal().
			Str("action", "server.stop").
			Int("port", config.Server.Port).
			Msg("HTTP Server Stopped.")
	}
}

// maxRequestBody is the ceiling on how much of a request body this service will
// read.
//
// nexcommon reads the body with an unbounded io.ReadAll before any validation
// runs, and no route sets a limit of its own - so without this ceiling the
// caller decides how much memory the process allocates. The ReadTimeout above
// bounds how long a client has, not how much it sends: a fast connection can
// deliver hundreds of megabytes inside 30 seconds, and concurrent requests
// multiply it.
//
// 1 MiB is far above the largest legitimate request here - a teacher with all
// its education rows is a few kilobytes - and far below anything that threatens
// the process.
const maxRequestBody = 1 << 20

// limitRequestBody caps the request body before the controller reads it.
//
// Applied outside the router so it covers every route, including ones added
// later.
//
// MaxBytesReader alone, with no pre-check on Content-Length, and that is a
// deliberate trade. A declared-length check would let the oversized request be
// answered with a 413 - but nexcommon has no error type for it, so the answer
// would have to be written here, outside the formator, as a plain-text body.
// Every other response this service produces is the {success, header, payload}
// envelope, and one endpoint that answers in a different shape is worse than
// one that answers 400 where 413 would be more precise. With the cap alone the
// read fails inside the framework, which reports it in the usual envelope
// (E-4-CMD-BDY-001). Both paths reject; neither allocates past the cap.
func limitRequestBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBody)
		}
		next.ServeHTTP(w, r)
	})
}

// newServer builds the *http.Server explicitly, so every timeout is named in
// one place rather than left at the zero value the package-level
// http.ListenAndServe would give it.
//
// The numbers are the stdlib's own suggested starting points.
func newServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
	}
}
