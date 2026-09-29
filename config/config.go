// Package config loads the service's configuration once, at process startup,
// via envconfig struct tags - never scattered os.Getenv calls. See
// nexcommon-go-config-standards. Never log a populated Configuration value or
// any individual secret field.
package config

import (
	"os"

	"github.com/kelseyhightower/envconfig"
	"github.com/nexsoft-git/nexlogger"
	"github.com/nexsoft-git/nexlogger/log"
)

var AppConfig Configuration

func init() {
	// The logger is initialized here, first, and this is the only place it
	// happens - see nexcommon-go-config-standards ("already initialized once at
	// bootstrap in config/config.go's init(); don't re-initialize it").
	//
	// It has to come before the config load below, and that ordering is the
	// whole point. nexlogger's global starts as the zero value of its Logger
	// struct, whose writes are discarded, and nothing in nexcommon calls
	// InitiateLogger. Left uninitialized, every log call in this service and in
	// the framework is a no-op: the framework's per-request completion line,
	// every log.Error() on an unexpected failure, and - worst of the three - the
	// log.Fatal() directly below. A misconfigured process then exits with status
	// 1 having printed nothing at all, so a deployment with a bad DSN becomes a
	// silent restart loop with no diagnostic anywhere. Writing to stderr is what
	// puts the output in front of whoever is reading the container's logs.
	log.InitiateLogger(nexlogger.New(os.Stderr))

	AppConfig = Configuration{}
	err := envconfig.Process("", &AppConfig)
	if err != nil {
		log.Fatal().Msg(err.Error())
	}
}

type Configuration struct {
	Server     server     `envconfig:"server"`
	Postgresql Postgresql `envconfig:"postgresql"`
	Nats       nats       `envconfig:"nats"`
	TokenKey   tokenKey   `envconfig:"token_key"`
}

type server struct {
	ResourceID string `envconfig:"resourceid" default:"sample-project"`
	Version    string `envconfig:"version" default:"1.0.0"`
	Host       string `envconfig:"host"`
	Port       int    `envconfig:"port" default:"8080"`
}

type Postgresql struct {
	Host              string `envconfig:"host" required:"true"`
	Port              int    `envconfig:"port" default:"5432"`
	Username          string `envconfig:"username" required:"true"`
	Password          string `envconfig:"password" required:"true"`
	DBName            string `envconfig:"dbname" required:"true"`
	SSLMode           string `envconfig:"sslmode" default:"disable"`
	DefaultSchema     string `envconfig:"defaultschema" default:"public"`
	MaxOpenConnection int    `envconfig:"maxopenconnection" default:"20"`
	MaxIdleConnection int    `envconfig:"maxidleconnection" default:"5"`
}

// nats holds the JetStream connection settings used by nexcommon's
// audit_helper (architecture.md A1). Audit entries are published to
// AuditSubject; nothing in this service consumes it - a downstream consumer
// persists them, which is why criterion 16 is at-most-once.
type nats struct {
	Host         string `envconfig:"host" default:"localhost"`
	Port         int    `envconfig:"port" default:"4222"`
	AuditSubject string `envconfig:"audit_subject" default:"audit.sample-project.teacher"`
	AuditStream  string `envconfig:"audit_stream" default:"AUDIT"`
}

// tokenKey holds the JWT verification keys. No login/token-issuance endpoint
// exists in this service; FEAT-001 has no authentication at all and every
// route uses WhitelistValidator (architecture.md A2), so these are unused by
// the feature's routes but required by NewHTTPController.
type tokenKey struct {
	User               string `envconfig:"user"`
	Internal           string `envconfig:"internal"`
	FixedInternalToken string `envconfig:"fixed"`
}
