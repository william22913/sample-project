package dao

import (
	"database/sql"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/util/multi_database"
)

type GetListDataDAO struct {
	DB            *sql.DB
	MultiDatabase *multi_database.MultiDatabase
}

func (g GetListDataDAO) NewGetListDataParam(
	ctx *internalCtx.ContextModel,
	query string,
	dto in.GetListRequest,
	searchBy []model.SearchParam,
	wrap RowsParser,
) (
	*GetListDataParam,
	error,
) {
	var err error

	dbConn := g.DB
	if g.MultiDatabase != nil {
		dbConn, err = g.MultiDatabase.GetDBConnectionByContext(ctx)
		if err != nil {
			return nil, err
		}
	}

	param := &GetListDataParam{
		db:             dbConn,
		prefixField:    NewParamPrefix(),
		fieldConverter: NewParaFieldConverter(),
		DTO:            dto,
		searchBy:       searchBy,
		wrap:           wrap,
		index:          1,
		query:          query,
		deleted:        true,
	}

	return param, nil
}

func (d GetListDataDAO) GetListDataWithDefaultMustCheck(
	param *GetListDataParam,
) (
	result []interface{},
	err error,
) {
	return d.defaultMustCheck(param)
}

func (d GetListDataDAO) GetListDataWithNextPageChecker(
	param *GetListDataParam,
) (
	result []interface{},
	isTokenAvailable bool,
	err error,
) {
	result, err = d.defaultMustCheck(param)

	if err != nil {
		return
	}

	if len(result) > param.DTO.Limit {
		isTokenAvailable = true
		result = result[:param.DTO.Limit]
	}

	return
}

func (d GetListDataDAO) defaultMustCheck(
	param *GetListDataParam,
) (
	result []interface{},
	err error,
) {
	param.queryWhere = ""
	// param.whereExist = false
	for i := 0; i < len(param.searchBy); i++ {
		if !strings.Contains(param.searchBy[i].SearchKey, ".") {
			param.searchBy[i].SearchKey = generateFieldQueryWithoutConverter(param, param.searchBy[i].SearchKey)
		}
	}

	tempQuery := SearchByParamToQueryWithDefaultField(param)

	param.queryWhere = param.queryWhere + tempQuery
	d.getListDataWithDefaultMustCheck(param)
	result, err = GetDataMultipleRowResult(param)

	return
}

func (d GetListDataDAO) GetCountDataWithDefaultMustCheck(
	param *GetListDataParam,
) (
	result int,
	err error,
) {
	param.queryWhere = ""

	if param.query == "" {
		param.query =
			"SELECT " +
				"	count(" + generateFieldQuery(param, "id") + ") " +
				"FROM " +
				"	" + param.tableName + " "
	}

	param.count = true

	tempQuery := SearchByParamToQueryWithDefaultField(param)
	param.queryWhere = param.queryWhere + tempQuery
	d.getListDataWithDefaultMustCheck(param)

	if param.db != nil {

		results := param.db.QueryRow(
			param.query+param.queryWhere,
			param.queryParam...,
		)

		err = results.Scan(&result)
		if err != nil && err != sql.ErrNoRows {
			return
		}
	}

	return
}

