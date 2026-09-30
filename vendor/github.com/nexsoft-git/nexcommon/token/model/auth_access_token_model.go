package model

type AuthAccessTokenModel struct {
	RedisAuthAccessTokenModel
	ClientID                   string `json:"cid"`
	AuthenticationServerUserID int64  `json:"aid"`
	Scope                      string `json:"scp"`
	Role                       AuthenticationRoleModel
	DataGroup                  AuthenticationDataModel
}

func (input AuthAccessTokenModel) ConvertToRedisModel() RedisAuthAccessTokenModel {
	return RedisAuthAccessTokenModel{
		ResourceUserID: input.ResourceUserID,
		Authentication: input.Authentication,
		IPWhiteList:    input.IPWhiteList,
		SignatureKey:   input.SignatureKey,
		Locale:         input.Locale,
		GrochatAccount: input.GrochatAccount,
	}
}

type RedisAuthAccessTokenModel struct {
	ResourceUserID int64                  `json:"rid"`
	Authentication string                 `json:"auth"`
	IPWhiteList    string                 `json:"ipl"`
	SignatureKey   string                 `json:"sign"`
	Locale         string                 `json:"locale"`
	AliasName      string                 `json:"als"`
	ClientAlias    string                 `json:"cls"`
	GrochatAccount string                 `json:"gca"`
	Schema         string                 `json:"sch"`
	DBName         string                 `json:"dbn"`
	Other          map[string]interface{} `json:"oth"`
}
