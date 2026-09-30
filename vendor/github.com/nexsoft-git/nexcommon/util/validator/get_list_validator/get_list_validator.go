package get_list_validator

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/in"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
)

var InSeparator = "-"
var orderSeparator = " "
var filterSeparator = ", "
var filterValueSeparator = " "
var rangeSeparator = ">>"

func NewGetListValidator(
	basicValidator basic_validator.BasicValidator,
) GetListValidator {
	return &getListValidator{
		basicValidator: basicValidator,
	}
}

type getListValidator struct {
	basicValidator basic_validator.BasicValidator
}

func (v *getListValidator) ValidateGetListData(
	ctx *context.ContextModel,
	dto *in.GetListRequest,
	service services.ServicesWithGetListData,
) (
	output []model.SearchParam,
	err error,
) {
	err = v.ValidateInputPageLimitAndOrderBy(ctx, dto, service)
	if err != nil {
		return
	}

	output, err = v.validateFilter(ctx, dto, service)
	if err != nil {
		return
	}

	return
}

func (v *getListValidator) ValidateGetCountData(
	ctx *context.ContextModel,
	dto *in.GetListRequest,
	service services.ServicesWithGetListData,
) (
	output []model.SearchParam,
	err error,
) {
	output, err = v.validateFilter(ctx, dto, service)
	if err != nil {
		return
	}

	return
}

func checkLimit(
	validLimit []int,
	limit int,
) (
	result int,
) {

	if len(validLimit) == 0 {
		return 0
	}

	if limit < validLimit[0] {
		if validLimit[0] == -1 && len(validLimit) > 1 {
			return validLimit[1]
		}

		return validLimit[0]
	}

	oldValidLimit := validLimit[0]
	for i := 0; i < len(validLimit); i++ {
		if validLimit[i] == limit {
			return limit
		} else {
			if i > 0 {
				if oldValidLimit < limit && validLimit[i] > limit {
					if limit-oldValidLimit > validLimit[i]-limit {
						return validLimit[i]
					} else {
						return oldValidLimit
					}
				}
			}
		}

		oldValidLimit = validLimit[i]
		if i == len(validLimit)-1 {
			return validLimit[i]
		}
	}

	return 0
}

func (v *getListValidator) ValidateInputPageLimitAndOrderBy(
	ctx *context.ContextModel,
	dto *in.GetListRequest,
	service services.ServicesWithGetListData,
) error {
	if dto.Page < 1 {
		return errors.ErrFormatFieldRule.Param("PAGE", "NEED_MORE_THAN", strconv.Itoa(0)).ContextModel(ctx)
	}

	dto.Limit = checkLimit(service.GetListValidLimit(), dto.Limit)

	if dto.Limit != -1 && dto.Limit < 1 {
		return errors.ErrFormatFieldRule.Param("LIMIT", "NEED_MORE_THAN", strconv.Itoa(0)).ContextModel(ctx)
	}

	dto.Order = strings.Trim(dto.Order, orderSeparator)
	if dto.Order == "" {
		dto.Order = service.GetListValidOrderBy()[0]
	} else {

		orderSplit := strings.Split(dto.Order, ",")
		var tempOrder []string

		for i := 0; i < len(orderSplit); i++ {
			orderSplitSeparator := strings.Split(strings.Trim(orderSplit[i], " "), orderSeparator)
			if len(orderSplitSeparator) > 2 {
				return errors.ErrFormatField.Param("ORDER").ContextModel(ctx)
			}

			if !v.basicValidator.ValidateStringContainInStringArray(service.GetListValidOrderBy(), orderSplitSeparator[0]) {
				return errors.ErrFormatField.Param("ORDER").ContextModel(ctx)
			}

			if len(orderSplitSeparator) == 1 {
				orderSplitSeparator = append(orderSplitSeparator, "ASC")
			}

			tempOrder = append(tempOrder, strings.Join(orderSplitSeparator, " "))
		}

		dto.Order = strings.Join(tempOrder, ", ")

	}

	if dto.OrderIDs != "" {
		orderIDsSplit := strings.Split(dto.OrderIDs, " in ")

		if len(orderIDsSplit) != 2 {
			return errors.ErrFormatField.Param("ORDER_IDS").ContextModel(ctx)
		}

		orderIDData := strings.Split(orderIDsSplit[1], InSeparator)

		field := orderIDsSplit[0]
		if !v.basicValidator.ValidateStringContainInStringArray(service.GetListValidOrderBy(), field) {
			return errors.ErrFormatField.Param("ORDER_IDS").ContextModel(ctx)
		}

		var tempOrder []string
		for _, id := range orderIDData {
			_, err := strconv.Atoi(id)

			if err != nil {
				return errors.ErrFormatField.Param("ORDER_IDS").ContextModel(ctx)
			}

			tempOrder = append(tempOrder, id)
		}

		dto.OrderIDs = fmt.Sprintf("%s in (%s)", field, strings.Join(tempOrder, ", "))

	}

	if dto.OrderIDOperator != "ASC" && dto.OrderIDOperator != "DESC" {
		dto.OrderIDOperator = "DESC"
	}

	return nil
}

