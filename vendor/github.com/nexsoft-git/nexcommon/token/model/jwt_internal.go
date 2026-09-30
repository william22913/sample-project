package model

import "github.com/nexsoft-git/jwt-go"

type PayloadJWTInternal struct {
	Locale     string `json:"locale"`
	ClientID   string `json:"cid"`
	Resource   string `json:"resource"`
	Scope      string `json:"scope"`
	Version    string `json:"version"`
	UserClient string `json:"user_client"`
	jwt.StandardClaims
}
