package tag_validator

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/context"
	errors "github.com/nexsoft-git/nexcommon/error"
	rgx "github.com/nexsoft-git/nexcommon/regex"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
)

func NewTagValidator() TagValidator {
	validator := &tagValidator{
		basic: basic_validator.BasicValidator{},
	}

	validator.build()
	return validator
}

type tagValidator struct {
	enum       map[string][]string
	dateFormat map[string]string
	regex      map[string]regexValue
	autoFix    map[string]func(string) string
	basic      basic_validator.BasicValidator
}

func (v *tagValidator) GetEnum(key string) []string {
	return v.enum[key]
}

func (v *tagValidator) GetDateFormat(key string) string {
	return v.dateFormat[key]

}

func (v *tagValidator) GetRegex(key string) regexValue {
	return v.regex[key]
}

func (v *tagValidator) AddEnum(key string, value ...string) *tagValidator {
	v.enum[key] = value
	return v
}

func (v *tagValidator) AddDateFormat(key string, dateFormat string) *tagValidator {
	v.dateFormat[key] = dateFormat
	return v
}

func (v *tagValidator) AddRegex(key string, regex string, name string) *tagValidator {
	v.regex[key] = regexValue{
		Regex:    regex,
		ruleName: name,
	}
	return v
}

func (v *tagValidator) AddAutoFix(key string, f func(string) string) *tagValidator {
	v.autoFix[key] = f
	return v
}

func (v *tagValidator) build() {
	v.enum = make(map[string][]string)
	v.enum["record_status"] = []string{"A", "N", "P"}
	v.enum["sharing_permission"] = []string{"edit", "view"}
	v.enum["boolean_permission"] = []string{"Y", "N"}

	v.dateFormat = make(map[string]string)
	v.dateFormat["default"] = constanta.DefaultTimeFormat
	v.dateFormat["date_only"] = constanta.DateOnlyTimeFormat
	v.dateFormat[DURATION_TIME_FORMAT_TYPE] = DURATION_TIME_FORMAT_TYPE

	v.regex = make(map[string]regexValue)
	v.autoFix = map[string]func(string) string{}

	v.regex["profile_name"] = regexValue{
		Regex:    rgx.PROFILE_NAME,
		ruleName: "PROFILE_NAME_REGEX_MESSAGE",
	}
	v.regex["directory_name"] = regexValue{
		Regex:    rgx.DIRECTORY_NAME,
		ruleName: "DIRECTORY_NAME_REGEX_MESSAGE",
	}
	v.regex["name"] = regexValue{
		Regex:    rgx.NAME_STANDARD,
		ruleName: "NAME_REGEX_MESSAGE",
	}
	v.regex["text_only"] = regexValue{
		Regex:    rgx.TEXT_ONLY,
		ruleName: "DESCRIPTION_REGEX_MESSAGE",
	}
	v.regex["alphanumeric"] = regexValue{
		Regex:    rgx.ALPHANUMERIC,
		ruleName: "ALPHANUMERIC_REGEX",
	}
	v.regex["country_code"] = regexValue{
		Regex:    rgx.COUNTRY_CODE,
		ruleName: "COUNTRY_CODE_REGEX",
	}
	v.regex["email"] = regexValue{
		Regex:    rgx.EMAIL_REGEX,
		ruleName: "EMAIL_REGEX",
	}
	v.regex["phone"] = regexValue{
		Regex:    rgx.PHONE_NUMBER_WITH_COUNTRY_CODE,
		ruleName: "PHONE_NUMBER_REGEX",
	}
	v.regex["additional_info"] = regexValue{
		Regex:    rgx.ADDITIONAL_INFO,
		ruleName: "ADDITIONAL_INFO_REGEX",
	}
	v.regex["username"] = regexValue{
		Regex:    rgx.USERNAME,
		ruleName: "USERNAME_REGEX",
	}
	v.regex["lowercase"] = regexValue{
		Regex:    rgx.LOWERCASE,
		ruleName: "LOWERCASE_REGEX",
	}
	v.regex["upercase"] = regexValue{
		Regex:    rgx.UPERCASE,
		ruleName: "UPERCASE_REGEX",
	}
	v.regex["npwp"] = regexValue{
		Regex:    rgx.NPWP,
		ruleName: "NPWP_REGEX",
	}
	v.regex["nik"] = regexValue{
		Regex:    rgx.NIK,
		ruleName: "NIK_REGEX",
	}
	v.regex["fax"] = regexValue{
		Regex:    rgx.FAX,
		ruleName: "FAX_REGEX",
	}
	v.regex["lowercase_number"] = regexValue{
		Regex:    rgx.LOWERCASE_AND_NUMBER,
		ruleName: "LOWERCASE_AND_NUMBER_REGEX",
	}
	v.regex["numeric"] = regexValue{
		Regex:    rgx.LONG_NUMERIC,
		ruleName: "LONG_NUMERIC_REGEX",
	}
	v.regex["permission"] = regexValue{
		Regex:    rgx.PERMISSION,
		ruleName: "PERMISSION_REGEX",
	}
	v.regex["profile_name"] = regexValue{
		Regex:    rgx.PROFILE_NAME,
		ruleName: "PROFILE_NAME_REGEX",
	}
	v.regex["data_scope"] = regexValue{
		Regex:    rgx.DATA_SCOPE,
		ruleName: "DATA_SCOPE_REGEX",
	}
	v.regex["directory_name"] = regexValue{
		Regex:    rgx.DIRECTORY_NAME,
		ruleName: "DIRECTORY_NAME_REGEX",
	}

	v.autoFix["uppercase"] = func(param string) string {
		return strings.ToUpper(param)
	}
	v.autoFix["lowercase"] = func(param string) string {
		return strings.ToLower(param)
	}
	v.autoFix["title"] = func(param string) string {
		return strings.Title(strings.ToLower(param))
	}
	v.autoFix["trim"] = func(param string) string {
		return strings.Trim(param, " ")
	}
	v.autoFix["filename"] = func(param string) string {
		param = strings.ReplaceAll(param, "\\", "/")
		param = strings.Trim(param, "./")
		param = strings.Trim(param, "/")
		return param
	}

}

