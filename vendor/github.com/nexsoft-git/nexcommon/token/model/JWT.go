package model

import (
	"github.com/nexsoft-git/jwt-go"
)

type JWTToken struct {
}

func (input JWTToken) generateJWT(
	Payload jwt.Claims,
	key string,
	method *jwt.SigningMethodHMAC,
) (
	string,
	error,
) {
	jwtToken := jwt.NewWithClaims(method, Payload)
	token, err := jwtToken.SignedString([]byte(key))
	if err != nil {
		return "", err
	}

	return token, nil
}

func (input JWTToken) GenerateToken(
	Payload jwt.Claims,
	key string,
	method *jwt.SigningMethodHMAC,
) (
	string,
	error,
) {
	return input.generateJWT(
		Payload,
		key,
		method,
	)
}
