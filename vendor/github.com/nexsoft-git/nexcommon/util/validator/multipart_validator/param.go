package multipart_validator

import (
	"net/http"

	"github.com/nexsoft-git/nexcommon/context"
)

func NewValidateMultipartParam(
	ctx *context.ContextModel,
	r *http.Request,
	dto interface{},
) *validateMultipartParam {
	return &validateMultipartParam{
		r:   r,
		dto: dto,
		ctx: ctx,
	}
}

type validateMultipartParam struct {
	ctx               *context.ContextModel
	r                 *http.Request
	dto               interface{}
	menu              string
	functionValidator string
}

func (v *validateMultipartParam) Menu(menu string) *validateMultipartParam {
	v.menu = menu
	return v
}

func (v *validateMultipartParam) FunctionValidator(functionValidator string) *validateMultipartParam {
	v.functionValidator = functionValidator
	return v
}
