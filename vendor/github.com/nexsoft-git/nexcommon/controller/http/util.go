package http

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/model"
	srv "github.com/nexsoft-git/nexcommon/server"
	token_model "github.com/nexsoft-git/nexcommon/token/model"
	txt "github.com/nexsoft-git/nexcommon/util/text"
)

func (g HTTPController) preStart(
	r *http.Request,
	param *wrapServiceParam,
) error {
	r.Header.Add(constanta.NexAuthHeaderScopeConstanta, param.apiScope)
	r.Header.Add(constanta.NexAuthHeaderCheckClientConstanta, fmt.Sprintf("%v", param.checkClientID))

	if param.multipart {
		return g.reader.ParseMultipartForm(r)
	}

	return nil
}

func (g HTTPController) getContext(
	r *http.Request,
	checkSignature,
	sendSignature bool,
) *internalCtx.ContextModel {

	var ctx *internalCtx.ContextModel

	rCtx := r.Context().Value(constanta.ApplicationContextConstanta)
	if rCtx == nil {

		ctx = internalCtx.NewContextModel()
		context := context.WithValue(
			r.Context(),
			constanta.ApplicationContextConstanta,
			ctx,
		)

		r = r.WithContext(context)
	} else {
		ctx = rCtx.(*internalCtx.ContextModel)
	}

	requestID := srv.ReadHeader(r, constanta.RequestIDConstanta)
	if requestID == "" {
		requestID = txt.GetUUID()
		r.Header.Set(constanta.RequestIDConstanta, requestID)
	}

	ctx.ClientAccess.Logger.RequestID = requestID
	ctx.ClientAccess.Logger.IP = srv.ReadHeader(r, constanta.IPAddressConstanta)
	ctx.ClientAccess.Logger.Source = srv.ReadHeader(r, constanta.SourceConstanta)
	ctx.ClientAccess.Logger.Source = srv.ReadHeader(r, constanta.SourceConstanta)
	ctx.ClientAccess.Logger.AccessToken = srv.ReadHeader(r, constanta.AuthorizationHeaderConstanta)

	ctx.Server.IsSendSignature = sendSignature
	ctx.Server.IsSignatureCheck = checkSignature

	return ctx

}

func (g HTTPController) getURLParam(
	request *http.Request,
	path []string,
	header []string,
) (
	model.URLParam,
	map[string]string,
) {
	param := model.URLParam{}

	param.Query = g.readQueryParam(request)
	param.Path = g.readPathParam(request, path)

	return param, g.readHeaders(request, header)
}

func (g HTTPController) readBody(
	ctx *internalCtx.ContextModel,
	param *wrapServiceParam,
	request *http.Request,
) (
	dto interface{},
	err error,
) {
	var stringBody string

	if request.Method != "GET" {
		if !param.readBody {
			return
		}

		dto = param.service.GetDTO()

		stringBody, ctx.ClientAccess.Logger.ByteIn, err = srv.ReadBody(request)
		if err != nil {
			return nil, errors.ErrReadBody
		}

		if param.recordActivity.saveRequest {
			param.recordActivity.body = stringBody
			param.recordActivity.header = ctx.ClientAccess.Headers
		}

		err = json.Unmarshal([]byte(stringBody), dto)
		if err != nil {
			return nil, errors.ErrMarshalingBody
		}

		err = g.validator.ValidateByTag(ctx, dto, param.functionValidator, param.menu)
		if err != nil {
			return nil, err
		}
	}

	if ctx.Server.IsSignatureCheck {
		if param.signatureValidator == nil {
			param.signatureValidator = g.defaultSignatureValidator()
		}

		if !param.signatureValidator(ctx, request, stringBody) {
			return nil, errors.ErrInvalidSignature.ContextModel(ctx)
		}

		ctx.Server.IsSendSignature = true
	}

	return
}

func (g HTTPController) readQueryParam(
	request *http.Request,
) map[string]string {
	return g.generateQueryParam(request)
}

func (g HTTPController) readPathParam(
	request *http.Request,
	params []string,
) map[string]string {

	path := make(map[string]string)

	for i := 0; i < len(params); i++ {
		path[params[i]] = mux.Vars(request)[params[i]]
	}

	return path
}

func (g HTTPController) defaultSignatureValidator() func(
	ctx *internalCtx.ContextModel,
	request *http.Request,
	body string,
) bool {
	return func(
		ctx *internalCtx.ContextModel,
		request *http.Request,
		body string,
	) bool {
		digest := txt.GenerateMessageDigest(body)
		return g.validateSignature(
			digest,
			ctx.AuthAccessTokenModel.SignatureKey,
			request,
		)
	}
}

