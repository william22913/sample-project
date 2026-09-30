package http

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	doc "github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/mvrilo/go-redoc"
	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/controller"
	"github.com/nexsoft-git/nexcommon/docs/swagger"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/server"
	"github.com/nexsoft-git/nexcommon/util/activity"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/get_list_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/multipart_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"
	"github.com/nexsoft-git/nexlogger/log"
	"gopkg.in/yaml.v2"
)

func NewHTTPController(
	fixedToken string,
) *HTTPController {
	validator := &HTTPController{
		ControllerValidator: controller.NewControllerValidator(fixedToken),
		version:             "1.0.0",
		skipEndpointsLog:    []string{"/metrics"},
		swaggerGenerator:    swagger.SwaggerGenerator{},
	}

	dtoConverter = &defaultNexsoftDTO{}

	validator.swaggerGenerator.DiscoverKnownAuthorization()

	knownHeader = map[string]string{
		constanta.NexAuthResultKey:                      "",
		constanta.NexAuthHeaderScopeConstanta:           "",
		constanta.NexAuthHeaderCheckClientConstanta:     "",
		constanta.AuthorizationHeaderConstanta:          "",
		constanta.RefreshTokenHeaderConstanta:           "",
		constanta.RequestIDConstanta:                    "",
		constanta.IPAddressConstanta:                    "",
		constanta.SourceConstanta:                       "",
		constanta.DefaultTokenKeyConstanta:              "",
		constanta.TimestampSignatureHeaderNameConstanta: "",
		constanta.SignatureHeaderNameConstanta:          "",
		constanta.DeviceHeaderConstanta:                 "",
		constanta.ResourceHeaderConstanta:               "",
		constanta.RedirectURINameConstanta:              "",
		constanta.RedirectMethodNameConstanta:           "",
		constanta.NexternalAccountIDKey:                 "",
		constanta.ClientRequestTimestamp:                "",
	}

	return validator
}

func (g *HTTPController) AddKnownHeader(param ...string) {
	for i := 0; i < len(param); i++ {
		knownHeader[param[i]] = ""
	}
}

func (g *HTTPController) Router(router *mux.Router) *HTTPController {
	g.router = router
	return g
}

func (g *HTTPController) Formator(formator errors.Formator) *HTTPController {
	g.formator = formator
	return g
}

func (g *HTTPController) TagValidator(validator tag_validator.TagValidator) *HTTPController {
	g.validator = validator
	g.swaggerGenerator.TagValidator = validator

	return g
}

func (g *HTTPController) BasicValidator(validator basic_validator.BasicValidator) *HTTPController {
	g.basicValidator = validator
	g.swaggerGenerator.BasicValidator = validator

	return g
}

func (g *HTTPController) ListDataValidator(validator get_list_validator.GetListValidator) *HTTPController {
	g.listDataValidator = validator
	return g
}

func (g *HTTPController) MultipartValidator(validator multipart_validator.MultipartValidator) *HTTPController {
	g.multipartValidator = validator
	return g
}

func (g *HTTPController) MultipartReader(reader server.MultipartReader) *HTTPController {
	g.reader = reader
	return g
}

func (g *HTTPController) Version(version string) *HTTPController {
	g.version = version
	dtoConverter.Version(version)
	return g
}

func (g *HTTPController) LogEndpointsHit(unlogs ...string) *HTTPController {
	g.logEveryHit = true
	g.skipEndpointsLog = append(g.skipEndpointsLog, unlogs...)
	return g
}

func (g *HTTPController) AdditionalOtherFunction(f additionalFunc) *HTTPController {
	g.additionalFunc = f
	return g
}

func (g *HTTPController) ActivityUtil(util activity.ActivityUtil) *HTTPController {
	g.activityUtil = util
	return g
}

func (g *HTTPController) GenerateSwaggerDocs(
	path string,
	docsTitle string,
	replaceOldFile bool,
) {

	if docsTitle == "" {
		docsTitle = "Auto Generated Resouce API Documentation"
	}

	timeGenerated := fmt.Sprintf("\nDocs Generated At : %s", time.Now().Format(constanta.DefaultDtoOutTimeFormat))
	swagger.SwaggerContainer.Info.Description = timeGenerated

	yamlData, err := yaml.Marshal(swagger.SwaggerContainer)

	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found When Parsing Swagger")

		return
	}

	err = os.MkdirAll("./docs/swagger", 0770)

	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found When create docs repository")

		return
	}

	fileName := "./docs/swagger/swagger.yaml"
	_, err = os.Stat(fileName)

	if err == nil && !replaceOldFile {
		log.
			Info().
			Msg("System Not Generate Swagger File.")
	} else {
		err = ioutil.WriteFile(fileName, yamlData, 0644)
		if err != nil {
			log.Error().
				Err(err).
				Caller().
				Msg("Error Found When Generate Swagger yaml")

			return
		}
	}

	g.router.Handle("/swagger/swagger.yaml", http.FileServer(http.Dir("./docs/")))
	var handler http.Handler

	if g.swaggerView.name == "REDOC" {
		redoc := redoc.Redoc{
			Title:       swagger.SwaggerContainer.Info.Title,
			Description: swagger.SwaggerContainer.Info.Description,
			SpecFile:    "./docs/swagger/swagger.yaml",
			SpecPath:    "/swagger/swagger.yaml",
			DocsPath:    path,
		}
		handler = redoc.Handler()
	} else {
		opts := doc.SwaggerUIOpts{
			Path:    path,
			SpecURL: "/swagger/swagger.yaml",
			Title:   docsTitle,
		}
		handler = doc.SwaggerUI(opts, nil)
	}

	if handler != nil {
		g.router.Handle(path, handler)
	}
}

var dtoConverter DTOResponseConverter

type DTOResponseConverter interface {
	GetFunction() func(ctx *internalCtx.ContextModel, payload interface{}) interface{}
	Version(version string)
}

type swaggers struct {
	name string
}

var RedocSwagger = swaggers{name: "REDOC"}
var DefaultSwagger = swaggers{name: "DEFAULT"}

func (g *HTTPController) ExternalActivityUtil(util activity.ActivityUtil) *HTTPController {
	g.externalAcivityUtil = util
	return g
}

func (g *HTTPController) SetSwaggerView(param swaggers) *HTTPController {
	g.swaggerView = param
	return g
}

type HTTPController struct {
	controller.ControllerValidator

	swaggerGenerator    swagger.SwaggerGenerator
	router              *mux.Router
	formator            errors.Formator
	version             string
	reader              server.MultipartReader
	basicValidator      basic_validator.BasicValidator
	validator           tag_validator.TagValidator
	listDataValidator   get_list_validator.GetListValidator
	multipartValidator  multipart_validator.MultipartValidator
	logEveryHit         bool
	skipEndpointsLog    []string
	additionalFunc      additionalFunc
	activityUtil        activity.ActivityUtil
	externalAcivityUtil activity.ActivityUtil
	swaggerView         swaggers
}

func ModifyDTOResponse(
	param DTOResponseConverter,
) {
	dtoConverter = param
}

func (g HTTPController) HandleFunc(
	param *handleFuncParam,
) {
	g.router.HandleFunc(param.path, param.f).Methods(param.method...)
}

var knownHeader map[string]string

func (g HTTPController) Converter(
	r *http.Request,
) (
	context.Context,
	map[string]string,
) {
	result := make(map[string]string)

	for keys := range knownHeader {
		value := r.Header.Get(keys)
		if value != "" {
			result[keys] = value
		}
	}

	if r.Context() != nil {
		r = r.WithContext(context.Background())
	}

	return r.Context(), result
}
