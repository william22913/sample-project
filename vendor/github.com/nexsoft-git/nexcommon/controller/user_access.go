package controller

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	errors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/token/model"
)

func (cv ControllerValidator) UserAccessValidatorDisablingResourceChecker() ServerAccessValidator {
	return ServerAccessValidator{
		Name: USER_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			if header != nil {
				header[constanta.RESOURCE_CHECKER_DISABLED] = "true"
			}

			return cv.getTokenOnRedis(ctx, header)
		},
	}
}

func (cv ControllerValidator) UserAccessValidatorDisablingResourceCheckerWithKong() ServerAccessValidator {
	return ServerAccessValidator{
		Name: USER_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			result := header[constanta.NexAuthResultKey]

			if result != constanta.NexAuthUserTokenResult {
				return errors.ErrUnauthorized
			}

			if header != nil {
				header[constanta.RESOURCE_CHECKER_DISABLED] = "true"
			}

			return cv.getTokenOnRedis(ctx, header)
		},
	}
}

func (cv ControllerValidator) UserAccessValidator() ServerAccessValidator {
	return ServerAccessValidator{
		Name: USER_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			return cv.getTokenOnRedis(ctx, header)

		},
	}
}

func (cv ControllerValidator) UserAccessValidatorWithKong() ServerAccessValidator {
	return ServerAccessValidator{
		Name: USER_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			result := header[constanta.NexAuthResultKey]

			if result != constanta.NexAuthUserTokenResult {
				return errors.ErrUnauthorized
			}

			return cv.getTokenOnRedis(ctx, header)
		},
	}
}

func (cv ControllerValidator) getTokenOnRedis(
	ctx context.Context,
	header map[string]string,
) error {

	token := header[constanta.AuthorizationHeaderConstanta]
	scope := header[constanta.NexAuthHeaderScopeConstanta]
	var resoureDisabled = false

	if val, _ := header[constanta.RESOURCE_CHECKER_DISABLED]; val == "true" {
		resoureDisabled = true
	}

	isNotChecked, err := cv.tokenValidator.IsTokenUnidentified(
		ctx,
		token,
	)

	if err != nil {
		return err
	}

	if isNotChecked {
		err = cv.tokenValidator.ValidateJWTToken(
			ctx,
			token,
			scope,
			resoureDisabled,
		)

		if err != nil {
			return err
		}

	}

	redisResult, err := cv.redis.Get(
		token,
	).Result()

	if err != nil {
		if err == redis.Nil {
			return errors.ErrUnauthorized
		}

		return err
	}

	return cv.setAuthDataToContext(ctx, redisResult)
}

func (cv ControllerValidator) setAuthDataToContext(
	ctx context.Context,
	result string,
) error {

	var tokenModel model.AuthAccessTokenModel
	var authModel model.AuthenticationModel

	err := json.Unmarshal([]byte(result), &tokenModel)

	if err != nil {
		return errors.ErrUnauthorized
	}

	_ctx, valid := ctx.Value(constanta.ApplicationContextConstanta).(*internalCtx.ContextModel)
	if !valid {
		_ctx = internalCtx.NewContextModel()
	}

	_ctx.AuthAccessTokenModel = tokenModel
	err = json.Unmarshal([]byte(tokenModel.Authentication), &authModel)

	if err != nil {
		return errors.ErrUnauthorized
	}

	additionalRedisParser := cv.tokenValidator.GetAdditionalRedisValueParser()

	if additionalRedisParser != nil {
		otherValue := tokenModel.Other
		if otherValue == nil {
			otherValue = make(map[string]interface{})
		}

		err := json.Unmarshal([]byte(result), &additionalRedisParser)
		if err != nil {
			return errors.ErrUnauthorized
		}

		otherValue[constanta.DEFAULT_REDIS_VALUE_KEY] = additionalRedisParser
		tokenModel.Other = otherValue
	}

	_ctx.AuthAccessTokenModel.Role = authModel.Role
	_ctx.AuthAccessTokenModel.DataGroup = authModel.Data
	auth, _ := strconv.Atoi(authModel.Oauth.UserID)
	_ctx.AuthAccessTokenModel.Locale = tokenModel.Locale
	_ctx.AuthAccessTokenModel.AuthenticationServerUserID = int64(auth)
	_ctx.AuthAccessTokenModel.ClientID = authModel.Oauth.ClientID
	_ctx.AuthAccessTokenModel.Scope = authModel.Oauth.Scope
	_ctx.AuthAccessTokenModel.DBName = tokenModel.DBName
	_ctx.AuthAccessTokenModel.Schema = tokenModel.Schema

	_ctx.Limitation.Other = tokenModel.Other

	ctx = context.WithValue(ctx, constanta.ApplicationContextConstanta, _ctx)

	return nil
}