func (g HTTPController) readHeaders(
	request *http.Request,
	headers []string,
) map[string]string {

	header := make(map[string]string)

	for i := 0; i < len(headers); i++ {
		header[headers[i]] = request.Header.Get(headers[i])
	}

	return header
}

func (g HTTPController) generateQueryParam(request *http.Request) map[string]string {
	result := make(map[string]string)
	defer func() {
		_ = recover()
	}()

	var errs error
	rawQuery := request.URL.RawQuery
	rawSplit := strings.Split(rawQuery, "&")

	for key := range rawSplit {
		splitEqual := strings.Split(rawSplit[key], "=")
		result[splitEqual[0]], errs = url.QueryUnescape(splitEqual[1])
		if errs != nil {
			result[splitEqual[0]] = splitEqual[1]
		}
	}

	return result
}

func (g HTTPController) validateSignature(
	messageDigest string,
	key string,
	request *http.Request,
) bool {

	signature := srv.ReadHeader(request, constanta.SignatureHeaderNameConstanta)
	timestamp := srv.ReadHeader(request, constanta.TimestampSignatureHeaderNameConstanta)
	internalToken := srv.ReadHeader(request, constanta.AuthorizationHeaderConstanta)

	if signature == "" || timestamp == "" || internalToken == "" {
		return false
	}

	return txt.ValidateSignature(
		request.Method,
		request.RequestURI,
		internalToken,
		messageDigest,
		timestamp,
		key,
		signature,
	)
}

func (g HTTPController) PermissionAndScopeChecker(
	ctx *internalCtx.ContextModel,
	permission string,
	apiScope string,
	dataScope []string,
) error {

	ctx.ClientAccess.Logger.ClientID = ctx.AuthAccessTokenModel.ClientID
	ctx.ClientAccess.Logger.UserID = fmt.Sprintf("%d", ctx.AuthAccessTokenModel.ResourceUserID)

	var tokenModel token_model.AuthenticationModel
	var err error

	_ = json.Unmarshal([]byte(ctx.AuthAccessTokenModel.Authentication), &tokenModel)

	if permission != "" || apiScope != "" {

		if permission != "" {
			ctx.Limitation.PermissionHave, err = g.ValidatePermissionWithRole(permission, tokenModel.Role)

			if err != nil {
				return err
			}

			if strings.Contains(ctx.Limitation.PermissionHave, "own") {
				ctx.Limitation.UserID = ctx.AuthAccessTokenModel.ResourceUserID
			}
		}

		if apiScope != "" {
			scopeSplit := strings.Split(apiScope, " ")
			for i := 0; i < len(scopeSplit); i++ {
				if !g.basicValidator.CheckIsScopeExist(ctx.AuthAccessTokenModel.Scope, apiScope) {
					return errors.ErrJWTForbiddenByScope
				}
			}
		}

		if dataScope != nil {
			ctx.Limitation.DataScope = g.CheckScope(tokenModel.Data.Scope, dataScope)
		}
	}

	if g.additionalFunc != nil {
		g.additionalFunc(ctx)
	}

	return nil

}

func (g HTTPController) ValidatePermissionWithRole(
	mustHavePermission string,
	roleModel token_model.AuthenticationRoleModel,
) (
	string,
	error,
) {
	var isValid = false
	var permissionAllowed string

	validationResult, _ := g.basicValidator.IsNexsoftPermissionStandardValid(mustHavePermission)
	if !validationResult {
		return permissionAllowed, errors.ErrFormatField.Param("PERMISSION")
	}

	splitMustHavePermission := strings.Split(mustHavePermission, ":")
	menu := splitMustHavePermission[0]
	permission := splitMustHavePermission[1]

	if roleModel.Permission[permission] == nil && roleModel.Permission["all"] != nil {
		permission = "all"
	}

	isValid, permissionAllowed = g.roleChecker(permission, roleModel.Permission[permission])

	if !isValid {
		return permissionAllowed, errors.ErrJWTForbiddenByPermission
	}

	isValid = false
	splitDotMenu := strings.Split(menu, ".")
	size := len(splitDotMenu)

	for size > 0 {
		menu = ""
		for i := 0; i < size; i++ {
			menu += splitDotMenu[i]
			if i < size-1 {
				menu += "."
			}
		}

		if roleModel.Permission[menu] == nil && roleModel.Permission["all"] != nil {
			menu = "all"
		}

		if roleModel.Permission[menu] != nil {
			isValid, permissionAllowed = g.roleChecker(permission, roleModel.Permission[menu])
			if isValid {
				break
			}
		}
		size--
	}

	if !isValid {
		return permissionAllowed, errors.ErrJWTForbiddenByPermission
	}

	return permissionAllowed, nil
}

