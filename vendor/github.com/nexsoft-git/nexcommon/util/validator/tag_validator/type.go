package tag_validator

import (
	"reflect"

	"github.com/nexsoft-git/nexcommon/context"
)

type TagValidator interface {
	AddEnum(
		key string,
		value ...string,
	) *tagValidator

	AddDateFormat(
		key string,
		dateFormat string,
	) *tagValidator

	AddRegex(
		key string,
		regex string,
		name string,
	) *tagValidator

	AddAutoFix(
		key string,
		f func(string) string,
	) *tagValidator

	ValidateByTag(
		ctx *context.ContextModel,
		dto interface{},
		function string,
		menu string,
	) (
		err error,
	)

	GetEnum(key string) []string
	GetDateFormat(key string) string
	GetRegex(key string) regexValue
	GetMinMaxValue(field reflect.StructField) (min int, isMinFound bool, max int, isMaxFound bool)
}

type regexValue struct {
	Regex    string
	ruleName string
}

const DURATION_TIME_FORMAT_TYPE = "duration"
