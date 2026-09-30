package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/nexsoft-git/nexcommon/regex"

	"github.com/nexsoft-git/nexcommon/constanta"
)

var RULE_ENGINE_ENABLE_LOG = false

type LegacyRuleEngine struct {
	CustomerAttribute     string `json:"customer_attribute"`
	CustomerAttributeType string `json:"customer_attribute_type"`
	Operator              string `json:"operator"`
	Value1                string `json:"value_1"`
	Value2                string `json:"value_2"`
	OpenBracket           bool   `json:"open_bracket"`
	CloseBracket          bool   `json:"close_bracket"`
	LogicalConnector      string `json:"logical_connector"`
}

type RuleEngine struct {
	Formula   []RuleEngineFormula `json:"formula"`
	Connector []string            `json:"connector"`
}

type RuleEngineFormula struct {
	Attribute  string     `json:"attribute"`
	Type       string     `json:"type"`
	Operator   string     `json:"operator"`
	Val        string     `json:"val"`
	AddOns     RuleEngine `json:"add_ons"`
	AddOnsConn string     `json:"add_ons_conn"`
}

func (r RuleEngine) Check(
	data map[string]interface{},
) (bool, string) {
	result := make(map[string]string)
	res := r.check(data, r.Formula, r.Connector, result)
	return res, strings.ReplaceAll(result["result"], "  ", " ")
}

func (r RuleEngine) check(
	data map[string]interface{},
	formula []RuleEngineFormula,
	connector []string,
	result map[string]string,
) bool {
	var checkerResult = true

	for i := 0; i < len(formula); i++ {

		var formulaResult = validateFormula(data, formula[i])

		if RULE_ENGINE_ENABLE_LOG {
			fmt.Print(formula[i].Attribute, " ", formula[i].Operator, " ", formula[i].Val, " ")
		}
		result["result"] = result["result"] + fmt.Sprintf("%v ", formulaResult)

		if formula[i].AddOns.Formula != nil {
			if RULE_ENGINE_ENABLE_LOG {
				fmt.Print(formula[i].AddOnsConn, " ( ")
			}
			result["result"] = result["result"] + " " + formula[i].AddOnsConn + " ( "

			formulaResult = validateConnector(formulaResult, r.check(data, formula[i].AddOns.Formula, formula[i].AddOns.Connector, result), formula[i].AddOnsConn)

			if RULE_ENGINE_ENABLE_LOG {
				fmt.Print(")", " ")
			}
			result["result"] = result["result"] + " ) "
		}

		if i > 0 {
			checkerResult = validateConnector(checkerResult, formulaResult, connector[i-1])
		} else {
			checkerResult = formulaResult
		}

		if i < len(formula)-1 {
			result["result"] = result["result"] + " " + connector[i] + " "
			if RULE_ENGINE_ENABLE_LOG {
				fmt.Print(connector[i], " ")
			}
		}
	}

	return checkerResult
}

func nilConverter(
	values map[string]interface{},
) map[string]interface{} {
	for key, value := range values {
		switch v := value.(type) {
		case string:
			if v == "" {
				values[key] = nil
			}
		case time.Time:
			if v.IsZero() {
				values[key] = nil
			}
		case int32:
			if v == 0 {
				values[key] = nil
			}
		case float64:
			if v == 0 {
				values[key] = nil
			}
		}
	}

	return values
}

