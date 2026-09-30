package dao

import (
	"database/sql"
	"strings"
)

type RowsParser func(rows *sql.Rows) (interface{}, error)

const (
	idDBField        = "id"
	deletedDBField   = "deleted"
	statusDBField    = "status"
	createdByDBField = "created_by"
	updatedByDBField = "updated_by"
	createdAtDBField = "created_at"
	updatedAtDBField = "updated_at"
)

type ParamPrefix interface {
	Add(key string, value string) paramPrefix
	Get(key string) string
}

type paramFieldConverter map[string]string
type paramPrefix map[string]string

func (p paramPrefix) Add(key string, value string) paramPrefix {
	value = strings.Trim(value, ".") + "."
	p[key] = value
	return p
}

func (p paramPrefix) Get(key string) string {
	result, _ := p[key]
	return result
}

func NewParamPrefix() ParamPrefix {
	return make(paramPrefix)
}

func NewParaFieldConverter() paramFieldConverter {
	return make(paramFieldConverter)
}

func (p paramFieldConverter) Add(key string, value string) paramFieldConverter {
	p[key] = value
	return p
}

func (p paramFieldConverter) Get(key string) string {
	result, _ := p[key]
	return result
}
