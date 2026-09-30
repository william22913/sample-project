package controller

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/nexcommon/idempotency"
	"github.com/nexsoft-git/nexcommon/token"
)

type apiAuthorization struct {
	AccessValidator ServerAccessValidator
	ApiScope        string
	Permission      string
}

type Converter func(ctx context.Context, header struct{}) error

var cv ControllerValidator
var cv_init bool

func NewControllerValidator(
	fixedToken string,
) ControllerValidator {
	if !cv_init {
		cv = ControllerValidator{
			serverAccess: make(map[string]apiAuthorization),
			fixedToken:   fixedToken,
		}
		cv_init = true
	}

	return cv
}

func (cv *ControllerValidator) Redis(redis *redis.Client) *ControllerValidator {
	cv.redis = redis
	return cv
}

func (cv *ControllerValidator) TokenValidator(tokenValidator token.UserJWTValidator) *ControllerValidator {
	cv.tokenValidator = tokenValidator
	return cv
}

func (cv *ControllerValidator) InternalTokenValidator(internalTokenValidator token.InternalJWTValidator) *ControllerValidator {
	cv.internalTokenValidator = internalTokenValidator
	return cv
}

func (cv *ControllerValidator) ExternalTokenValidator(externalTokenValidator token.ExternalJWTValidator) *ControllerValidator {
	cv.externalTokenValidator = externalTokenValidator
	return cv
}

func (cv *ControllerValidator) EnableIdempotencyProcessing(idempotency idempotency.IdempotencyProcessing) *ControllerValidator {
	cv.idempotencyProcessing = idempotency
	return cv
}

func (cv *ControllerValidator) GetIdempotencyProcessing() idempotency.IdempotencyProcessing {
	return cv.idempotencyProcessing
}

type ControllerValidator struct {
	serverAccess           map[string]apiAuthorization
	fixedToken             string
	redis                  *redis.Client
	tokenValidator         token.UserJWTValidator
	internalTokenValidator token.InternalJWTValidator
	externalTokenValidator token.ExternalJWTValidator
	idempotencyProcessing  idempotency.IdempotencyProcessing
}

func (cv ControllerValidator) AddControllerValidator(
	_package,
	apiScope,
	permission string,
	access ServerAccessValidator,
	method ...string,
) {
	for key := range method {
		cv.serverAccess[cv.keys(_package, method[key])] = apiAuthorization{
			AccessValidator: access,
			ApiScope:        apiScope,
			Permission:      permission,
		}
	}
}

func (cv ControllerValidator) GetControllerValidator(
	_package,
	method string,
) (
	apiAuthorization,
	bool,
) {
	f, ok := cv.serverAccess[cv.keys(_package, method)]
	return f, ok
}

func (cv ControllerValidator) keys(
	_package,
	method string,
) string {
	return fmt.Sprintf("%s/%s", _package, method)
}
