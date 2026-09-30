package controller

import (
	"context"

	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	errors "github.com/nexsoft-git/nexcommon/error"
)

func (cv ControllerValidator) ExternalAccessValidator() ServerAccessValidator {

	return ServerAccessValidator{
		Name: EXTERNAL_TOKEN_VALIDATOR,
		Func: func(ctx context.Context, header map[string]string) error {
			result := header[constanta.NexAuthResultKey]
			scope := header[constanta.NexAuthHeaderScopeConstanta]

			if result != constanta.NexternalResultValueExternal {
				return errors.ErrUnauthorized
			}

			accountKey := header[constanta.NexternalAccountIDKey]
			token := header[constanta.AuthorizationHeaderConstanta]

			return cv.validateUserExternalToken(
				ctx,
				token,
				accountKey,
				scope,
			)
		},
	}
}

func (cv ControllerValidator) validateUserExternalToken(
	ctx context.Context,
	token string,
	accountKey string,
	scope string,
) error {

	err := cv.externalTokenValidator.ValidateJWTToken(ctx, token, scope, accountKey)

	if err != nil {
		return err
	}

	_ctx, valid := ctx.Value(constanta.ApplicationContextConstanta).(*internalCtx.ContextModel)
	if !valid {
		_ctx = internalCtx.NewContextModel()
	}

	ctx = context.WithValue(ctx, constanta.ApplicationContextConstanta, _ctx)

	return nil
}