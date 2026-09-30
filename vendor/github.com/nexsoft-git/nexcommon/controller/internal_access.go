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

func (cv ControllerValidator) InternalAccessValidator() ServerAccessValidator {
	return ServerAccessValidator{
		Name: INTERNAL_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			return cv.checkInternalTokenOnRedis(ctx, header)
		},
	}
}

func (cv ControllerValidator) InternalAccessValidatorWithKong() ServerAccessValidator {
	return ServerAccessValidator{
		Name: INTERNAL_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			result := header[constanta.NexAuthResultKey]

			if result != constanta.NexAuthInternalTokenResult {
				return errors.ErrUnauthorized
			}

			return cv.checkInternalTokenOnRedis(
				ctx,
				header,
			)
		},
	}
}

func (cv ControllerValidator) checkInternalTokenOnRedis(
	ctx context.Context,
	header map[string]string,
) error {
	token := header[constanta.AuthorizationHeaderConstanta]
	isCheckClientID := header[constanta.NexAuthHeaderCheckClientConstanta] == "true"

	err := cv.internalTokenValidator.ValidateInternalToken(
		ctx,
		isCheckClientID,
		token,
	)

	if err != nil {
		return err
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

	return cv.setAuthDataToContextInternal(ctx, redisResult)

}

func (cv ControllerValidator) setAuthDataToContextInternal(
	ctx context.Context,
	result string,
) error {

	var redisModel model.RedisAuthAccessTokenModel
	var authModel model.AuthenticationModel

	_ = json.Unmarshal([]byte(result), &redisModel)
	_ = json.Unmarshal([]byte(redisModel.Authentication), &authModel)

	_ctx, valid := ctx.Value(constanta.ApplicationContextConstanta).(*internalCtx.ContextModel)
	if !valid {
		_ctx = internalCtx.NewContextModel()
	}

	auth, _ := strconv.Atoi(authModel.Oauth.UserID)
	_ctx.AuthAccessTokenModel.AuthenticationServerUserID = int64(auth)
	_ctx.AuthAccessTokenModel.ResourceUserID = redisModel.ResourceUserID
	_ctx.AuthAccessTokenModel.Locale = redisModel.Locale
	_ctx.AuthAccessTokenModel.SignatureKey = redisModel.SignatureKey
	_ctx.AuthAccessTokenModel.ClientAlias = redisModel.ClientAlias
	_ctx.AuthAccessTokenModel.AliasName = redisModel.AliasName
	_ctx.AuthAccessTokenModel.ClientID = authModel.Oauth.ClientID
	_ctx.AuthAccessTokenModel.DBName = redisModel.DBName
	_ctx.AuthAccessTokenModel.Schema = redisModel.Schema

	_ctx.Limitation.UserID = int64(auth)
	_ctx.Limitation.ServiceUserID = redisModel.ResourceUserID

	ctx = context.WithValue(ctx, constanta.ApplicationContextConstanta, _ctx)

	return nil
}
