package audit_helper

import (
	"database/sql"
	"time"

	"github.com/nexsoft-git/nexcommon/context"
)

const (
	AuditServiceActionDelete = 0
	AuditServiceActionInsert = 1
	AuditServiceActionUpdate = 2
)

type AuditHelper interface {
	InitAuditService(
		ctx *context.ContextModel,
		inputStruct interface{},
		serveFunction ServiceFunction,
	) (
		*auditBuilder,
		error,
	)

	GetAuditData(
		f GetAuditData,
	) *auditHelper

	GetDefaultSchema() string
}

type AuditSystemModel struct {
	ID            int64     `json:"id"`
	UUIDKey       string    `json:"uuid_key"`
	TableName     string    `json:"table_name"`
	PrimaryKey    int64     `json:"primary_key"`
	Data          string    `json:"data"`
	Action        int32     `json:"action"`
	SchemaName    string    `json:"schema_name"`
	Description   string    `json:"description"`
	CreatedBy     int64     `json:"created_by"`
	CreatedClient string    `json:"created_client"`
	CreatedAt     time.Time `json:"created_at"`
}

type ServiceFunction func(*context.ContextModel, *sql.Tx, interface{}, time.Time) (interface{}, []AuditSystemModel, error)
type GetAuditData func(ctx *context.ContextModel, tx *sql.Tx, action int32, tableName string, id int64, createdBy int64, schema string) ([]AuditSystemModel, error)
type AfterCommit func(*context.ContextModel, interface{})