func (g HTTPController) roleChecker(
	permissionNeed string,
	listPermission []string,
) (
	bool,
	string,
) {
	for i := 0; i < len(listPermission); i++ {
		if listPermission[i] == permissionNeed {
			return true, listPermission[i]
		}
		if listPermission[i] == permissionNeed+"-own" {
			return true, listPermission[i]
		}
	}
	return false, ""
}

func (g HTTPController) CheckScope(
	listScope map[string]interface{},
	checkedScope []string,
) map[string]interface{} {
	result := make(map[string]interface{})
	numberOfAddedQuery := 0
	for key := range listScope {
		switch listScope[key].(type) {
		case []interface{}:
			for i := 0; i < len(checkedScope); i++ {
				scopeQuery, isExist := g.checkIsScopeContains(key, checkedScope[i])
				if isExist {
					numberOfAddedQuery++
					result = g.appendQuery(result, checkedScope[i], listScope[scopeQuery])
				}
			}
		case []map[string][]string:
			data := listScope[key].([]map[string][]string)
			isExist := g.isScopeExistInListHashmap(data, checkedScope)
			if isExist {
				numberOfAddedQuery++
				keySplit := strings.Split(key, ".")
				result = g.appendQuery(result, keySplit[len(keySplit)-1], listScope[key].([]map[string][]string))
			}
		}
	}
	if numberOfAddedQuery >= len(checkedScope) {
		return result
	} else {
		return nil
	}
}

func (g HTTPController) checkIsScopeContains(
	listScopeKey string,
	scope string,
) (
	string,
	bool,
) {
	splitMustHavePermission := strings.Split(scope, ":")
	menu := splitMustHavePermission[0]
	splitDotMenu := strings.Split(scope, ".")
	size := len(splitDotMenu)
	for size > 0 {
		menu = ""
		for i := 0; i < size; i++ {
			menu += splitDotMenu[i]
			if i < size-1 {
				menu += "."
			}
		}
		if menu == listScopeKey {
			return menu, true
		}
		size--
	}
	return "", false
}

func (g HTTPController) appendQuery(
	data map[string]interface{},
	field string,
	value interface{},
) map[string]interface{} {
	temp := data[field]
	switch vals := value.(type) {
	case []map[string][]string:
		data = g.appendListScope(data, field, vals)
	case []interface{}:
		if temp == nil {
			if len(data) == 0 {
				data[field] = value
			} else {
				isFound := false
				for key := range data {
					switch data[key].(type) {
					case []map[string][]string:
						temp := data[key].([]map[string][]string)
						for i := 0; i < len(temp); i++ {
							for keyOnHash := range temp[i] {
								if keyOnHash == field {
									isFound = true
									break
								}
							}
						}
					}
				}
				if !isFound {
					data[field] = value
				}
			}
		}
	}
	return data
}

func (g HTTPController) appendListScope(
	data map[string]interface{},
	field string,
	value []map[string][]string,
) map[string]interface{} {
	var fieldValue []map[string][]string

	if data[field] == nil {
		fieldValue = append(fieldValue, value...)
	} else {
		switch data[field].(type) {
		case []string:
			fieldValue = append(fieldValue, value...)
		case []map[string][]string:
			fieldValue = data[field].([]map[string][]string)
			fieldValue = append(fieldValue, value...)
		}
	}

	data[field] = fieldValue

	for i := 0; i < len(fieldValue); i++ {
		dataSlice := fieldValue[i]
		for key := range dataSlice {
			if data[key] != nil {
				delete(data, key)
			}
		}
	}
	return data
}

func (g HTTPController) isScopeExistInListHashmap(
	listScope []map[string][]string,
	listNeedScope []string,
) (
	isAllExist bool,
) {
	for i := 0; i < len(listScope); i++ {
		for key := range listScope[i] {
			isAllExist = false
			for j := 0; j < len(listNeedScope); j++ {
				_, isExist := g.checkIsScopeContains(key, listNeedScope[j])
				isAllExist = isExist || isAllExist
			}
			if !isAllExist {
				return false
			}
		}
	}
	return true
}
