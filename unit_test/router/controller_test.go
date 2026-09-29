// Package router_test guards the one property of the HTTP server that a
// compiled test cannot reach.
//
// It reads router/controller.go as source rather than importing it, and that is
// not laziness: importing sample-project/router transitively imports
// sample-project/config, whose init() calls log.Fatal when the POSTGRESQL_*
// environment is absent - so any test that links the package exits the test
// binary with status 1 and prints no results at all. A source-level assertion
// is the only form this guard can take here.
package router_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func controllerSource(t *testing.T) string {
	t.Helper()

	src, err := os.ReadFile(filepath.Join("..", "..", "router", "controller.go"))
	if err != nil {
		t.Fatalf("reading router/controller.go: %v", err)
	}

	return string(src)
}

// TestServerIsBuiltWithTimeouts is the slowloris guard.
//
// http.ListenAndServe builds a Server with all four timeouts at their zero
// value, and a zero timeout is not a default - it is no limit at all. A client
// that opens a connection and then sends a single header byte holds a socket
// and a goroutine for as long as it likes.
func TestServerIsBuiltWithTimeouts(t *testing.T) {
	src := controllerSource(t)

	if regexp.MustCompile(`http\.ListenAndServe\(`).MatchString(src) {
		t.Error("router/controller.go calls http.ListenAndServe, which leaves every " +
			"timeout at zero - that is no timeout, not a default. Build the http.Server explicitly.")
	}

	literals := regexp.MustCompile(
		`(?s)http\.Server\{(.*?)\n\t\}`,
	).FindStringSubmatch(src)
	if literals == nil {
		t.Fatal("could not find an http.Server composite literal in router/controller.go")
	}

	for _, field := range []string{"ReadHeaderTimeout", "ReadTimeout", "WriteTimeout", "IdleTimeout"} {
		if !regexp.MustCompile(field + `:\s+\d+\s*\*\s*time\.Second`).MatchString(literals[1]) {
			t.Errorf("the http.Server has no %s set to an explicit duration", field)
		}
	}
}

// TestServerIsActuallyUsed checks the literal is not dead code - a Server built
// and then ignored would satisfy the test above while serving through the
// package-level function.
func TestServerIsActuallyUsed(t *testing.T) {
	src := controllerSource(t)

	if !strings.Contains(src, "ListenAndServe()") {
		t.Error("nothing calls ListenAndServe on the constructed server")
	}
	if !strings.Contains(src, "newServer(") {
		t.Error("the constructed server no longer comes from newServer, so the timeouts " +
			"may not be the ones being served")
	}
}

// TestRequestBodyIsCapped guards the memory-exhaustion defence.
//
// nexcommon reads the request body with an unbounded io.ReadAll before any
// validation runs, and no route sets a limit of its own - so the caller decides
// how much this process allocates. ReadTimeout does not cover it: a fast
// connection delivers hundreds of megabytes well inside 30 seconds, and
// concurrent requests multiply that.
//
// The wiring is what this checks. The middleware's own arithmetic is
// http.MaxBytesReader's, which the stdlib tests; what a future edit could
// silently drop is the middleware being in the handler chain at all. That is
// the same failure the two tests above guard against, and it is the reason this
// one is source-level too - see the package comment for why a compiled test
// cannot reach this package.
func TestRequestBodyIsCapped(t *testing.T) {
	src := controllerSource(t)

	// The handler newServer is given - everything between its first comma and
	// the ListenAndServe that follows. Matching the argument's text rather than
	// an identifier is deliberate: the handler is a call expression
	// (limitRequestBody(router)), and a pattern that only accepted a bare name
	// would fail to match and turn this guard into a parse error instead of a
	// check.
	handler := regexp.MustCompile(`newServer\([^,]+,\s*(.+?)\)\s*\.\s*ListenAndServe\(\)`).
		FindStringSubmatch(src)
	if handler == nil {
		t.Fatal("could not read the handler newServer is constructed with")
	}

	if !strings.Contains(src, "func limitRequestBody(") {
		t.Fatal("router/controller.go no longer defines limitRequestBody")
	}
	if !strings.Contains(handler[1], "limitRequestBody") {
		t.Errorf("newServer is served through %q, which is not the body-limiting "+
			"handler - the request body is unbounded again", handler[1])
	}
	if !strings.Contains(src, "http.MaxBytesReader(") {
		t.Error("limitRequestBody is defined but nothing wraps the body in http.MaxBytesReader, " +
			"so it caps nothing")
	}
}
