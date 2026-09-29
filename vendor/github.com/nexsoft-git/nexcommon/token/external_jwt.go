package token

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	errors "errors"

	"github.com/nexsoft-git/nexcommon/constanta"
	internalCtx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dao"
	xerrors "github.com/nexsoft-git/nexcommon/error"

	"github.com/nexsoft-git/nexcommon/http/client"
	"github.com/nexsoft-git/nexcommon/model"
	tokenModel "github.com/nexsoft-git/nexcommon/token/model"
)

type hitAuthCheckClientResponse struct {
	client.HTTPClientDTO

	Nexsoft struct {
		Header struct {
			RequestID string `json:"request_id"`
			Version   string `json:"version"`
			Timestamp string `json:"timestamp"`
		} `json:"header"`
		Payload struct {
			Status struct {
				Success bool   `json:"success"`
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"status"`
			Data struct {
				Content struct {
					AliasName            string    `json:"alias_name"`
					ClientID             string    `json:"client_id"`
					ResourceIds          string    `json:"resource_ids"`
					ClientSecret         string    `json:"client_secret"`
					SignatureKey         string    `json:"signature_key"`
					Scope                string    `json:"scope"`
					GrantTypes           string    `json:"grant_types"`
					RedirectURI          string    `json:"redirect_uri"`
					IPWhitelist          string    `json:"ip_whitelist"`
					Authorities          string    `json:"authorities"`
					AccessTokenValidity  int64     `json:"access_token_validity"`
					RefreshTokenValidity int64     `json:"refresh_token_validity"`
					MultipleLogin        bool      `json:"multiple_login"`
					Locale               string    `json:"locale"`
					MaxAuthFail          int       `json:"max_auth_fail"`
					UpdatedAt            time.Time `json:"updated_at"`
					CreatedBy            int       `json:"created_by"`
				} `json:"content"`
			} `json:"data"`
		} `json:"payload"`
	} `json:"nexsoft"`
}

type authDestination struct {
	host           string
	clientCheckURL string
	clientID       string
	authUserID     int64
	resourceID     string
}

type externalUserJWTValidator struct {
	nexsoftJWT
	dao               dao.ExternalUserDAO
	authURL           authDestination
	apiConnector      client.APIConnector
	internalValidator InternalJWTValidator
}

func (e *externalUserJWTValidator) SetCheckClientURL(
	url string,
) {
	e.authURL.clientCheckURL = url
}

func (u externalUserJWTValidator) ValidateJWTToken(
	ctx context.Context,
	jwtTokenStr string,
	scope string,
	accountKey string,
) (
	err error,
) {
	if jwtTokenStr == "" {
		err = xerrors.ErrUnauthorized
		return
	} else {

		var payload tokenModel.PayloadJWTToken
		payload, err = u.ValidateTokenWithoutCheckSignature(
			jwtTokenStr,
			u.config.ResourceID,
			scope,
		)

		if err != nil {
			return
		}

		if scope == "" {
			scope = payload.Scope
		}

		clientData, err := u.checkUserOnDB(ctx, jwtTokenStr, accountKey, payload)

		if err != nil {
			return err
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
		_ctx.AuthAccessTokenModel.DBName = clientData.DBName.String
		_ctx.AuthAccessTokenModel.Schema = clientData.AccountSchema.String

		_ctx.Limitation.DBSchema = clientData.AccountSchema.String
		_ctx.Limitation.UserID = int64(auth)
		_ctx.Limitation.ServiceUserID = clientData.UserID.Int64
		_ctx.ClientAccess.ClientAccount = clientData.AccountID.Int64
		_ctx.ClientAccess.ClientAccountName = clientData.AccountName.String

		ctx = context.WithValue(ctx, constanta.ApplicationContextConstanta, _ctx)

		return nil
	}
}

func (u externalUserJWTValidator) checkUserOnDB(
	ctx context.Context,
	token string,
	accountKey string,
	payload tokenModel.PayloadJWTToken,
) (
	model.UserExternalModel,
	error,
) {
	result := model.UserExternalModel{}
	clientData, err := u.dao.GetExternalClientWithAccountKey(payload.ClientID, accountKey)

	if err == sql.ErrNoRows {
		err = nil
		header := make(map[string]string)
		header[constanta.AuthorizationHeaderConstanta] = u.internalValidator.GenerateInternalToken(
			"auth",
			u.authURL.authUserID,
			u.authURL.clientID,
			u.authURL.resourceID,
			payload.Locale,
		)

		apiResult := hitAuthCheckClientResponse{}
		status, err := u.apiConnector.HitAPI(
			ctx,
			http.MethodGet,
			u.authURL.host,
			fmt.Sprintf(u.authURL.clientCheckURL, payload.ClientID),
			header,
			nil,
			&apiResult,
		)

		if err != nil {
			return result, err
		}

		if status != 200 {
			return result, xerrors.NewUnBundledErrorMessages(
				status,
				errors.New(apiResult.Unsuccessfull.Payload.Code),
				nil,
			).Reason(apiResult.Unsuccessfull.Payload.Message)
		}

		user := model.UserExternalModel{
			ClientID:     sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.ClientID},
			ClientAlias:  sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.AliasName},
			ClientSecret: sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.ClientSecret},
			SignatureKey: sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.SignatureKey},
			IPWhitelist:  sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.IPWhitelist},
			Locale:       sql.NullString{String: apiResult.Nexsoft.Payload.Data.Content.Locale},
		}

		err = u.dao.InsertExternalClient(user)

		if err != nil {
			return result, err
		}

		clientData, err = u.dao.GetExternalClientWithAccountKey(payload.ClientID, accountKey)
	}

	if err != nil {
		return model.UserExternalModel{}, err
	}

	if !clientData.ID.Valid {
		return model.UserExternalModel{}, xerrors.ErrUnknownData.Param("USER")
	}

	if clientData.Deleted.Bool {
		return model.UserExternalModel{}, xerrors.ErrUserNonActive
	}

	clientData.UserID = clientData.ID

	return clientData, nil

}
