package server_config

import (
	"fmt"

	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	mdl "github.com/nexsoft-git/nexcommon/http/server/middleware"
	"github.com/nexsoft-git/nexcommon/services/audit_helper"
	"github.com/nexsoft-git/nexcommon/util"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/get_list_validator"
	"github.com/nexsoft-git/nexlogger"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/config"
	"sample-project/validator"
)

func NewServerAttribute(
	config config.Configuration,
) *serverAttribute {
	return &serverAttribute{
		config: config,
	}
}

// Init builds everything the process needs, in four phases, and it has to be
// this order:
//
//  1. connections - nothing downstream can be built without a driver.
//  2. bundles and validators - the controller reads its rules from these at
//     construction, so they must exist first.
//  3. the HTTP controller - configured from phase 2, and needed by DAOs?
//     no: needed by the middleware and the endpoints, both of which come later.
//  4. DAOs then services - built from the DB connection and the audit helper.
//
// A failure in any phase returns before the next one starts, so a partially
// built root is never handed to main.
func (s *serverAttribute) Init() (err error) {
	// Phase 1 - connections.
	//
	// MaxOpen/MaxIdle come from config rather than the library's own defaults
	// (500/100), which are sized for a service with a much larger request
	// concurrency than this one has.
	s.DBConnection = util.GetDbConnection(
		util.DBAddressParam().
			Host(s.config.Postgresql.Host).
			Port(s.config.Postgresql.Port).
			DBName(s.config.Postgresql.DBName).
			DefaultSchema(s.config.Postgresql.DefaultSchema).
			Username(s.config.Postgresql.Username).
			Password(s.config.Postgresql.Password).
			MaxOpenConnection(s.config.Postgresql.MaxOpenConnection).
			MaxIdleConnection(s.config.Postgresql.MaxIdleConnection),
	)

	// JetStream, because audit_helper publishes through JetStreamContext, not
	// the plain connection - a core NATS publish would be dropped outright if
	// nothing were subscribed at the instant it fired.
	//
	// The stream is created here rather than assumed to exist: the compose file
	// starts a stock nats:alpine, which has JetStream enabled but no streams
	// declared. CreateStream is idempotent, so a restart against a stream that
	// already exists is a no-op.
	s.NatsConn, s.NatsJS, err = util.ConnectNatsJs(
		fmt.Sprintf("%s:%d", s.config.Nats.Host, s.config.Nats.Port),
	)
	if err != nil {
		return
	}

	err = util.CreateStream(
		s.NatsJS,
		s.config.Nats.AuditStream,
		[]string{s.config.Nats.AuditSubject},
	)
	if err != nil {
		return
	}

	// Phase 2 - bundles and validators.
	s.Bundles, err = bundles.NewBundles("i18n", "id-ID")
	if err != nil {
		return
	}

	s.Validator.basicValidator = basic_validator.BasicValidator{}
	s.Validator.tagValidator = validator.NewTagValidator()
	s.Validator.HttpController = http_validator.NewHTTPController(
		s.config.TokenKey.FixedInternalToken,
	)

	// The project's tag validator, not the library's - see
	// validator.NewTagValidator for the three rules it adds and why an
	// unregistered regex would fail open.
	s.Validator.HttpController.
		Version(s.config.Server.Version).
		BasicValidator(s.Validator.basicValidator).
		TagValidator(s.Validator.tagValidator).
		ListDataValidator(get_list_validator.NewGetListValidator(s.Validator.basicValidator)).
		LogEndpointsHit()

	// The internal code is the one a caller sees when an error reached the
	// formator that is not one of this feature's typed sentinels - a driver
	// failure, a marshal failure. It carries no detail on purpose: there is
	// nothing about a raw DB error the caller can act on, and the real error is
	// in the log.
	s.errorFormator = errors.NewErrorFormator(s.Bundles).
		DefaultInternalCode("E-5-TCH-SRV-001").
		DefaultLanguage("id-ID").
		Version(s.config.Server.Version)

	s.Validator.HttpController.Formator(s.errorFormator)

	// applicationName is the resource id: this process serves exactly one
	// service, so the two names are the same string and a second config field
	// would only be a second thing to keep in step.
	nexlogger.InitLoggerModel(
		s.config.Server.ResourceID,
		s.config.Server.ResourceID,
		s.config.Server.Version,
	)

	// Phase 3 - global middleware, for the CORS headers it writes. No metrics
	// collector is attached: nothing scrapes this service.
	s.MetricMiddleware = mdl.NewHTTPMiddleware(
		*s.Validator.HttpController,
		s.Validator.basicValidator,
		s.errorFormator,
	)

	s.HealthChecker = health.NewHealthEndpoint(
		health.NewListTools("db", health.NewDBConnChecker(s.DBConnection)),
	)

	// Phase 4 - audit helper, DAOs, services.
	//
	// enable is hardcoded true, and that is not a shortcut around configuration.
	// audit_helper with enable=false makes InitAuditService return
	// ErrAuditIsDisabled (E-4-CMD-ADT-001, 404) *and* a nil builder, so every
	// write route in this service would fail - create, update and deactivate
	// alike. There is no deployment of this feature in which auditing is off
	// and the service still works, so there is nothing for a flag to express.
	//
	// The schema is the same one the DAOs connect with, because the snapshot
	// query audit_helper runs is `SELECT ... FROM <schema>teachers`, unqualified
	// by search_path.
	s.auditHelper = audit_helper.NewAuditHelper(
		true,
		s.DBConnection,
		s.NatsJS,
		s.config.Nats.AuditSubject,
		s.config.Postgresql.DefaultSchema,
	)

	log.Info().
		Str("action", "server.init").
		Int("db_max_open_connection", s.config.Postgresql.MaxOpenConnection).
		Msg("Server attribute initialized")

	s.InitDAO()
	s.InitServices()

	return
}
