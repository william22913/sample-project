package http

import (
	"context"
	"net/http"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/controller"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/dto/out"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services"
)

type HTTPServerAccess func(
	ctx context.Context,
	r *http.Request,
) error

type FunctionServe func(
	*internalCtx.ContextModel,
	model.URLParam,
	interface{},
) (
	map[string]string,
	interface{},
	error,
)

type FunctionServeListData func(
	*internalCtx.ContextModel,
	[]model.SearchParam,
	in.GetListRequest,
) (
	map[string]string,
	interface{},
	error,
)

type additionalFunc func(*internalCtx.ContextModel)

type HTTPControllerParam struct {
	Ctx context.Context
	R   *http.Request
}

func NewWarpServiceParam(
	service services.Services,
	serve FunctionServe,
	cv controller.ServerAccessValidator,
) *wrapServiceParam {
	return &wrapServiceParam{
		cv:       cv,
		service:  service,
		serve:    serve,
		readBody: true,
	}
}

func NewWarpGetListServiceParam(
	service services.ServicesWithGetListData,
	serve FunctionServeListData,
	cv controller.ServerAccessValidator,
) *wrapGetListServiceParam {
	return &wrapGetListServiceParam{
		cv:           cv,
		service:      service,
		serve:        serve,
		defaultOrder: "id",
	}
}

type wrapServiceParam struct {
	serve                  FunctionServe
	service                services.Services
	cv                     controller.ServerAccessValidator
	readBody               bool
	menu                   string
	pathParams             []string
	headers                []string
	apiScope               string
	permission             string
	isCheckSignature       bool
	isSendSignature        bool
	checkClientID          bool
	dataScope              []string
	functionValidator      string
	multipart              bool
	isRecordActivityActive bool
	recordActivity         recordActivity
	description            string
	tags                   []string
	operationID            string
	dtoParam               *dtoParam
	output                 interface{}
	isIdempotency          bool
	isURLEncoded           bool
	signatureValidator     func(ctx *internalCtx.ContextModel, request *http.Request, body string) bool
}

type dtoParam struct {
	output      interface{}
	description string
	fileResult  bool
}

func NewDTOParam(output interface{}) *dtoParam {
	return &dtoParam{
		output:      output,
		description: "dto out",
	}
}

func (d *dtoParam) Description(description string) *dtoParam {
	d.description = description
	return d
}

func (r *wrapServiceParam) DTOOut(d *dtoParam) *wrapServiceParam {
	r.dtoParam = d

	switch r.dtoParam.output.(type) {
	case model.FileURLPath:
		r.dtoParam.fileResult = true
		r.output = d.output
	default:
		r.output = dtoConverter.GetFunction()(internalCtx.NewContextModel(), d.output)
	}

	return r
}

func (r *wrapGetListServiceParam) DTOOut(d *dtoParam) *wrapGetListServiceParam {
	r.dtoParam = d

	r.output = &out.DefaultResponse{
		Payload: d.output,
	}

	return r
}

type wrapGetListServiceParam struct {
	serve                    FunctionServeListData
	service                  services.ServicesWithGetListData
	cv                       controller.ServerAccessValidator
	pathParams               []string
	headers                  []string
	apiScope                 string
	permission               string
	isCheckSignature         bool
	isSendSignature          bool
	checkClientID            bool
	dataScope                []string
	defaultOrder             string
	isRecordActivityActive   bool
	recordActivity           recordActivity
	description              string
	tags                     []string
	operationID              string
	output                   interface{}
	dtoParam                 *dtoParam
	isIdempotency            bool
	signatureValidator       func(ctx *internalCtx.ContextModel, request *http.Request, body string) bool
	isEnableNextTokenChecker bool
}

type recordActivity struct {
	activity         map[string]interface{}
	menuCode         string
	isMenuAdmin      bool
	externalActivity bool
	localActivity    bool
	saveRequest      bool
	body             string
	header           map[string]string
}

func (r *recordActivity) AddActivity(key string, value interface{}) *recordActivity {
	r.activity[key] = value
	return r
}

func (r *recordActivity) MenuAdmin() *recordActivity {
	r.isMenuAdmin = true
	return r
}

func (r *recordActivity) SaveRequest() *recordActivity {
	r.saveRequest = true
	return r
}

func (r *recordActivity) ExternalActivityWithDisabledLocalActivity() *recordActivity {
	r.externalActivity = true
	r.localActivity = false
	return r
}

func (r *recordActivity) ExternalActivityWithEnabledLocalActivity() *recordActivity {
	r.externalActivity = true
	r.localActivity = true
	return r
}

func NewRecordActivity(menuCode string) *recordActivity {
	return &recordActivity{
		activity:      make(map[string]interface{}),
		menuCode:      menuCode,
		localActivity: true,
	}
}

func (w *wrapServiceParam) DataScope(dataScope ...string) *wrapServiceParam {
	w.dataScope = dataScope
	return w
}

func (w *wrapServiceParam) URLEncoded() *wrapServiceParam {
	w.isURLEncoded = true
	return w
}

func (w *wrapServiceParam) Multipart() *wrapServiceParam {
	w.multipart = true
	return w
}

func (w *wrapServiceParam) EnableServiceIdempotency() *wrapServiceParam {
	w.isIdempotency = true
	return w
}

