package get_list_validator

import (
	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services"
)

type GetListValidator interface {
	ValidateGetListData(
		ctx *context.ContextModel,
		dto *in.GetListRequest,
		service services.ServicesWithGetListData,
	) (
		output []model.SearchParam,
		err error,
	)

	ValidateGetCountData(
		ctx *context.ContextModel,
		dto *in.GetListRequest,
		service services.ServicesWithGetListData,
	) (
		output []model.SearchParam,
		err error,
	)
}