var (
	IntType   = []string{reflect.Int64.String(), reflect.Int.String(), reflect.Int32.String()}
	FloatType = []string{reflect.Float32.String(), reflect.Float64.String()}
)

func (v *tagValidator) ValidateByTag(
	ctx *context.ContextModel,
	dto interface{},
	functionValidator string,
	menu string,
) (
	err error,
) {

	reflectType := reflect.TypeOf(dto).Elem()
	reflectValue := reflect.ValueOf(dto).Elem()

	max := 0
	min := 0
	isMinFound := false
	isMaxFound := false

	for i := 0; i < reflectType.NumField(); i++ {
		currentField := reflectType.Field(i)
		currentValue := reflectValue.FieldByName(currentField.Name)

		if currentField.Name == "AbstractDTO" {
			continue
		}

		if currentField.Type.Kind() == reflect.Struct && currentField.Type.String() != "time.Time" {
			newDTO := currentValue.Addr().Interface()
			err = v.ValidateByTag(ctx, newDTO, functionValidator, menu)
			if err != nil {
				return
			}
		}

		if currentField.Type.Kind() == reflect.String {
			autoFix := currentField.Tag.Get("auto_fix")
			split := strings.Split(autoFix, ",")
			for i := 0; i < len(split); i++ {
				f, valid := v.autoFix[autoFix]
				if valid {
					currentValue.SetString(f(currentValue.String()))
				}
			}
		}

		required := currentField.Tag.Get("required")
		jsonField := strings.ToUpper(currentField.Tag.Get("json"))
		requiredArray := strings.Split(required, ",")
		reservedValues := currentField.Tag.Get("reserved")
		if reservedValues != "" {
			reservedValue := strings.Split(reservedValues, ",")
			if v.basic.ValidateStringContainInStringArray(reservedValue, currentValue.String()) {
				err = errors.ErrReservedValueString.Param(jsonField).ContextModel(ctx)
				return
			}
		}

		empty := currentField.Tag.Get("empty")
		if empty == "allowed" {
			if currentValue.IsZero() && currentField.Type.String() != "time.Time" {
				continue
			}
		}

		minEqMax := false
		if v.basic.ValidateStringContainInStringArray(requiredArray, menu) {
			defaultValue := currentField.Tag.Get("default")
			min, isMinFound, max, isMaxFound = v.GetMinMaxValue(currentField)
			minEqMax = min == max

			if v.basic.ValidateStringContainInStringArray(IntType, currentField.Type.String()) {
				if currentValue.IsZero() {
					valueIn, _ := strconv.Atoi(defaultValue)
					currentValue.SetInt(int64(valueIn))
				}

				value := currentValue.Int()
				if isMinFound {
					if min != 0 && int(value) == 0 {
						err = errors.ErrEmptyField.Param(jsonField).ContextModel(ctx)
						return
					}
					if int(value) < min {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_MORE_THAN"), strconv.Itoa(min)).ContextModel(ctx)
					}
				}
				if isMaxFound {
					if int(value) > max {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_LESS_THAN"), strconv.Itoa(max)).ContextModel(ctx)
					}
				}
			} else if v.basic.ValidateStringContainInStringArray(FloatType, currentField.Type.String()) {
				if currentValue.IsZero() {
					valueIn, _ := strconv.ParseFloat(defaultValue, 64)
					currentValue.SetFloat(valueIn)
				}

				value := currentValue.Float()
				if isMinFound {
					if value < float64(min) {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_MORE_THAN"), strconv.Itoa(min)).ContextModel(ctx)
					}
				}
				if isMaxFound {
					if value > float64(max) {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_LESS_THAN"), strconv.Itoa(max)).ContextModel(ctx)
					}
				}
			} else if reflect.String.String() == currentField.Type.String() {
				currentValue.SetString(strings.Trim(currentValue.String(), " "))
				if currentValue.IsZero() {
					currentValue.SetString(defaultValue)
				}

				if currentValue.IsZero() {
					return errors.ErrEmptyField.Param(jsonField).ContextModel(ctx)
				}

				value := currentValue.String()
				err = v.ValidateMinMaxString(ctx, value, jsonField, min, max)
				if err != nil {
					return
				}

				enumField := currentField.Tag.Get("enum")
				if enumField != "" {
					if !v.basic.ValidateStringContainInStringArray(v.enum[enumField], currentValue.String()) {
						err = errors.ErrFormatFieldRule.Param(jsonField, "FIXED_VALUE", strings.Join(v.enum[enumField], " , ")).ContextModel(ctx)
						return
					}
				}

				regexField := currentField.Tag.Get("regex")
				if regexField != "" {
					if v.regex[regexField].Regex != "" {
						if len(value) > 0 || min != 0 {
							if !regexp.MustCompile(v.regex[regexField].Regex).MatchString(currentValue.String()) {
								return errors.ErrFormatFieldRule.Param(jsonField, v.regex[regexField].ruleName, "").ContextModel(ctx)
							}
						}

					}
				}
			} else if currentField.Type.String() == "time.Time" {
				var timeObject time.Time
				dateFormatTag := currentField.Tag.Get("dateFormat")
				strField := currentField.Name + "Str"
				timeFormatUsed := v.dateFormat["default"]

				dateSplit := strings.Split(dateFormatTag, ",")
				var timeValid = false

				for i := 0; i < len(dateSplit); i++ {
					if v.dateFormat[dateSplit[i]] != "" {
						timeFormatUsed = v.dateFormat[dateSplit[i]]
					}

					field, valid := reflectType.FieldByName(strField)

					if valid {

						val := reflectValue.FieldByName(strField).String()
						if val == "" && empty == "allowed" {
							timeValid = true
							continue
						}

						jsonField = field.Tag.Get("json")
						timeObject, err = v.TimeStrToTime(ctx, val, jsonField, timeFormatUsed)

						if err != nil {
							continue
						}

						currentValue.Set(reflect.ValueOf(timeObject))
						timeValid = true
						break

					} else {
						return errors.ErrUnknownData.Param(strField).ContextModel(ctx)
					}
				}

				if !timeValid {
					return errors.ErrFormatField.Param(jsonField).ContextModel(ctx)
				}

			} else if currentValue.Kind() == reflect.Slice {
				if isMinFound {
					if currentValue.Len() == 0 {
						return errors.ErrEmptyField.Param(jsonField).ContextModel(ctx)
					}
					if currentValue.Len() < min {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_MORE_THAN"), strconv.Itoa(min)).ContextModel(ctx)
					}
				}

				if isMaxFound {
					if currentValue.Len() > max {
						return errors.ErrFormatFieldRule.Param(jsonField, getRuleName(minEqMax, "NEED_LESS_THAN"), strconv.Itoa(max)).ContextModel(ctx)
					}
				}

				for i := 0; i < currentValue.Len(); i++ {
					temp := currentValue.Index(i)
					if temp.Type().String() == reflect.Struct.String() || temp.Type().Kind().String() == reflect.Struct.String() {
						newDTO := currentValue.Index(i).Addr().Interface()
						err = v.ValidateByTag(ctx, newDTO, functionValidator, menu)
						if err != nil {
							return
						}
					}

					if temp.Type().Kind() == reflect.String {
						enumField := currentField.Tag.Get("enum")
						if enumField != "" {
							if !v.basic.ValidateStringContainInStringArray(v.enum[enumField], temp.String()) {
								err = errors.ErrFormatFieldRule.Param(jsonField, "FIXED_VALUE", strings.Join(v.enum[enumField], " , ")).ContextModel(ctx)
								return
							}
						}
					}
				}
			}
		}
	}

	if functionValidator != "" {
		st := reflect.TypeOf(dto)
		vt := reflect.ValueOf(dto)

		m, ok := st.MethodByName(functionValidator)

		if ok {
			val := m.Func.Call([]reflect.Value{vt})
			if len(val) > 0 {
				if !val[0].IsNil() {
					return val[0].Interface().(error)
				}
			}
		}
	}

	return
}

