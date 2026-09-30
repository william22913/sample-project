package authorization_dto

import (
	"github.com/nexsoft-git/nexcommon/dto/out"
	"github.com/nexsoft-git/nexcommon/http/client"
)

type AuthorizationRoleMappingResponse struct {
	client.HTTPClientDTO

	Header  out.Header `json:"header"`
	Payload struct{}   `json:"payload"`
}

type AuthorizationRoleMappingRequest struct {
	UserID   int64  `json:"user_id" required:"auth" empty:"allowed"`
	ClientID string `json:"client_id" required:"auth"`
	Scope    string `json:"scope" required:"auth"`
	Locale   string `json:"locale" required:"auth"`
	Token    string `json:"token" required:"auth"`
	Path     string `json:"path" required:"auth" empty:"allowed"`
}
