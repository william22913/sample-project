package audit_helper

import (
	"database/sql"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexlogger/log"
)

type publisher struct {
	nats    nats.JetStreamContext
	subject string
}

type auditBuilder struct {
	ctx           *context.ContextModel
	inputStruct   interface{}
	serveFunction ServiceFunction
	afterCommit   AfterCommit
	getAuditData  GetAuditData
	db            *sql.Tx
	publisher     publisher
	time          time.Time
	defaultSchema string
}

func (ab *auditBuilder) AfterCommit(
	f AfterCommit,
) *auditBuilder {
	ab.afterCommit = f
	return ab
}

func (ab *auditBuilder) CompletedAuditData() (
	interface{},
	error,
) {
	return ab.doAuditData()
}

func (ab *auditBuilder) doAuditData() (
	output interface{},
	err error,
) {

	var dataAudit []AuditSystemModel

	defer func() {
		if err != nil {
			_ = ab.db.Rollback()
		} else {
			for i := 0; i < len(dataAudit); i++ {
				if dataAudit[i].Action == AuditServiceActionInsert {
					dataAudit, err = ab.preparedAuditData(
						dataAudit,
					)

					if err != nil {
						log.Error().
							Err(err).
							Msg("Error found when process the audit data.")
						_ = ab.db.Rollback()

						return
					}
				}
			}

			if ab.db != nil {
				err = ab.db.Commit()

				if err != nil {
					return
				}
			}

			if output != nil && ab.afterCommit != nil {
				go ab.afterCommit(ab.ctx, output)
			}

			ab.serviceWithAudit(ab.ctx, dataAudit)
		}
	}()

	output, dataAudit, err = ab.serveFunction(ab.ctx, ab.db, ab.inputStruct, ab.time)

	if err != nil {
		return
	}

	return
}

func (ab *auditBuilder) preparedAuditData(
	dataAudit []AuditSystemModel,
) (
	result []AuditSystemModel,
	err error,
) {
	for i := 0; i < len(dataAudit); i++ {
		var temp []AuditSystemModel

		schemaName := ab.defaultSchema
		if dataAudit[i].SchemaName != "" {
			schemaName = ab.defaultSchema
		}

		temp, err = ab.getAuditData(
			ab.ctx,
			ab.db,
			dataAudit[i].Action,
			dataAudit[i].TableName,
			dataAudit[i].PrimaryKey,
			ab.ctx.AuthAccessTokenModel.ResourceUserID,
			schemaName,
		)

		if err != nil {
			return nil, err
		}

		result = append(
			result,
			temp...,
		)
	}

	return
}

func (ab *auditBuilder) serviceWithAudit(
	ctx *context.ContextModel,
	data []AuditSystemModel,
) {
	go ab.pushAuditToMessageBroker(ctx, data)
}

func (ab *auditBuilder) pushAuditToMessageBroker(
	ctx *context.ContextModel,
	data []AuditSystemModel,
) {

	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Caller().
				Interface("panic", r).
				LoggerModel(ctx.ClientAccess.Logger).
				Str("stack", string(debug.Stack())).
				Msg("Panic recovered")
		}
	}()

	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Error().
			LoggerModel(ctx.ClientAccess.Logger).
			Err(err).
			Interface("data", data).
			Caller().
			Msg("Error Found when marshaling data audit")
	}

	_, err = ab.publisher.nats.Publish(ab.publisher.subject, jsonData)
	if err != nil {
		log.Error().
			Err(err).
			LoggerModel(ctx.ClientAccess.Logger).
			Interface("data", data).
			Interface("subject", ab.publisher.subject).
			Caller().
			Msg("Error Found when publishing data audit")
	}

	return
}
