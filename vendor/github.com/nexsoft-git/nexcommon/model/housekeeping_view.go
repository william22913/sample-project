package model

import "database/sql"

type GetListHousekeepingModel struct {
	ID            sql.NullInt64
	JobID         sql.NullString
	Group         sql.NullString
	Type          sql.NullString
	Name          sql.NullString
	Counter       sql.NullInt32
	Total         sql.NullInt32
	Status        sql.NullString
	CreatedBy     sql.NullInt64
	CreatedClient sql.NullString
	CreatedAt     sql.NullTime
	UpdatedAt     sql.NullTime
}
