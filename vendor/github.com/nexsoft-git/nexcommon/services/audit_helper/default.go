package audit_helper

import (
	"database/sql"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
)

func GetDataForAuditByIDTx(
	ctx *context.ContextModel,
	tx *sql.Tx,
	action int32,
	tableName string,
	id int64,
	createdBy int64,
	schema string,
) (
	[]AuditSystemModel,
	error,
) {
	auditField := make(map[string]model.AuditSystemFieldParam)
	auditField["id"] = model.AuditSystemFieldParam{
		IsEqual:    true,
		ParamValue: id,
	}
	if action != AuditServiceActionDelete {
		auditField["deleted"] = model.AuditSystemFieldParam{
			IsEqual:    true,
			ParamValue: false,
		}
	}
	if createdBy > 0 {
		auditField["created_by"] = model.AuditSystemFieldParam{
			IsEqual:    true,
			ParamValue: createdBy,
		}
	}

	return GetDataForAuditTx(ctx, tx, action, tableName, auditField, schema)
}

func GetDataForAuditByIDsTx(
	ctx *context.ContextModel,
	tx *sql.Tx,
	action int32,
	tableName string,
	ids []int64,
	createdBy int64,
	schema string,
) (
	[]AuditSystemModel,
	error,
) {
	auditField := make(map[string]model.AuditSystemFieldParam)
	auditField["id"] = model.AuditSystemFieldParam{
		IsEqual:    true,
		ParamValue: ids,
	}
	if action != AuditServiceActionDelete {
		auditField["deleted"] = model.AuditSystemFieldParam{
			IsEqual:    true,
			ParamValue: false,
		}
	}
	if createdBy > 0 {
		auditField["created_by"] = model.AuditSystemFieldParam{
			IsEqual:    true,
			ParamValue: createdBy,
		}
	}

	return GetDataForAuditTx(ctx, tx, action, tableName, auditField, schema)
}

func GetDataForAuditTx(
	ctx *context.ContextModel,
	tx *sql.Tx,
	action int32,
	tableName string,
	auditField map[string]model.AuditSystemFieldParam,
	schema string,
) (
	result []AuditSystemModel,
	err error,
) {
	schemaName := ctx.Limitation.DBSchema
	queryAdd, param := AuditSystemFieldParamToQuery(1, auditField)

	if schemaName == "" {
		schemaName = schema
	}

	schemaName = fmt.Sprintf("%s.", strings.Trim(schemaName, "."))
	table := fmt.Sprintf("%s%s", schemaName, tableName)

	query := `SELECT a.id, uuid_key, row_to_json(a) FROM (SELECT * FROM ` + table + ` WHERE ` + queryAdd + ` FOR UPDATE) a `

	rows, err := tx.Query(query, param...)
	if err != nil {
		return result, err
	}

	if rows != nil {
		defer func() {
			err = rows.Close()
			if err != nil {
				return
			}
		}()

		for rows.Next() {
			var temp AuditSystemModel
			err = rows.Scan(&temp.PrimaryKey, &temp.UUIDKey, &temp.Data)
			if err != nil {
				return
			}

			temp.CreatedAt = time.Now()
			temp.Action = action
			temp.TableName = tableName
			temp.CreatedBy = ctx.AuthAccessTokenModel.ResourceUserID
			temp.CreatedClient = ctx.AuthAccessTokenModel.ClientID
			temp.SchemaName = schemaName
			result = append(result, temp)
		}
	} else {
		return result, err
	}

	return
}

func AuditSystemFieldParamToQuery(
	index int,
	userParam map[string]model.AuditSystemFieldParam,
) (
	query string,
	param []interface{},
) {

	if index < 1 {
		index = 1
	}

	keys := make([]string, 0, len(userParam))

	for key := range userParam {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for i := 0; i < len(keys); i++ {

		key := keys[i]

		if query != "" {
			query += " AND "
		}

		query += "\"" + key + "\" "
		if reflect.TypeOf(userParam[key].ParamValue).Kind() == reflect.Slice {
			val := reflect.ValueOf(userParam[key].ParamValue)

			if val.Len() > 0 {
				query += " IN ("
				for i := 0; i < val.Len(); i++ {
					if i > 0 {
						query += ","
					}
					query += "$" + strconv.Itoa(index)
					param = append(param, val.Index(i).Interface())
					index++
				}
				query += ")"
			}

		} else {
			if userParam[key].IsEqual {
				query += " = $" + strconv.Itoa(index)
				param = append(param, userParam[key].ParamValue)
			} else {
				query += " LIKE $" + strconv.Itoa(index)
				param = append(param, userParam[key].ParamValue)
			}

			index++
		}

	}

	return
}