func validateFormula(
	data map[string]interface{},
	formula RuleEngineFormula,
) bool {

	var textVal string
	var numberVal int64
	var floatVal float64
	var booleanVal bool
	var dateVal time.Time
	var valid bool

	var format = constanta.DateOnlyTimeFormat

	if formula.Operator == constanta.EMPTY_OPERATOR || formula.Operator == constanta.NOT_EMPTY_OPERATOR {
		data = nilConverter(data)
	}

	if formula.Operator == constanta.EMPTY_OPERATOR && data[formula.Attribute] == nil {
		return true
	} else if formula.Operator == constanta.NOT_EMPTY_OPERATOR && data[formula.Attribute] != nil {
		return true
	}

	switch formula.Type {
	case constanta.NUMBER_TYPE:
		if numberVal, valid = readNumberField(data, formula); !valid {
			return false
		}
	case constanta.FLOAT_TYPE:
		if floatVal, valid = readFloatField(data, formula); !valid {
			return false
		}
	case constanta.BOOLEAN_TYPE:
		if booleanVal, valid = data[formula.Attribute].(bool); !valid {
			return false
		}
	case constanta.DATETIME_TYPE:
		format = constanta.DefaultDBTimeFormat
		if dateVal, valid = data[formula.Attribute].(time.Time); !valid {
			return false
		}

		if dateVal.IsZero() {
			return false
		}
	case constanta.DATE_TYPE:
		if dateVal, valid = data[formula.Attribute].(time.Time); !valid {
			return false
		}

		if dateVal.IsZero() {
			return false
		}
	default:
		if textVal, valid = data[formula.Attribute].(string); !valid {
			return false
		}
	}

	if formula.Operator == constanta.LAST_OPERATOR && (formula.Type == constanta.DATE_TYPE || formula.Type == constanta.DATETIME_TYPE) {
		data, _ := strconv.Atoi(formula.Val)

		timeStr := time.Now().Format("2006-01-02")
		timeNow, _ := time.Parse("2006-01-02", timeStr)

		if dateVal.Unix() >= timeNow.Add(-time.Duration(data)*time.Hour*24).Unix() {
			return true
		}
	}

	if formula.Operator == constanta.BETWEEN_OPERATOR {
		splited := strings.Split(formula.Val, constanta.BETWEEN_KEY_SEPARATOR)
		if len(splited) != 2 {
			return false
		}

		switch formula.Type {
		case constanta.DATE_TYPE, constanta.DATETIME_TYPE:
			formulaValueMin, _ := time.Parse(format, splited[0])
			formulaValueMax, _ := time.Parse(format, splited[1])

			if dateVal.Unix() >= formulaValueMin.Unix() && dateVal.Unix() <= formulaValueMax.Unix() {
				return true
			}
		case constanta.NUMBER_TYPE:
			formulaValueMin, _ := strconv.Atoi(splited[0])
			formulaValueMax, _ := strconv.Atoi(splited[1])

			if numberVal >= int64(formulaValueMin) && numberVal <= int64(formulaValueMax) {
				return true
			}
		case constanta.FLOAT_TYPE:
			formulaValueMin, _ := strconv.ParseFloat(splited[0], 64)
			formulaValueMax, _ := strconv.ParseFloat(splited[1], 64)

			if floatVal >= formulaValueMin && floatVal <= formulaValueMax {
				return true
			}
		}
	}

	switch formula.Type {
	case constanta.TEXT_TYPE:
		switch formula.Operator {
		case constanta.EQUALS_OPERATOR:
			return textVal == formula.Val
		case constanta.NOT_EQUALS_OPERATOR:
			return textVal != formula.Val
		case constanta.CONTAINS_OPERATOR:
			return strings.Contains(textVal, formula.Val)
		case constanta.DOES_NOT_CONTAINS_OPERATOR:
			return !strings.Contains(textVal, formula.Val)
		case constanta.IS_VALID_OPERATOR:
			attributeData, isValid := data[formula.Attribute].(string)

			if !isValid {
				return isValid
			}

			var regexUsed string

			switch formula.Attribute {
			case constanta.NIK_ATTRIBUTE:
				regexUsed = regex.NIK
			case constanta.PHONE_ATTRIBUTE:
				regexUsed = regex.FORMAT_PHONE_NUMBER_ND
			}

			if regexUsed != "" {
				checker := regexp.MustCompile(regexUsed)
				return checker.MatchString(attributeData)
			} else {
				return false
			}
		}
	case constanta.NUMBER_TYPE:
		formulaValue, _ := strconv.Atoi(formula.Val)

		switch formula.Operator {
		case constanta.EQUALS_OPERATOR:
			return numberVal == int64(formulaValue)
		case constanta.NOT_EQUALS_OPERATOR:
			return numberVal != int64(formulaValue)
		case constanta.GREATER_THAN_OPERATOR:
			return numberVal > int64(formulaValue)
		case constanta.GREATER_THAN_EQUAL_OPERATOR:
			return numberVal >= int64(formulaValue)
		case constanta.LESS_THAN_OPERATOR:
			return numberVal < int64(formulaValue)
		case constanta.LESS_THAN_EQUAL_OPERATOR:
			return numberVal <= int64(formulaValue)
		}
	case constanta.FLOAT_TYPE:
		formulaValue, _ := strconv.ParseFloat(formula.Val, 64)

		switch formula.Operator {
		case constanta.EQUALS_OPERATOR:
			return floatVal == formulaValue
		case constanta.NOT_EQUALS_OPERATOR:
			return floatVal != formulaValue
		case constanta.GREATER_THAN_OPERATOR:
			return floatVal > formulaValue
		case constanta.GREATER_THAN_EQUAL_OPERATOR:
			return floatVal >= formulaValue
		case constanta.LESS_THAN_OPERATOR:
			return floatVal < formulaValue
		case constanta.LESS_THAN_EQUAL_OPERATOR:
			return floatVal <= formulaValue
		}
	case constanta.BOOLEAN_TYPE:
		formulaValue := formula.Val == constanta.BOOL_TRUE_VALUE
		if formula.Operator == constanta.EQUALS_OPERATOR {
			return booleanVal == formulaValue
		} else {
			return booleanVal != formulaValue
		}
	case constanta.DATETIME_TYPE, constanta.DATE_TYPE:
		formulaValue, _ := time.Parse(format, formula.Val)

		switch formula.Operator {
		case constanta.EQUALS_OPERATOR:
			return dateVal.Unix() == formulaValue.Unix()
		case constanta.NOT_EQUALS_OPERATOR:
			return dateVal.Unix() != formulaValue.Unix()
		case constanta.GREATER_THAN_OPERATOR:
			return dateVal.Unix() > formulaValue.Unix()
		case constanta.GREATER_THAN_EQUAL_OPERATOR:
			return dateVal.Unix() >= formulaValue.Unix()
		case constanta.LESS_THAN_OPERATOR:
			return dateVal.Unix() < formulaValue.Unix()
		case constanta.LESS_THAN_EQUAL_OPERATOR:
			return dateVal.Unix() <= formulaValue.Unix()
		}
	}

	return false
}

func readNumberField(
	data map[string]interface{},
	formula RuleEngineFormula,
) (
	int64,
	bool,
) {
	val, valid := data[formula.Attribute].(int64)
	if !valid {
		val32, valid := data[formula.Attribute].(int32)
		if !valid {
			valInt, valid := data[formula.Attribute].(int)
			if !valid {
				return 0, false
			} else {
				val = int64(valInt)
			}
		} else {
			val = int64(val32)
		}
	}
	return val, true
}

func readFloatField(
	data map[string]interface{},
	formula RuleEngineFormula,
) (
	float64,
	bool,
) {
	val, valid := data[formula.Attribute].(float64)
	if !valid {
		val32, valid := data[formula.Attribute].(float32)
		if !valid {
			return 0, false
		} else {
			val = float64(val32)
		}
	}
	return val, true
}

func validateConnector(
	field1 bool,
	field2 bool,
	connector string,
) bool {

	var result bool

	if connector == constanta.AND_CONNECTOR {
		result = field1 && field2
	} else if connector == constanta.OR_CONNECTOR {
		result = field1 || field2
	} else if connector == constanta.AND_NOT_CONNECTOR {
		result = field1 && !field2
	} else {
		result = field1 || !field2
	}

	return result

}
