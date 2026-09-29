package model

import "github.com/nexsoft-git/jwt-go"

type PayloadJWTToken struct {
	ClientID string `json:"cid"`
	Resource string `json:"resource"`
	Scope    string `json:"scope"`
	Locale   string `json:"locale"`
	jwt.StandardClaims
}
