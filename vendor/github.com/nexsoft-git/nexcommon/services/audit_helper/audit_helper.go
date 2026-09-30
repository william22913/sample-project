package audit_helper

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nexsoft-git/nexcommon/context"
	errors "github.com/nexsoft-git/nexcommon/error"
)

func NewAuditHelper(
	enable bool,
	db *sql.DB,
	nats nats.JetStreamContext,
	subject string,
	defaultSchema string,
) AuditHelper {
	defaultSchema = fmt.Sprintf("%s.", strings.Trim(defaultSchema, "."))
	return &auditHelper{
		db:            db,
		getAuditData:  GetDataForAuditByIDTx,
		nats:          nats,
		enabled:       enable,
		subject:       subject,
		defaultSchema: defaultSchema,
	}
}

func (a *auditHelper) GetAuditData(f GetAuditData) *auditHelper {
	a.getAuditData = f
	return a
}

func (a *auditHelper) GetDefaultSchema() string {
	return a.defaultSchema
}

type auditHelper struct {
	db            *sql.DB
	nats          nats.JetStreamContext
	enabled       bool
	getAuditData  GetAuditData
	subject       string
	defaultSchema string
}

func (a *auditHelper) InitAuditService(
	ctx *context.ContextModel,
	inputStruct interface{},
	serveFunction ServiceFunction,
) (
	*auditBuilder,
	error,
) {

	if !a.enabled {
		return nil, errors.ErrAuditIsDisabled.ContextModel(ctx)
	}

	builder := &auditBuilder{
		ctx:          ctx,
		inputStruct:  inputStruct,
		getAuditData: a.getAuditData,
		time:         time.Now(),
		publisher: publisher{
			subject: a.subject,
			nats:    a.nats,
		},
		defaultSchema: a.defaultSchema,
		serveFunction: serveFunction,
	}

	if a.db != nil {
		tx, err := a.db.Begin()

		if err != nil {
			return nil, err
		}
		builder.db = tx
	}

	return builder, nil

}
