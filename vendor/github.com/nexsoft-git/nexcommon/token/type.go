package token

import (
	"context"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/nexcommon/dao"
	"github.com/nexsoft-git/nexcommon/http/client"
	"github.com/nexsoft-git/nexcommon/token/model"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
)

type RoleMapping func(
	ctx context.Context,
	token string,
	payload model.PayloadJWTInternal,
) (
	model.RedisAuthAccessTokenModel,
	error,
)

type JWTValidatorConfig struct {
	ResourceID       string
	UserKey          string
	InternalKey      string
	ClientID         string
	UserID           int64
	Version          string
	Duration         time.Duration
	FixedToken       string
	GrochatAPI       GrochatApi
	AuthorizationAPI AuthApi
}

type AuthApi struct {
	Host                string
	RoleMapping         Path
	RoleMappingInternal Path
}

type GrochatApi struct {
	Host               string
	CheckToken         Path
	CheckTokenInternal Path
	RefreshToken       Path
}

type Path struct {
	Uri    string
	Method string
}

type UserJWTValidator interface {
	ValidateJWTToken(
		ctx context.Context,
		jwtTokenStr string,
		scope string,
		resoureDisabled bool,
	) (
		err error,
	)

	IsTokenUnidentified(
		ctx context.Context,
		token string,
	) (
		bool,
		error,
	)

	RefreshToken(
		ctx context.Context,
		oldToken string,
		refreshToken string,
	) (
		newToken string,
		_refreshToken string,
		err error,
	)

	LogoutToken(
		ctx context.Context,
		jwtToken string,
	) (
		err error,
	)

	ValidateTokenWithoutCheckSignature(
		jwtTokenStr string,
		resourceID string,
		scope string,
	) (
		jwtToken model.PayloadJWTToken,
		err error,
	)

	SetAdditionalRedisValueParser(
		additional interface{},
	)

	GetAdditionalRedisValueParser() interface{}
}

type ExternalJWTValidator interface {
	ValidateJWTToken(
		ctx context.Context,
		jwtTokenStr string,
		scope string,
		accountKey string,
	) (
		err error,
	)

	SetCheckClientURL(
		string,
	)
}

type InternalJWTValidator interface {
	ValidateInternalToken(
		ctx context.Context,
		isCheckClientID bool,
		jwtTokenStr string,
	) (
		err error,
	)

	GenerateInternalToken(
		resourceDestination string,
		userID int64,
		clientID string,
		issuer string,
		locale string,
	) string

	GenerateInternalTokenGrochat(
		resourceDestination string,
		userID int64,
		clientID string,
		issuer string,
		locale string,
	) string

	SetRoleMappingInternal(
		roleMappingInternal RoleMapping,
	)
}

func NewExternalJWTValidator(
	externalDAO dao.ExternalUserDAO,
	apiConnector client.APIConnector,
	internalValidator InternalJWTValidator,
	authHost string,
	clientID string,
	authUserID int64,
	resourceID string,
	prefixAuthUrl string,
) ExternalJWTValidator {
	defaultURL := "/v1/internal/clients/%s"

	if prefixAuthUrl != "" {
		prefixAuthUrl = strings.Trim(prefixAuthUrl, "/")
		defaultURL = "/v1/" + prefixAuthUrl + "/internal/clients/%s"
	}

	return &externalUserJWTValidator{
		dao: externalDAO,
		authURL: authDestination{
			host:           authHost,
			clientCheckURL: defaultURL,
			clientID:       clientID,
			authUserID:     authUserID,
			resourceID:     resourceID,
		},
		apiConnector:      apiConnector,
		internalValidator: internalValidator,
	}
}

func NewJWTTokenValidator(
	config JWTValidatorConfig,
	basicValidator basic_validator.BasicValidator,
	apiConnector client.APIConnector,
	client *redis.Client,
) UserJWTValidator {
	return &userJWTValidator{
		nexsoftJWT: nexsoftJWT{
			config:         config,
			basicValidator: basicValidator,
			apiConnector:   apiConnector,
			client:         client,
		},
	}
}

func NewJWTTokenInternalValidator(
	config JWTValidatorConfig,
	basicValidator basic_validator.BasicValidator,
	apiConnector client.APIConnector,
	client *redis.Client,
) InternalJWTValidator {
	return &internalJWTValidator{
		nexsoftJWT: nexsoftJWT{
			config:         config,
			basicValidator: basicValidator,
			apiConnector:   apiConnector,
			client:         client,
		},
	}
}