func (w *wrapServiceParam) NotReadBody() *wrapServiceParam {
	w.readBody = false
	return w
}

func (w *wrapServiceParam) Menu(menu string) *wrapServiceParam {
	w.menu = menu
	return w
}

func (w *wrapServiceParam) PathParams(paths ...string) *wrapServiceParam {
	w.pathParams = paths
	return w
}

func (w *wrapServiceParam) Headers(headers ...string) *wrapServiceParam {
	w.headers = headers
	return w
}

func (w *wrapServiceParam) SendSignature() *wrapServiceParam {
	w.isSendSignature = true
	return w
}

func (w *wrapServiceParam) RecordActivity(activity *recordActivity) *wrapServiceParam {
	w.isRecordActivityActive = true
	w.recordActivity = *activity
	return w
}

func (w *wrapServiceParam) APIScope(scope string) *wrapServiceParam {
	w.apiScope = scope
	return w
}

func (w *wrapServiceParam) Permisson(permission string) *wrapServiceParam {
	w.permission = permission
	return w
}

func (w *wrapServiceParam) FunctionValidator(functionValidator string) *wrapServiceParam {
	w.functionValidator = functionValidator
	return w
}

func (w *wrapServiceParam) CheckSignature() *wrapServiceParam {
	w.isCheckSignature = true
	return w
}

func (w *wrapServiceParam) CheckClientID() *wrapServiceParam {
	w.checkClientID = true
	return w
}

func (w *wrapGetListServiceParam) DataScope(dataScope ...string) *wrapGetListServiceParam {
	w.dataScope = dataScope
	return w
}

func (w *wrapGetListServiceParam) PathParams(paths ...string) *wrapGetListServiceParam {
	w.pathParams = paths
	return w
}

func (w *wrapGetListServiceParam) Headers(headers ...string) *wrapGetListServiceParam {
	w.headers = headers
	return w
}

func (w *wrapGetListServiceParam) SendSignature() *wrapGetListServiceParam {
	w.isSendSignature = true
	return w
}

func (w *wrapGetListServiceParam) APIScope(scope string) *wrapGetListServiceParam {
	w.apiScope = scope
	return w
}

func (w *wrapGetListServiceParam) Permisson(permission string) *wrapGetListServiceParam {
	w.permission = permission
	return w
}

func (w *wrapGetListServiceParam) CheckSignature() *wrapGetListServiceParam {
	w.isCheckSignature = true
	return w
}

func (w *wrapGetListServiceParam) CheckClientID() *wrapGetListServiceParam {
	w.checkClientID = true
	return w
}

func (w *wrapGetListServiceParam) DefaultDataOrdering(order string) *wrapGetListServiceParam {
	w.defaultOrder = order
	return w
}

func (w *wrapGetListServiceParam) Menu(menu string) *wrapGetListServiceParam {
	return w
}

func (w *wrapGetListServiceParam) RecordActivity(activity *recordActivity) *wrapGetListServiceParam {
	w.isRecordActivityActive = true
	w.recordActivity = *activity
	return w
}

func (w *wrapGetListServiceParam) Description(param string) *wrapGetListServiceParam {
	w.description = param
	return w
}

func (w *wrapGetListServiceParam) Tags(param ...string) *wrapGetListServiceParam {
	w.tags = param
	return w
}

func (w *wrapGetListServiceParam) OperationID(param string) *wrapGetListServiceParam {
	w.operationID = param
	return w
}

func (w *wrapGetListServiceParam) EnableServiceIdempotency() *wrapGetListServiceParam {
	w.isIdempotency = true
	return w
}

func (w *wrapServiceParam) Description(param string) *wrapServiceParam {
	w.description = param
	return w
}

func (w *wrapServiceParam) Tags(param ...string) *wrapServiceParam {
	w.tags = param
	return w
}

func (w *wrapServiceParam) OperationID(param string) *wrapServiceParam {
	w.operationID = param
	return w
}

type defaultNexsoftDTO struct {
	version string
}

func (d *defaultNexsoftDTO) Version(param string) {
	d.version = param
}

func (d defaultNexsoftDTO) GetFunction() func(ctx *internalCtx.ContextModel, payload interface{}) interface{} {
	return func(ctx *internalCtx.ContextModel, payload interface{}) interface{} {
		result := out.DefaultResponse{
			DefaultMessage: out.DefaultMessage{
				Success: true,
			},
		}

		result.DefaultMessage.Header = out.Header{
			Timestamp: time.Now().Format(constanta.DefaultDtoOutTimeFormat),
			Version:   d.version,
		}

		if ctx != nil {
			result.DefaultMessage.Header.RequestID = ctx.ClientAccess.Logger.RequestID
		}

		result.Payload = payload
		payload = result

		return result
	}
}

func (w *wrapServiceParam) SignatureChecker(f func(ctx *internalCtx.ContextModel, request *http.Request, body string) bool) *wrapServiceParam {
	w.signatureValidator = f
	return w
}

func (w *wrapGetListServiceParam) SignatureChecker(f func(ctx *internalCtx.ContextModel, request *http.Request, body string) bool) *wrapGetListServiceParam {
	w.signatureValidator = f
	return w
}

func (w *wrapGetListServiceParam) EnableNextTokenChecker() *wrapGetListServiceParam {
	w.isEnableNextTokenChecker = true
	return w
}