func (v *getListValidator) validateFilter(
	ctx *context.ContextModel,
	dto *in.GetListRequest,
	service services.ServicesWithGetListData,
) (
	listSearchParam []model.SearchParam,
	err error,
) {
	if strings.ToLower(dto.Connector) != "and" && strings.ToLower(dto.Connector) != "or" {
		return nil, errors.ErrFormatField.Param("CONNECTOR").ContextModel(ctx)
	}

	filter := dto.Filter

	if filter != "" {
		filterSplitComma := strings.Split(filter, ", ")
		for _, filterIndex := range filterSplitComma {
			filterIndexSplitSpace := strings.Split(filterIndex, " ")
			if len(filterIndexSplitSpace) > 2 {
				tempSearchKey := strings.Trim(filterIndexSplitSpace[0], " ")
				operator := strings.Trim(filterIndexSplitSpace[1], " ")
				searchValue := ""
				for j := 2; j < len(filterIndexSplitSpace); j++ {
					searchValue += filterIndexSplitSpace[j] + " "
				}
				searchValue = strings.Trim(searchValue, " ")

				validationResult := v.validateSearchValue(searchValue)
				if !validationResult {
					err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
					return
				}

				var listSearchKey []string
				var isNegationQuery bool

				if string(tempSearchKey[0]) == "!" {
					tempSearchKey = tempSearchKey[1:]
					isNegationQuery = true
				}

				listSearchKey, err = v.validateSearchKey(ctx, service.GetListValidSearch(), tempSearchKey, service.GetDefaultOperator())
				if err != nil {
					return
				}

				var searchKey string
				if listSearchKey != nil {
					searchKey = listSearchKey[0]
				} else {
					searchKey = tempSearchKey
				}

				if !v.isOperatorValid(tempSearchKey, operator, service.GetDefaultOperator(), listSearchKey) {
					err = errors.ErrUnknownData.Param("OPERATOR").ContextModel(ctx)
					return
				}

				if service.GetDefaultOperator()[tempSearchKey].DataType == "number" && !(operator == "in" || operator == "rng") {
					_, err = strconv.Atoi(searchValue)
					if err != nil {
						err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
						return
					}
				}

				switch operator {
				case "in":
					var searchBy model.SearchParam
					searchBy, err = listSearch(ctx, service.GetDefaultOperator(), searchKey, searchValue, operator, tempSearchKey)
					if err != nil {
						return
					}

					searchBy.NegationQuery = isNegationQuery
					listSearchParam = append(listSearchParam, searchBy)

				case "rng":
					if listSearchKey != nil {
						err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
						return
					}

					var tempListSearchParam []model.SearchParam
					tempListSearchParam, err = rangeSearch(ctx, service.GetDefaultOperator(), tempSearchKey, searchValue, isNegationQuery)
					if err != nil {
						return
					}

					listSearchParam = append(listSearchParam, tempListSearchParam...)
				default:
					listSearchParam = append(listSearchParam, model.SearchParam{
						DataType:       service.GetDefaultOperator()[searchKey].DataType,
						SearchKey:      tempSearchKey,
						SearchOperator: operator,
						SearchValue:    searchValue,
						NegationQuery:  isNegationQuery,
					})
				}
			} else {
				err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
				return
			}
		}
	}

	st := reflect.TypeOf(service)
	vt := reflect.ValueOf(service)

	m, ok := st.MethodByName("GetListEnum")

	var enum map[string][]string

	if ok {
		val := m.Func.Call([]reflect.Value{vt})
		if len(val) > 0 {
			inter := val[0].Interface()
			enum, ok = inter.(map[string][]string)
			if !ok && enum == nil {
				return
			}
		}
	}

	for i := 0; i < len(listSearchParam); i++ {
		if listSearchParam[i].DataType == "enum" {
			if value, valid := listSearchParam[i].SearchValue.(string); valid {
				if valid := v.basicValidator.ValidateStringContainInStringArray(enum[listSearchParam[i].SearchKey], value); !valid {
					return nil, errors.ErrUnknownData.Param(enum[listSearchParam[i].SearchKey]).ContextModel(ctx)
				}
			}
		}
	}

	return
}

func listSearch(
	ctx *context.ContextModel,
	validOperator map[string]model.DefaultOperator,
	searchKey string,
	searchValue string,
	operator string,
	tempSearchKey string,
) (
	searchBy model.SearchParam,
	err error,
) {
	switch validOperator[searchKey].DataType {
	case "char":
		var tempListString []interface{}
		tempList := strings.Split(searchValue, "-")
		for _, value := range tempList {
			value = strings.ReplaceAll(value, "~", "-")
			tempListString = append(tempListString, value)
		}
		searchBy = model.SearchParam{
			DataType:       validOperator[searchKey].DataType,
			SearchKey:      tempSearchKey,
			SearchOperator: operator,
			SearchValue:    tempListString,
		}

	case "number":
		var listID []interface{}
		tempID := strings.Split(searchValue, "-")
		for _, id := range tempID {
			idInt64, errS := strconv.Atoi(id)
			if errS != nil {
				err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
				return
			}
			listID = append(listID, int64(idInt64))
		}
		searchBy = model.SearchParam{
			DataType:       validOperator[searchKey].DataType,
			SearchKey:      tempSearchKey,
			SearchOperator: operator,
			SearchValue:    listID,
		}
	default:
		err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
	}

	return
}

