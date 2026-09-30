package refresh_token_dto

import (
	"github.com/nexsoft-git/nexcommon/http/client"
)

type RefreshTokenResponse struct {
	client.HTTPClientDTO

	Code   int `json:"code"`
	Status int `json:"status"`
	Data   struct {
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	} `json:"data"`
}
