package token

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-redis/redis/v7"
	"github.com/nexsoft-git/nexcommon/constanta"
	xerrors "github.com/nexsoft-git/nexcommon/error"
	"github.com/nexsoft-git/nexcommon/token/model"
	"github.com/nexsoft-git/nexcommon/token/refresh_token_dto"
	util "github.com/nexsoft-git/nexcommon/util/text"
	"github.com/nexsoft-git/nexlogger/log"
)

func (u userJWTValidator) LogoutToken(
	ctx context.Context,
	jwtToken string,
) (
	err error,
) {

	isChecked, err := u.IsTokenUnidentified(ctx, jwtToken)
	if err != nil {
		return
	}

	if isChecked {
		return xerrors.ErrUnauthorized
	}

	ttl, err := u.client.TTL(jwtToken).Result()

	if err != nil {
		return
	}

	_, err = u.client.Set(
		jwtToken,
		constanta.INVALID_TOKEN_REDIS_VALUE,
		ttl,
	).Result()

	// _, err = u.client.Del(
	// 	jwtToken,
	// ).Result()

	if err != nil {
		return
	}

	return
}

// func (u userJWTValidator) logoutTokenGrochat(
// 	ctx context.Context,
// 	token string,
// ) error {

// 	var bodyObject authentication_dto.CheckTokenAuthenticationResponse

// 	header := make(map[string]string)
// 	header[constanta.AuthorizationHeaderConstanta] = token

// 	code, err := u.apiConnector.HitAPI(
// 		ctx,
// 		u.config.GrochatAPI.CheckToken.Method,
// 		u.config.GrochatAPI.Host,
// 		u.config.GrochatAPI.CheckToken.Uri,
// 		header,
// 		"",
// 		&bodyObject,
// 	)

// 	if code != http.StatusOK {
// 		return xerrors.ErrUnauthorized
// 	}

// 	return err
// }

func (u userJWTValidator) RefreshToken(
	ctx context.Context,
	oldToken string,
	refreshToken string,
) (
	newToken string,
	_refreshToken string,
	err error,
) {

	redisResult, err := u.client.Get(
		oldToken,
	).Result()

	if err != nil && err != redis.Nil {
		err = xerrors.ErrUnauthorized
		return
	} else {

		var statusCode int
		var response refresh_token_dto.RefreshTokenResponse

		statusCode, response, err = u.refreshTokenOnGrochat(
			ctx,
			oldToken,
			refreshToken,
		)

		if err != nil {
			return
		}

		if statusCode != http.StatusOK {

			err = xerrors.NewUnBundledErrorMessages(
				statusCode,
				errors.New(response.Unsuccessfull.Payload.Code),
				nil,
			).Reason(
				response.Unsuccessfull.Payload.Message,
			)

			return
		} else {

			newToken = response.Data.Token
			_refreshToken = response.Data.RefreshToken
			// TODO HIT API INSERT USER TOKEN

			go u.refreshTokenOnRedis(
				ctx,
				redisResult,
				newToken,
				oldToken,
			)
		}
	}
	return
}

func (u userJWTValidator) refreshTokenOnRedis(
	ctx context.Context,
	redisResult string,
	newToken string,
	oldToken string,
) {

	var accessTokenModel model.AuthAccessTokenModel
	_ = json.Unmarshal([]byte(redisResult), &accessTokenModel)

	payload, _ := u.ValidateTokenWithoutCheckSignature(newToken, u.config.ResourceID, "")

	var expiration time.Duration
	if u.config.Duration == 0 {
		expiration = time.Until(time.Unix(payload.ExpiresAt, 0))
	} else {
		expiration = time.Duration(u.config.Duration)
	}

	_, err := u.client.Set(newToken, util.StructToJSON(accessTokenModel), -1).Result()

	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found When Set New Refresh Token To Redis")

		return
	}

	_, err = u.client.Expire(newToken, expiration).Result()

	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found When Set New Refresh Token Expiration")

		return
	}

	_, err = u.client.Del(oldToken).Result()

	if err != nil {
		log.Error().
			Err(err).
			Caller().
			Msg("Error Found When Deleting Old Token")

		return
	}

}

func (u userJWTValidator) refreshTokenOnGrochat(
	ctx context.Context,
	oldToken string,
	refreshToken string,
) (
	statusCode int,
	result refresh_token_dto.RefreshTokenResponse,
	err error,
) {

	header := make(map[string]string)
	header[constanta.AuthorizationHeaderConstanta] = oldToken
	header[constanta.RefreshTokenHeaderConstanta] = refreshToken

	// refreshTokenBody := model.RefreshTokenBody{
	// 	GrantType:    "refresh_token",
	// 	RefreshToken: refreshToken,
	// }

	statusCode, err = u.apiConnector.HitAPI(
		ctx,
		u.config.GrochatAPI.RefreshToken.Method,
		u.config.GrochatAPI.Host,
		u.config.GrochatAPI.RefreshToken.Uri,
		header,
		nil,
		&result,
	)

	return
}
