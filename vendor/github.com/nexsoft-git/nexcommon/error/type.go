package error

import (
	"strings"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/out"
)

type Formator interface {
	ReformatErrorMessage(
		param errorMessageParam,
	) out.DefaultErrorResponse

	DefaultLanguage(
		lang string,
	) *formator

	Version(
		version string,
	) *formator

	DefaultInternalCode(
		defaultCode string,
	) *formator
}

type ErrorParam struct {
	Param       interface{}
	IsConverted bool
}

type Converter func(...interface{}) map[string]ErrorParam

type UnbundledErrorMessages struct {
	status   int
	code     error
	reason   string
	param    []interface{}
	function Converter
	ctx      *context.ContextModel
}

func (u *UnbundledErrorMessages) ContextModel(
	ctx *context.ContextModel,
) *UnbundledErrorMessages {
	u.ctx = ctx
	return u
}

func (e UnbundledErrorMessages) Error() string {
	if _formator == nil {
		return e.code.Error()
	} else {
		var result string
		language := defaultLanguage

		param := make(map[string]interface{})
		if e.function != nil {
			tempParam := e.function(e.param...)

			if e.ctx != nil {
				if e.ctx.AuthAccessTokenModel.Locale != "" {
					language = e.ctx.AuthAccessTokenModel.Locale
				}
			}

			for key := range tempParam {
				param[key] = tempParam[key].Param
				val, _ := tempParam[key].Param.(string)

				if tempParam[key].IsConverted {
					param[key] = _formator.bundles.ReadMessageBundle("common.constanta", strings.ToUpper(val), language, nil)
				}
			}
		}

		result = _formator.bundles.ReadMessageBundle("common.error", e.code.Error(), language, param)

		if e.reason != "" {
			result += " ,Reason : " + e.reason
		}

		return result
	}
}

func (e *UnbundledErrorMessages) Reason(reason string) *UnbundledErrorMessages {
	e.reason = reason
	return e
}

func (e *UnbundledErrorMessages) Param(param ...interface{}) *UnbundledErrorMessages {
	e.param = param
	return e
}
