package idempotency

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/nexsoft-git/nexcommon/model"
)

func NewPGIdempotencyDAO(
	db *sql.DB,
	tableName string,
) PgIdempotencyDAO {
	return PgIdempotencyDAO{
		db:        db,
		tableName: tableName,
	}
}

type PgIdempotencyDAO struct {
	db        *sql.DB
	tableName string
}

func (p PgIdempotencyDAO) CheckIdempotency(
	key string,
) (
	tx *sql.Tx,
	idempotency model.IdempotencyModel,
	err error,
) {

	err = p.InsertIdempotencyKey(key)

	if err != nil {
		return
	}

	tx, err = p.db.Begin()

	if err != nil {
		return
	}

	idempotency, err = p.GetIdempotencyKey(tx, key)

	return

}

func (p PgIdempotencyDAO) GetIdempotencyKey(
	tx *sql.Tx,
	key string,
) (
	idempotency model.IdempotencyModel,
	err error,
) {

	query := `
	SELECT 
		id, result, created_at
	FROM
		` + p.tableName + ` 
	WHERE 
		idempotency_key = $1
	FOR UPDATE 
`

	if err = tx.QueryRow(query, key).Scan(
		&idempotency.Id,
		&idempotency.ResultStr,
		&idempotency.CreatedAt,
	); err != nil {
		return
	}

	idempotency.IdempotencyKey.String = key
	idempotency.IdempotencyKey.Valid = true

	if idempotency.ResultStr.String != "" {
		_ = json.Unmarshal([]byte(idempotency.ResultStr.String), &idempotency.Result)
	}

	return
}

func (p PgIdempotencyDAO) InsertIdempotencyKey(
	idempotencyKey string,
) error {

	query := `INSERT INTO ` + p.tableName + `(idempotency_key) VALUES ($1) ON CONFLICT DO NOTHING`
	_, err := p.db.Query(query, idempotencyKey)

	return err
}

func (p PgIdempotencyDAO) SetIdempotencyResult(
	idempotencyModel model.IdempotencyModel,
) error {

	byteData, _ := json.Marshal(idempotencyModel.Result)

	query := `UPDATE ` + p.tableName + ` set result = $1, updated_at = NOW() WHERE idempotency_key = $2`
	_, err := p.db.Query(query, byteData, idempotencyModel.IdempotencyKey.String)

	return err
}

func (p PgIdempotencyDAO) DeleteExpiredIdempotencyKey(
	duration time.Duration,
) error {

	timeThreshold := time.Now().Add(-duration).UTC()

	query := `DELETE FROM ` + p.tableName + ` WHERE created_at < $1`
	_, err := p.db.Exec(query, timeThreshold)

	return err
}
