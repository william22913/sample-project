package repository

import "database/sql"

// InstitutionLevelModel mirrors one institution_levels row. The table is a
// seeded lookup with no CRUD surface (architecture A4) - it is only ever read,
// to resolve the code a request carries into the id the education table's
// foreign key needs.
type InstitutionLevelModel struct {
	ID        sql.NullInt64
	UUIDKey   sql.NullString
	Name      sql.NullString
	CreatedAt sql.NullTime
	UpdatedAt sql.NullTime
}
