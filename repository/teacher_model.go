// Package repository holds one struct per database row shape. These are the
// only types that know a column exists - DAOs read into them and services
// translate them into DTOs. Mirrors kelasku's repository/ package.
package repository

import "database/sql"

// TeacherModel mirrors one teachers row (sql_migrations/01_init_schema.sql).
//
// Every field is a sql.Null* rather than a bare Go type so a NULL column stays
// distinguishable from a zero value. That matters immediately for phone (the
// one nullable column) and for created_by/updated_by, which are NULL on every
// row this phase - there is no authentication yet, so no actor to record.
type TeacherModel struct {
	ID          sql.NullInt64
	UUIDKey     sql.NullString
	TeacherCode sql.NullString
	FirstName   sql.NullString
	LastName    sql.NullString
	Email       sql.NullString
	Phone       sql.NullString
	HireDate    sql.NullTime
	Status      sql.NullString
	Deleted     sql.NullBool
	CreatedBy   sql.NullInt64
	UpdatedBy   sql.NullInt64
	CreatedAt   sql.NullTime
	UpdatedAt   sql.NullTime
}
