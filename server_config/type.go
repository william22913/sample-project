// Package server_config is the composition root: it builds every connection,
// validator and service exactly once, at startup, and hands them to the layers
// that need them.
//
// The split into five files is the layout the other services in this
// ecosystem use, and the split is by *when* things can be built rather than by
// package - connections first, then validators, then the controller, then DAOs
// and services from those. Init() depends on that order, and the file a field
// lives in says which phase owns it.
package server_config

import (
	"database/sql"

	"github.com/nats-io/nats.go"
	"github.com/nexsoft-git/nexcommon/bundles"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/http/endpoint/health"
	mdl "github.com/nexsoft-git/nexcommon/http/server/middleware"
	"github.com/nexsoft-git/nexcommon/services/audit_helper"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"

	"sample-project/config"
	"sample-project/dao"
	teacher "sample-project/service/teacher"
)

type serverAttribute struct {
	config config.Configuration

	DBConnection *sql.DB
	NatsConn     *nats.Conn
	NatsJS       nats.JetStreamContext

	Bundles   bundles.Bundles
	Validator validators

	MetricMiddleware mdl.MetricMiddleware
	HealthChecker    health.HealthEndpoint
	errorFormator    errors.Formator

	listDAO     listDAO
	auditHelper audit_helper.AuditHelper
	Services    services
}

type validators struct {
	basicValidator basic_validator.BasicValidator
	tagValidator   tag_validator.TagValidator

	// HttpController is a pointer because it is configured by chained setters
	// that return the receiver; a value copy here would take the configuration
	// and drop it.
	HttpController *http_validator.HTTPController
}

type listDAO struct {
	TeacherDAO          dao.TeacherDAO
	EducationDAO        dao.EducationDAO
	InstitutionLevelDAO dao.InstitutionLevelDAO
}

type services struct {
	Teacher          *teacher.Service
	TeacherEducation *teacher.EducationService
}
