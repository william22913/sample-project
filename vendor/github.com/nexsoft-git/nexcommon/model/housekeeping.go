package model

import "database/sql"

type HousekeepingModel struct {
	ID            sql.NullInt64
	Name          sql.NullString
	Type          sql.NullString
	Total         sql.NullInt64
	JobID         sql.NullString
	CreatedBy     sql.NullInt64
	CreatedAt     sql.NullTime
	CreatedClient sql.NullString
	UpdatedBy     sql.NullInt64
	UpdatedAt     sql.NullTime
	UpdatedClient sql.NullString
	Deleted       sql.NullBool
}
