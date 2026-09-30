package token

import (
	"strconv"
	"time"

	"github.com/nexsoft-git/jwt-go"
	"github.com/nexsoft-git/nexcommon/token/model"
)

func NewLocalToken(
	resourceID string,
	tokenKey string,
	defaultClientID string,
	tokenDuration time.Duration,
) LocalToken {
	return &localToken{
		resourceID:      resourceID,
		tokenKey:        tokenKey,
		defaultClientID: defaultClientID,
		tokenDuration:   tokenDuration,
	}
}

type LocalToken interface {
	GenerateLocalToken(
		clientID string,
		userID int64,
		issuer string,
		locale string,
	) (
		result string,
	)
}

type localToken struct {
	resourceID      string
	tokenKey        string
	defaultClientID string
	tokenDuration   time.Duration
}

func (l localToken) GenerateLocalToken(
	clientID string,
	userID int64,
	issuer string,
	locale string,
) (
	result string,
) {
	usedUserID := userID
	timeNow := time.Now()
	expiredAt := timeNow.Add(l.tokenDuration)

	JWTTokenPayload := model.PayloadJWTToken{
		ClientID: func() string {
			if clientID == "" {
				return l.defaultClientID
			} else {
				return clientID
			}
		}(),
		Resource: l.resourceID,
		Scope:    "read write",
		Locale:   locale,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expiredAt.Unix(),
			Issuer:    issuer,
			IssuedAt:  timeNow.Unix(),
			Subject:   strconv.Itoa(int(usedUserID)),
		},
	}

	jwtToken, _ := generateToken(JWTTokenPayload, l.tokenKey)

	return jwtToken
}

func generateJWT(
	Payload jwt.Claims,
	key string,
) (
	token string,
	err error,
) {
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS512, Payload)
	token, err = jwtToken.SignedString([]byte(key))
	if err != nil {
		return
	}
	return
}

func generateToken(
	Payload jwt.Claims,
	key string,
) (
	string,
	error,
) {
	return generateJWT(Payload, key)
}
