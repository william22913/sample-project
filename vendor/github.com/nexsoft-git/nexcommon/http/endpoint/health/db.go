package health

import (
	"context"
	"database/sql"
	"time"
)

type dbConn struct {
	db *sql.DB
}

func (d *dbConn) Ping() string {

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := d.db.PingContext(ctx); err != nil {
		return "DOWN"
	}

	return "UP"
}

func NewDBConnChecker(
	db *sql.DB,
) *dbConn {
	return &dbConn{
		db: db,
	}
}
