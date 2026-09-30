package dao

import (
	"database/sql"
	"fmt"

	"github.com/nexsoft-git/nexcommon/model"
)

func NewExternalUserDAO(
	db *sql.DB,
	schema string,
	table string,
	getAccountQuery GetAccountQuery,
) *ExternalUserDAO {
	return &ExternalUserDAO{
		db:              db,
		schema:          schema,
		table:           table,
		getAccountQuery: getAccountQuery,
		isUseMultipleDB: false,
	}
}

func (e *ExternalUserDAO) SetUseMultipleDB() *ExternalUserDAO {
	e.isUseMultipleDB = true
	return e
}

type ExternalUserDAO struct {
	db              *sql.DB
	schema          string
	table           string
	getAccountQuery GetAccountQuery
	isUseMultipleDB bool
}

type GetAccountQuery func() string

func (e ExternalUserDAO) GetExternalClientWithAccountKey(
	clientID string,
	accountKey string,
) (
	result model.UserExternalModel,
	err error,
) {

	tableName := e.table
	if e.schema != "" {
		tableName = fmt.Sprintf("%s.%s", e.schema, e.table)
	}

	selectQuery := `
	SELECT 
		u.id, client_id, signature_key, 
		locale, client_alias, deleted, 
		account.id, account.name, account.schema`

	if e.isUseMultipleDB {
		selectQuery += `, account.db_name`
	}

	query := `
	` + selectQuery + `
	FROM
		` + tableName + ` u left join (` + e.getAccountQuery() + `) as account
		on u.id is not null
	WHERE 
		(auth_user_id = 0 or auth_user_id is null) and 
		client_id = $2   
	`

	row := e.db.QueryRow(query, []interface{}{accountKey, clientID}...)
	if e.isUseMultipleDB {
		err = row.Scan(
			&result.ID, &result.ClientID, &result.SignatureKey,
			&result.Locale, &result.ClientAlias, &result.Deleted,
			&result.AccountID, &result.AccountName, &result.AccountSchema,
			&result.DBName,
		)
		return result, err
	}

	err = row.Scan(
		&result.ID, &result.ClientID, &result.SignatureKey,
		&result.Locale, &result.ClientAlias, &result.Deleted,
		&result.AccountID, &result.AccountName, &result.AccountSchema,
	)

	return result, err

}

func (e ExternalUserDAO) InsertExternalClient(
	user model.UserExternalModel,
) (
	err error,
) {

	tableName := e.table
	if e.schema != "" {
		tableName = fmt.Sprintf("%s.%s", e.schema, e.table)
	}

	query := `
	INSERT INTO ` + tableName + `
		(
		client_id, client_alias, client_secret, 
		signature_key, ip_whitelist, locale 
		) VALUES (
			$1, $2, $3, 
			$4, $5, $6
		) `

	stmt, err := e.db.Prepare(query)
	if err != nil {
		return
	}

	_, err = stmt.Exec(
		user.ClientID.String, user.ClientAlias.String, user.ClientSecret.String,
		user.SignatureKey.String, user.IPWhitelist.String, user.Locale.String,
	)

	return err

}
