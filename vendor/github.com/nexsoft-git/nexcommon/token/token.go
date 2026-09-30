package token

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/jwt-go"
	xerrors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/http/client"
	"github.com/nexsoft-git/nexcommon/token/model"
	util "github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexlogger/log"
)

type nexsoftJWT struct {
	config         JWTValidatorConfig
	basicValidator basic_validator.BasicValidator
	apiConnector   client.APIConnector
	client         *redis.Client
}

func (n nexsoftJWT) MandatoryValidateJWTToken(
	jwtTokenStr string,
	apiAllowedScope string,
	resourceID string,
) (
	jwtToken model.PayloadJWTToken,
	err error,
) {

	jwtToken, err = n.ValidateJWT(
		jwtTokenStr,
		n.config.UserKey,
	)

	if err != nil {
		return
	}

	if !n.basicValidator.CheckIsResourceIDExist(jwtToken.Resource, resourceID) {
		err = xerrors.ErrJWTForbiddenByResource
		return
	}

	return
}

func (n nexsoftJWT) ValidateTokenWithoutCheckSignature(
	jwtTokenStr string,
	resourceID string,
	scope string,
) (
	jwtToken model.PayloadJWTToken,
	err error,
) {
	jwtToken, err = n.ConvertJWTToPayload(jwtTokenStr)
	if err != nil {
		return
	}

	if time.Now().Unix() > jwtToken.ExpiresAt {
		err = xerrors.ErrExpiredToken
		return
	}

	if resourceID != "" {
		if !n.basicValidator.CheckIsResourceIDExist(jwtToken.Resource, resourceID) {
			err = xerrors.ErrJWTForbiddenByResource
			return
		}
	}

	if scope != "" {
		if !n.basicValidator.CheckIsScopeExist(jwtToken.Scope, scope) {
			err = xerrors.ErrJWTForbiddenByScope
			return
		}
	}

	return
}

func (n nexsoftJWT) ConvertJWTToPayload(
	jwtTokenStr string,
) (
	jwtToken model.PayloadJWTToken,
	err error,
) {
	splitJWT := strings.Split(jwtTokenStr, ".")
	if len(splitJWT) == 3 {
		payload := splitJWT[1]

		byteData, errs := util.Base64decoder(payload)
		if errs != nil {
			err = errs
			return
		}
		_ = json.Unmarshal(byteData, &jwtToken)
	} else {
		err = xerrors.ErrUnauthorized
	}

	return
}

func (n nexsoftJWT) ValidateJWT(
	jwtTokenStr string,
	key string,
) (
	payload model.PayloadJWTToken,
	errors error,
) {

	claims := &model.PayloadJWTToken{}
	jwtToken, err := jwt.ParseWithClaims(jwtTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if err != nil {
		if strings.Contains(err.Error(), "expired") {
			errors = xerrors.ErrExpiredToken
		} else {
			errors = xerrors.ErrUnauthorized
		}
		return
	}

	if jwtToken.Header["alg"] != "HS512" && jwtToken.Header["alg"] != "HS256" {
		return payload, xerrors.ErrJWTInvalidMethod
	}

	payload = *jwtToken.Claims.(*model.PayloadJWTToken)
	return
}

func (n nexsoftJWT) ValidateJWTInternal(
	jwtTokenStr string,
	key string,
) (
	payload model.PayloadJWTInternal,
	err error,
) {
	claims := &model.PayloadJWTInternal{}
	jwtToken, errors := jwt.ParseWithClaims(jwtTokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(key), nil
	})

	if errors != nil {
		if strings.Contains(errors.Error(), "expired") {
			err = xerrors.ErrExpiredToken
			return
		}

		log.Error().
			Err(err).
			Msg("Error Found when parsing JWT Token")

		err = xerrors.ErrUnauthorized
		return
	}

	if jwtToken.Header["alg"] != "HS512" && jwtToken.Header["alg"] != "HS256" {
		return payload, xerrors.ErrJWTInvalidMethod
	}

	payload = *jwtToken.Claims.(*model.PayloadJWTInternal)
	return
}