func (d GetListDataDAO) getListDataWithDefaultMustCheck(
	param *GetListDataParam,
) {

	if param.createdBy > 0 {
		param.queryParam = append(param.queryParam, param.createdBy)
	}

	for i := 0; i < len(param.searchBy); i++ {
		if reflect.TypeOf(param.searchBy[i].SearchValue).Kind() == reflect.Slice {
			values := param.searchBy[i].SearchValue.([]interface{})
			for i := range values {
				param.queryParam = append(param.queryParam, values[i])
			}
		} else {
			if param.searchBy[i].FormatedSearchValue != nil {
				param.queryParam = append(param.queryParam, param.searchBy[i].FormatedSearchValue)
			} else {
				if param.searchBy[i].SearchValue != "</nil>" {
					param.queryParam = append(param.queryParam, param.searchBy[i].SearchValue)
				}
			}
		}
	}

	if param.checkStatus {
		param.queryWhere += fmt.Sprintf(" AND %s = 'A' ", generateFieldQuery(param, statusDBField))
	}

	if param.additionalWhere != "" {
		if param.additionalWhereParam != nil {
			for i := 0; i < len(param.additionalWhereParam); i++ {
				param.queryParam = append(param.queryParam, param.additionalWhereParam[i])
				param.additionalWhere = strings.Replace(param.additionalWhere, "??", fmt.Sprintf("$%d", param.index), 1)
				param.index++
			}
		}

		replacedWhere := strings.ReplaceAll(strings.ToUpper(param.additionalWhere), "where ", "WHERE ")
		additionalWhereExist := strings.Contains(replacedWhere, "WHERE ")

		if additionalWhereExist && param.whereExist {

			trim := strings.Trim(param.additionalWhere, " ")
			if strings.HasPrefix(trim, "WHERE") {
				trim = strings.Replace(trim, "WHERE", "AND", 1)
				param.additionalWhere = " " + trim + " "
			} else if !(strings.HasPrefix(trim, "AND")) {
				param.additionalWhere = " AND " + param.additionalWhere
			}

		} else {
			connector := strings.Split(strings.Trim(param.additionalWhere, " "), " ")[0]

			if connector != "AND" && connector != "WHERE" {
				if param.whereExist {
					param.additionalWhere = " AND " + param.additionalWhere
				} else {
					param.additionalWhere = " WHERE " + param.additionalWhere
					param.whereExist = true
				}
			}

		}

		param.queryWhere += param.additionalWhere
	}

	if len(param.groupBy) > 0 {
		param.queryWhere += " GROUP BY "
		for i := 0; i < len(param.groupBy); i++ {
			param.queryWhere += " " + generateFieldQuery(param, param.groupBy[i]) + " "
			if i < len(param.groupBy)-1 {
				param.queryWhere += " , "
			}
		}
	}

	if !param.count {
		var order string
		orderSplit := strings.Split(param.DTO.Order, ",")
		var orderTemp []string

		for i := 0; i < len(orderSplit); i++ {
			splitSpace := strings.Split(strings.Trim(orderSplit[i], " "), " ")

			if splitSpace[0] != "" {
				if len(splitSpace) == 1 {
					orderTemp = append(orderTemp, fmt.Sprintf("%s %s", generateFieldQuery(param, splitSpace[0]), "ASC"))
				} else {
					orderTemp = append(orderTemp, fmt.Sprintf("%s %s", generateFieldQuery(param, splitSpace[0]), splitSpace[len(splitSpace)-1]))
				}
			}
		}

		if len(orderTemp) > 0 || param.DTO.OrderIDs != "" {
			order = strings.Join(orderTemp, ", ")
			orders := []string{}

			if param.DTO.OrderIDs != "" {
				splitData := strings.Split(param.DTO.OrderIDs, " in ")
				field := generateFieldQuery(param, splitData[0])
				queryOrder := field + " in " + splitData[1]
				orders = append(orders, queryOrder+" "+param.DTO.OrderIDOperator)
			}

			if order != "" {
				orders = append(orders, order)
			}

			param.queryWhere += " ORDER BY " + strings.Join(orders, ", ") + " "
		}

		if param.DTO.Limit != -1 {
			param.queryWhere += "LIMIT $" + strconv.Itoa(param.index) + " OFFSET $" + strconv.Itoa(param.index+1)
			limit := param.DTO.Limit
			offset := CountOffset(param.DTO.Page, param.DTO.Limit)

			if param.DTO.EnableTokenChecker {
				limit += 1
			}

			param.queryParam = append(param.queryParam, limit, offset)
		}
	}

}
