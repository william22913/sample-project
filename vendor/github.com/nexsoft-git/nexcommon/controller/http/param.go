package http

import (
	"net/http"
	"strings"

	"github.com/nexsoft-git/nexcommon/docs/swagger"
)

func NewHandleFuncParam(
	path string,
	ws WrapperServiceResult,
	method ...string,
) *handleFuncParam {
	swagger.SwaggerContainer.SwaggerPath[path] = make(swagger.SwaggerControllerMethods)

	if strings.Contains(strings.ToLower(strings.Join(method, ", ")), "get") {
		ws.cs.RequestBody = nil
	}

	for i := 0; i < len(method); i++ {
		if method[i] != http.MethodOptions {
			swagger.SwaggerContainer.SwaggerPath[path][strings.ToLower(method[i])] = ws.cs
		}
	}

	return &handleFuncParam{
		path:   path,
		f:      ws.f,
		method: method,
	}
}

type handleFuncParam struct {
	path   string
	f      func(http.ResponseWriter, *http.Request)
	method []string
}