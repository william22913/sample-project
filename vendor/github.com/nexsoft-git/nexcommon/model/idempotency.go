package model

import "database/sql"

type IdempotencyServiceProcessResult struct {
	Output interface{}       `json:"output"`
	Header map[string]string `json:"header"`
	Type   string            `json:"type"`
}

type IdempotencyModel struct {
	Id             sql.NullInt64
	IdempotencyKey sql.NullString
	ResultStr      sql.NullString
	Result         IdempotencyServiceProcessResult
	CreatedAt      sql.NullTime
}
