package model

import "database/sql"

type UserExternalModel struct {
	ID            sql.NullInt64
	UserID        sql.NullInt64
	UserAlias     sql.NullString
	ClientID      sql.NullString
	ClientSecret  sql.NullString
	IPWhitelist   sql.NullString
	ClientAlias   sql.NullString
	SignatureKey  sql.NullString
	Locale        sql.NullString
	Deleted       sql.NullBool
	AccountID     sql.NullInt64
	AccountName   sql.NullString
	AccountSchema sql.NullString
	DBName        sql.NullString
}