func getRuleName(minEqmax bool, ruleName string) string {
	if minEqmax {
		return "EQUAL"
	}

	return ruleName
}

func (v *tagValidator) GetMinMaxValue(field reflect.StructField) (min int, isMinFound bool, max int, isMaxFound bool) {
	maxStr, isMaxFound := field.Tag.Lookup("max")
	minStr, isMinFound := field.Tag.Lookup("min")

	min, _ = strconv.Atoi(minStr)
	max, _ = strconv.Atoi(maxStr)

	return
}

func (v *tagValidator) TimeStrToTime(
	ctx *context.ContextModel,
	timeStr string,
	fieldName string,
	format string,
) (
	output time.Time,
	err error,
) {

	if format == DURATION_TIME_FORMAT_TYPE {
		var duration time.Duration
		duration, err = v.basic.ValidateISO8601Duration(timeStr)
		if err != nil {
			err = errors.ErrFormatField.Param(fieldName).ContextModel(ctx)
			return
		}
		output = time.Now().Add(duration)
	} else {
		output, err = time.Parse(format, timeStr)
		if err != nil {
			err = errors.ErrFormatField.Param(fieldName).ContextModel(ctx)
			return
		}
	}

	return output, nil
}

func (v *tagValidator) ValidateMinMaxString(
	ctx *context.ContextModel,
	inputStr string,
	fieldName string,
	min int,
	max int,
) error {
	minEqMax := min == max
	if min != 0 {
		if len(inputStr) == 0 {
			return errors.ErrEmptyField.Param(fieldName).ContextModel(ctx)
		}
		if len(inputStr) < min {
			if min == 1 {
				return errors.ErrEmptyField.Param(fieldName).ContextModel(ctx)
			} else {
				return errors.ErrFormatFieldRule.Param(fieldName, getRuleName(minEqMax, "NEED_MORE_THAN"), strconv.Itoa(min)).ContextModel(ctx)
			}
		}
	}
	if max != 0 {
		if len(inputStr) > max {
			return errors.ErrFormatFieldRule.Param(fieldName, getRuleName(minEqMax, "NEED_LESS_THAN"), strconv.Itoa(max)).ContextModel(ctx)
		}
	}

	return nil
}
