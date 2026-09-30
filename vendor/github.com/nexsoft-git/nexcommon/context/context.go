package context

import (
	"context"
	"time"

	"github.com/nexsoft-git/nexcommon/constanta"
	"github.com/nexsoft-git/nexcommon/token/model"
	"github.com/nexsoft-git/nexlogger"
)

func (c *ContextModel) ToContext() context.Context {
	return context.WithValue(
		context.Background(),
		constanta.ApplicationContextConstanta,
		c,
	)
}

func NewContextModel() *ContextModel {
	ctx := new(ContextModel)
	ctx.ClientAccess.Logger = nexlogger.NewloggerModel()
	return ctx
}

type ContextModel struct {
	AuthAccessTokenModel model.AuthAccessTokenModel
	Limitation           Limitation
	Server               Server
	ClientAccess         ClientAccess
}

type ClientAccess struct {
	IdempotencyKey    string
	Timestamp         time.Time
	ClientAccount     int64
	ClientAccountName string
	Logger            nexlogger.LoggerModel
	Headers           map[string]string
	Path              string
}

type Server struct {
	IsSignatureCheck bool
	IsSendSignature  bool
	IsInternal       bool
}

type Limitation struct {
	DBSchema        string
	ParsedDBNode    string
	PermissionHave  string
	UserID          int64
	ServiceUserID   int64
	DataScope       map[string]interface{}
	Other           map[string]interface{}
	AdditionalParse interface{}
}
