package token

import (
	"context"
	"net/http"
	"reflect"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	xerrors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/token/authorization_dto"
	"github.com/nexsoft-git/nexcommon/token/model"
)

type userJWTValidator struct {
	nexsoftJWT
	additionalUserValue interface{}
}

func (u *userJWTValidator) SetAdditionalRedisValueParser(
	additional interface{},
) {
	u.additionalUserValue = additional
}

func (u *userJWTValidator) GetAdditionalRedisValueParser() interface{} {
	if u.additionalUserValue == nil {
		return nil
	}

	valueType := reflect.TypeOf(u.additionalUserValue)

	if valueType.Kind() == reflect.Ptr {
		valueType = valueType.Elem()
	}

	newValue := reflect.New(valueType)

	return newValue.Interface()
}

func (u userJWTValidator) ValidateJWTToken(
	ctx context.Context,
	jwtTokenStr string,
	scope string,
	resoureDisabled bool,
) (
	err error,
) {
	if jwtTokenStr == "" {
		err = xerrors.ErrUnauthorized
		return
	} else {
		var resourceID = ""
		if !resoureDisabled {
			resourceID = u.config.ResourceID
		}

		var payload model.PayloadJWTToken
		payload, err = u.ValidateTokenWithoutCheckSignature(
			jwtTokenStr,
			resourceID,
			scope,
		)

		if err != nil {
			return
		}

		if scope == "" {
			scope = payload.Scope
		}

		_ctx, valid := ctx.Value(constanta.ApplicationContextConstanta).(*internalCtx.ContextModel)
		if !valid {
			_ctx = internalCtx.NewContextModel()
		}

		auth, _ := strconv.Atoi(payload.Subject)
		_ctx.AuthAccessTokenModel.ClientID = payload.ClientID
		_ctx.AuthAccessTokenModel.AuthenticationServerUserID = int64(auth)
		_ctx.AuthAccessTokenModel.Scope = scope
		_ctx.AuthAccessTokenModel.Locale = payload.Locale

		ctx = context.WithValue(ctx, constanta.ApplicationContextConstanta, _ctx)

		return u.checkTokenInRedis(
			ctx,
			_ctx.ClientAccess.Path,
			jwtTokenStr,
			scope,
			payload,
		)
	}
}

func (u userJWTValidator) IsTokenUnidentified(
	ctx context.Context,
	token string,
) (
	bool,
	error,
) {

	redisResult, err := u.client.Get(
		token,
	).Result()

	if err != nil && err != redis.Nil {
		return false, err
	}

	if redisResult == constanta.INVALID_TOKEN_REDIS_VALUE {
		return false, xerrors.ErrUnauthorized
	}

	return redisResult == "", nil
}

func (u userJWTValidator) checkTokenInRedis(
	ctx context.Context,
	path string,
	token string,
	scope string,
	jwtPayload model.PayloadJWTToken,
) (
	err error,
) {

	notChecked, err := u.IsTokenUnidentified(ctx, token)

	if err != nil {
		return
	}

	if notChecked {
		err = u.roleMappingAuthorization(
			ctx,
			path,
			token,
			jwtPayload,
		)

		if err != nil {
			return
		}

	}

	return nil

}

func (u userJWTValidator) roleMappingAuthorization(
	ctx context.Context,
	path string,
	token string,
	jwtPayload model.PayloadJWTToken,
) error {

	var authorizationResponse authorization_dto.AuthorizationRoleMappingResponse
	header := make(map[string]string)
	header[constanta.AuthorizationHeaderConstanta] = u.config.FixedToken

	var userID int
	userID, err := strconv.Atoi(jwtPayload.Subject)

	if err != nil {
		return xerrors.ErrUnauthorized
	}

	code, err := u.apiConnector.HitAPI(
		ctx,
		u.config.AuthorizationAPI.RoleMapping.Method,
		u.config.AuthorizationAPI.Host,
		u.config.AuthorizationAPI.RoleMapping.Uri,
		header,
		authorization_dto.AuthorizationRoleMappingRequest{
			UserID:   int64(userID),
			ClientID: jwtPayload.ClientID,
			Scope:    jwtPayload.Scope,
			Locale:   jwtPayload.Locale,
			Token:    token,
			Path:     path,
		},
		&authorizationResponse,
	)

	if code != http.StatusOK {
		return xerrors.ErrUnauthorized
	}

	return err
}
