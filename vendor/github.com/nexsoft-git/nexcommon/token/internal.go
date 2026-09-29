package token

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/jwt-go"
	"github.com/nexsoft-git/nexcommon/constanta"
	xerrors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/token/authorization_dto"
	"github.com/nexsoft-git/nexcommon/token/model"
	util "github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"
)

type internalJWTValidator struct {
	sync.Mutex
	nexsoftJWT

	roleMappingInternal RoleMapping
}

func (i *internalJWTValidator) SetRoleMappingInternal(
	roleMappingInternal RoleMapping,
) {
	i.roleMappingInternal = roleMappingInternal
}

func (i *internalJWTValidator) ValidateInternalToken(
	ctx context.Context,
	isCheckClientID bool,
	jwtTokenStr string,
) (
	err error,
) {

	var payload model.PayloadJWTInternal

	if jwtTokenStr == "" {
		err = xerrors.ErrUnauthorized
		return
	} else {

		payload, err = i.ValidateJWTInternal(
			jwtTokenStr,
			i.config.InternalKey,
		)

		if err != nil {
			return
		}

		if isCheckClientID {

			err = i.checkInternalTokenInRedis(
				ctx,
				jwtTokenStr,
				payload,
			)

			if err != nil {
				return
			}

		}

		if !i.basicValidator.CheckIsResourceIDExist(
			payload.Resource,
			i.config.ResourceID,
		) {
			err = xerrors.ErrJWTForbiddenByResource
			return
		}

		if payload.UserClient == "" {
			return xerrors.ErrUnauthorized
		}

	}

	return
}

func (i *internalJWTValidator) checkInternalTokenInRedis(
	ctx context.Context,
	token string,
	payload model.PayloadJWTInternal,
) (
	err error,
) {

	redisResult, err := i.client.Get(
		token,
	).Result()

	if err != nil && err != redis.Nil {

		log.Error().
			Err(err).
			Msg("Error Found get token data from redis")

		return
	}

	if redisResult == "" {

		err = i.checkInternalTokenToAuthorization(
			ctx,
			token,
			payload,
		)

		if err != nil {
			return
		}

	}

	return
}

func (i *internalJWTValidator) checkInternalTokenToAuthorization(
	ctx context.Context,
	token string,
	payload model.PayloadJWTInternal,
) (
	err error,
) {

	var userID int

	userID, err = strconv.Atoi(payload.Subject)

	if err != nil {
		return xerrors.ErrUnauthorized
	}

	if i.apiConnector != nil {
		var code int
		var response authorization_dto.AuthorizationRoleMappingResponse

		header := make(map[string]string)
		header[constanta.AuthorizationHeaderConstanta] = i.config.FixedToken

		code, err = i.apiConnector.HitAPI(
			ctx,
			i.config.AuthorizationAPI.RoleMappingInternal.Method,
			i.config.AuthorizationAPI.Host,
			i.config.AuthorizationAPI.RoleMappingInternal.Uri,
			header,
			authorization_dto.AuthorizationRoleMappingRequest{
				UserID:   int64(userID),
				ClientID: payload.ClientID,
				Locale:   payload.Locale,
				Token:    token,
			},
			&response,
		)

		if err != nil {

			log.Error().
				Err(err).
				Msg("Error Found when hit api authorization service")

			return
		}

		if code != http.StatusOK {

			err = xerrors.NewUnBundledErrorMessages(
				code,
				errors.New(response.Unsuccessfull.Payload.Code),
				nil,
			).Reason(
				response.Unsuccessfull.Payload.Message,
			)

			return
		}

	} else if i.roleMappingInternal != nil {
		var redisModel model.RedisAuthAccessTokenModel
		redisModel, err = i.roleMappingInternal(ctx, token, payload)

		if err != nil {

			log.Error().
				Err(err).
				Msg("Error Found when do role mapping internal")

			return
		}

		err = i.client.Set(token, util.StructToJSON(redisModel), 2*time.Hour).Err()
		if err != nil {
			log.Error().
				Err(err).
				Caller().
				Msg("Error to set token to redis")
			return
		}

	}

	return nil
}

type tokenDestination struct {
	token         string
	generate_time time.Time
}

var jwtToken = make(map[string]tokenDestination)

func (i *internalJWTValidator) GenerateInternalToken(
	resourceDestination string,
	userID int64,
	clientID string,
	issuer string,
	locale string,
) string {
	i.Lock()
	defer i.Unlock()

	return i.generateInternalToken(
		resourceDestination,
		userID,
		clientID,
		issuer,
		locale,
		jwt.SigningMethodHS512,
	)
}

func (i *internalJWTValidator) GenerateInternalTokenGrochat(
	resourceDestination string,
	userID int64,
	clientID string,
	issuer string,
	locale string,
) string {
	return i.generateInternalToken(
		resourceDestination,
		userID,
		clientID,
		issuer,
		locale,
		jwt.SigningMethodHS256,
	)
}

func (i *internalJWTValidator) generateInternalToken(
	resourceDestination string,
	userID int64,
	clientID string,
	issuer string,
	locale string,
	method *jwt.SigningMethodHMAC,
) string {

	userClientID := i.config.ClientID
	if clientID != "" {
		userClientID = clientID
	}

	usedUserID := i.config.UserID
	if userID > 0 {
		usedUserID = userID
	}

	tokenCode := model.PayloadJWTInternal{
		Locale:     locale,
		ClientID:   i.config.ClientID,
		Scope:      "read write",
		UserClient: userClientID,
		Resource:   resourceDestination,
		Version:    i.config.Version,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(24 * time.Hour).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    issuer,
			Subject:   strconv.Itoa(int(usedUserID)),
		},
	}

	var tokenDestinations = jwtToken[resourceDestination]

	if tokenDestinations.token == "" {
		tokenUsed, _ := model.JWTToken{}.GenerateToken(
			tokenCode,
			i.config.InternalKey,
			method,
		)

		tokenDestinations = tokenDestination{
			token:         tokenUsed,
			generate_time: time.Now(),
		}

		jwtToken[resourceDestination] = tokenDestinations
	} else {
		if time.Now().Sub(tokenDestinations.generate_time) > 2*time.Hour {

			tokenUsed, _ := model.JWTToken{}.GenerateToken(
				tokenCode,
				i.config.InternalKey,
				method,
			)

			tokenDestinations = tokenDestination{
				token:         tokenUsed,
				generate_time: time.Now(),
			}

			jwtToken[resourceDestination] = tokenDestinations
		}
	}

	return tokenDestinations.token
}