func rangeSearch(
	ctx *context.ContextModel,
	validOperator map[string]model.DefaultOperator,
	searchKey string,
	searchValue string,
	isNegationQuery bool,
) (
	searchBy []model.SearchParam,
	err error,
) {
	splitMinusRange := strings.Split(searchValue, rangeSeparator)

	if len(splitMinusRange) != 2 {
		err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
		return
	}

	switch validOperator[searchKey].DataType {
	case "number":
		var startNumber float64
		var endNumber float64
		var err error

		if startNumber, err = strconv.ParseFloat(splitMinusRange[0], 64); err != nil {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return searchBy, err
		}

		if endNumber, err = strconv.ParseFloat(splitMinusRange[1], 64); err != nil {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return searchBy, err
		}

		if startNumber > endNumber {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return searchBy, err
		}

		searchBy = append(searchBy, model.SearchParam{
			DataType:       validOperator[searchKey].DataType,
			SearchKey:      searchKey,
			SearchOperator: "be",
			SearchValue:    startNumber,
			NegationQuery:  isNegationQuery,
		})

		searchBy = append(searchBy, model.SearchParam{
			DataType:       validOperator[searchKey].DataType,
			SearchKey:      searchKey,
			SearchOperator: "lq",
			SearchValue:    endNumber,
			NegationQuery:  isNegationQuery,
		})
	case "date", "time", "unix", "unix-nano", "unix-milli":
		var startTime time.Time
		var endTime time.Time

		startTime, err = time.Parse(constanta.DefaultTimeFormat, splitMinusRange[0])
		if err != nil {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return
		}

		endTime, err = time.Parse(constanta.DefaultTimeFormat, splitMinusRange[1])
		if err != nil {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return
		}

		result := startTime.Before(endTime)
		if !result {
			err = errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			return
		}

		searchBy = append(searchBy, model.SearchParam{
			DataType:            validOperator[searchKey].DataType,
			SearchKey:           searchKey,
			SearchOperator:      "be",
			SearchValue:         splitMinusRange[0],
			NegationQuery:       isNegationQuery,
			FormatedSearchValue: reformatDateValue(validOperator[searchKey].DataType, startTime),
		})

		searchBy = append(searchBy, model.SearchParam{
			DataType:            validOperator[searchKey].DataType,
			SearchKey:           searchKey,
			SearchOperator:      "lq",
			SearchValue:         splitMinusRange[1],
			NegationQuery:       isNegationQuery,
			FormatedSearchValue: reformatDateValue(validOperator[searchKey].DataType, endTime),
		})
	}

	return
}

func reformatDateValue(
	dataType string,
	date time.Time,
) interface{} {
	switch dataType {
	case "unix":
		return date.Unix()
	case "unix-nano":
		return date.UnixNano()
	case "unix-milli":
		return date.UnixMilli()
	}

	return nil
}

func (v getListValidator) validateSearchKey(
	ctx *context.ContextModel,
	validSearchKey []string,
	searchKey string,
	validOperator map[string]model.DefaultOperator,
) (
	[]string,
	error,
) {
	if strings.Contains(searchKey, "&&") || strings.Contains(searchKey, "||") {
		ListSearchKey := regexp.MustCompile("[&][&]|[|][|]").Split(searchKey, -1)

		lastValue := ""
		for _, key := range ListSearchKey {
			if !v.basicValidator.ValidateStringContainInStringArray(validSearchKey, key) {
				return nil, errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
			}

			if lastValue == "" {
				lastValue = validOperator[key].DataType
			} else {
				if validOperator[key].DataType != lastValue {
					return nil, errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
				}
			}
		}

		return ListSearchKey, nil
	}

	if !v.basicValidator.ValidateStringContainInStringArray(validSearchKey, searchKey) {
		return nil, errors.ErrFormatField.Param("FILTER").ContextModel(ctx)
	}

	return nil, nil
}

func (v getListValidator) validateSearchValue(searchValue string) bool {
	if searchValue == "</nil>" {
		return true
	}

	searchValueRegex := regexp.MustCompile("^[a-zA-Z0-9-._@ ()+>:~]+$")
	return searchValueRegex.MatchString(searchValue)
}

func (v getListValidator) isOperatorValid(key string, operator string, validOperator map[string]model.DefaultOperator, listSearchKey []string) bool {
	if listSearchKey != nil {
		for _, tempKey := range listSearchKey {
			if !v.validateOperator(validOperator, tempKey, operator) {
				return false
			}
		}
		return true
	} else {
		return v.validateOperator(validOperator, key, operator)
	}
}

func (v getListValidator) validateOperator(validOperator map[string]model.DefaultOperator, searchKey string, operator string) bool {
	if validOperator[searchKey].Operator == nil {
		return false
	} else {
		return v.basicValidator.ValidateStringContainInStringArray(validOperator[searchKey].Operator, operator)
	}
}
