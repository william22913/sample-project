package dao

import (
	"database/sql"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"
)

type GetListDataParam struct {
	ctx                  *internalCtx.ContextModel
	db                   *sql.DB
	queryParam           []interface{}
	query                string
	queryWhere           string
	DTO                  in.GetListRequest
	searchBy             []model.SearchParam
	wrap                 func(rows *sql.Rows) (interface{}, error)
	additionalWhere      string
	additionalWhereParam []interface{}
	prefixField          ParamPrefix
	fieldConverter       paramFieldConverter
	createdBy            int64
	checkStatus          bool
	index                int
	deleted              bool
	groupBy              []string
	whereExist           bool
	scope                map[string]interface{}

	// For Count Only
	tableName string
	count     bool
}

func (g *GetListDataParam) GetQuery() string {
	return g.query + g.queryWhere
}

func (g *GetListDataParam) DB(db *sql.DB) *GetListDataParam {
	g.db = db
	return g
}

func (g *GetListDataParam) AdditionalWhereParam(param ...interface{}) *GetListDataParam {
	g.additionalWhereParam = param
	return g
}

func (g *GetListDataParam) Scope(scope map[string]interface{}) *GetListDataParam {
	g.scope = scope
	return g
}

func (g *GetListDataParam) WhereExistOnQuery() *GetListDataParam {
	g.whereExist = true
	return g
}

func (g *GetListDataParam) GroupBy(fields ...string) *GetListDataParam {
	g.groupBy = fields
	return g
}

func (g *GetListDataParam) NoDeleted() *GetListDataParam {
	g.deleted = false
	return g
}

func (g *GetListDataParam) AdditionalWhere(
	add string,
) *GetListDataParam {
	g.additionalWhere = add
	return g
}

func (g *GetListDataParam) Prefix(
	prefix ParamPrefix,
) *GetListDataParam {
	g.prefixField = prefix
	return g
}

func (g *GetListDataParam) FieldConverter(
	fieldConverter paramFieldConverter,
) *GetListDataParam {
	g.fieldConverter = fieldConverter
	return g
}

func (g *GetListDataParam) TableName(
	table string,
) *GetListDataParam {
	g.tableName = table
	return g
}

func (g *GetListDataParam) CreatedBy(
	createdBy int64,
) *GetListDataParam {
	g.createdBy = createdBy
	return g
}

func (g *GetListDataParam) CheckStatus() *GetListDataParam {
	g.checkStatus = true
	return g
}

func (g *GetListDataParam) Index(
	i int,
) *GetListDataParam {
	g.index = i
	return g
}

func (g *GetListDataParam) SetCustomCountQuery(query string) *GetListDataParam {
	g.query = query
	return g
}
