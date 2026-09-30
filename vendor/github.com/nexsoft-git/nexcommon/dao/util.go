package dao

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexlogger/log"
)

func SearchByParamToQueryWithDefaultField(
	param *GetListDataParam,
) string {

	var result string

	if len(param.searchBy) > 0 || param.createdBy != 0 {
		if !param.whereExist {
			result = "WHERE \n"
			param.whereExist = true
		} else {
			result = "AND \n"
		}

		if param.createdBy > 0 {
			temp := fmt.Sprintf(" %s = $%d ", generateFieldQuery(param, createdByDBField), param.index)

			if len(param.searchBy) != 0 {
				temp += " AND "
			}

			result += temp
			param.index++
		}

		for i := 0; i < len(param.searchBy); i++ {
			operator := ""
			queryValue := ""

			//TODO validate || &&
			if param.searchBy[i].SearchOperator == "in" {
				operator = "IN"
				value := param.searchBy[i].SearchValue.([]interface{})
				if param.searchBy[i].NegationQuery {
					operator = "NOT IN"
				}

				queryValue = "("

				for j := range value {
					queryValue += fmt.Sprintf(" $%d ", param.index)

					if j != len(value)-1 {
						queryValue += ", "
					}

					param.index++
				}

				queryValue += ")"
			} else {
				if param.searchBy[i].DataType == "enum" {
					param.searchBy[i].SearchKey = "cast( " + param.searchBy[i].SearchKey + " AS VARCHAR)"
				}

				if param.searchBy[i].SearchOperator == "lk" {
					param.searchBy[i].SearchValue = strings.ReplaceAll(strings.ToLower(param.searchBy[i].SearchValue.(string)), "_", "\\_")
					param.searchBy[i].SearchValue = "%" + param.searchBy[i].SearchValue.(string) + "%"
					param.searchBy[i].SearchOperator = "like"
					if param.searchBy[i].NegationQuery {
						param.searchBy[i].SearchOperator = "NOT LIKE"
					}
				}

				if param.searchBy[i].SearchValue == "</nil>" {
					queryValue = "NULL"
					operator = "IS"
					if param.searchBy[i].NegationQuery {
						operator = "IS NOT"
					}
				} else {
					operator = param.searchBy[i].SearchOperator
					if param.searchBy[i].SearchOperator == "eq" {
						operator = "="
						if param.searchBy[i].NegationQuery {
							operator = "<>"
						}
					} else if param.searchBy[i].SearchOperator == "be" {
						operator = ">="
						if param.searchBy[i].NegationQuery {
							operator = "<"
						}
					} else if param.searchBy[i].SearchOperator == "lq" {
						operator = "<="
						if param.searchBy[i].NegationQuery {
							operator = ">"
						}
					} else if param.searchBy[i].SearchOperator == "lt" {
						operator = "<"
						if param.searchBy[i].NegationQuery {
							operator = ">="
						}
					} else if param.searchBy[i].SearchOperator == "bt" {
						operator = ">"
						if param.searchBy[i].NegationQuery {
							operator = "<="
						}
					}

					queryValue = " $" + strconv.Itoa(param.index)
					param.index++
				}
			}

			query := "( "
			key := ""

			for j := 0; j < len(param.searchBy[i].SearchKey)-1; j++ {

				current := string(param.searchBy[i].SearchKey[j])
				next := string(param.searchBy[i].SearchKey[j+1])

				found := false

				keyUsed := generateFieldQuery(param, key)
				if current+next == "&&" {
					found = true

					query += generateQuery(keyUsed, operator, queryValue, "AND")
					key = ""
				} else if current+next == "||" {
					found = true

					query += generateQuery(keyUsed, operator, queryValue, "OR")
					key = ""
				} else {
					key += current
					if j == len(param.searchBy[i].SearchKey)-2 {
						key += next
					}
				}

				if found {
					if j+1 <= len(param.searchBy[i].SearchKey)-1 {
						j = j + 1
					}
				}

			}

			keyUsed := generateFieldQuery(param, key)
			query += generateQuery(keyUsed, operator, queryValue, ")")

			if i < len(param.searchBy)-1 {
				query += fmt.Sprintf(" %s ", param.DTO.Connector)
			}

			result += query
		}
	}

	scopePrefix := " WHERE "
	deletedPrefix := " WHERE "

	if param.whereExist {
		scopePrefix = " AND "
	}

	if param.whereExist {
		deletedPrefix = " AND "
	}

	if param.scope != nil {
		for key := range param.scope {
			field := generateFieldQuery(param, key)
			if listID, ok := param.scope[key].([]interface{}); ok {
				query := ListDataToInQuery(len(param.queryParam), listID)
				param.query += fmt.Sprintf(" %s %s IN ( %s )", scopePrefix, field, query)
				deletedPrefix = " AND "
			}
		}
	}

	if param.deleted {
		param.whereExist = true
		result += fmt.Sprintf("%s %s = FALSE ", deletedPrefix, generateFieldQuery(param, deletedDBField))
	}

	return result
}

func ListDataToInQuery(
	start int,
	listScope []interface{},
) string {
	result := ""
	if start > 0 {
		start--
	}
	for i := 0; i < len(listScope); i++ {
		result += "$" + strconv.Itoa(i+1+start)
		if i < len(listScope)-1 {
			result += ", "
		}
	}
	return result
}

func generateFieldQuery(
	param *GetListDataParam,
	key string,
) string {
	field := param.fieldConverter.Get(key)
	if field == "" {
		return param.prefixField.Get(key) + key
	}

	return param.prefixField.Get(field) + field
}

func generateFieldQueryWithoutConverter(
	param *GetListDataParam,
	key string,
) string {
	return param.prefixField.Get(key) + key
}

func generateQuery(key, operator, value, additional string) string {

	if operator == "like" {
		key = fmt.Sprintf("LOWER(%s)", key)
	}

	return " " + key + " " + operator + " " + value + " " + additional + " "
}

func CountOffset(page int, limit int) int {
	return (page - 1) * limit
}

func GetDataMultipleRowResult(
	param *GetListDataParam,
) (
	result []interface{},
	err error,
) {
	if param.db == nil {
		return []interface{}{}, nil
	}

	rows, err := param.db.Query(
		param.query+param.queryWhere,
		param.queryParam...,
	)

	if err != nil {
		return
	}

	if rows != nil {
		defer func() {
			errs := rows.Close()
			if errs != nil {
				log.Error().
					Err(errs).
					Caller().
					Msg("Error Found when close get list dao connection")
				return
			}
		}()

		for rows.Next() {
			var temp interface{}

			if param.wrap != nil {
				temp, err = param.wrap(rows)
				if err != nil {
					return
				}

				result = append(result, temp)
			}

		}
	}

	return
}

func GetDBTable(
	ctx *context.ContextModel,
	table string,
) string {
	if ctx.Limitation.DBSchema == "" {
		return table
	} else {
		return fmt.Sprintf("%s.%s", ctx.Limitation.DBSchema, table)
	}
}
