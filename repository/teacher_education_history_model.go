package repository

import "database/sql"

// TeacherEducationHistoryModel mirrors one teacher_education_histories row.
//
// There is no Deleted field: education rows hard-delete (spec criterion 27),
// and the table deliberately has no `deleted` column - which is also why it
// must never be registered with nexcommon's audit_helper, whose before-snapshot
// query filters `deleted = false` by literal name.
type TeacherEducationHistoryModel struct {
	ID                 sql.NullInt64
	UUIDKey            sql.NullString
	TeacherID          sql.NullInt64
	InstitutionLevelID sql.NullInt64
	// InstitutionLevel is the institution_levels.name the row points at. It is
	// not a column on this table - the read query joins it in so callers get
	// the code (e.g. BACHELOR) rather than the surrogate id, and so the detail
	// view needs no second round-trip per education row.
	InstitutionLevel sql.NullString
	Institution      sql.NullString
	StudyStartDate   sql.NullTime
	StudyEndDate     sql.NullTime
	// Score is a string, not a float64, because the column is NUMERIC(5,2) and
	// architecture C25 requires the Go side to hold a decimal representation.
	// A float64 cannot distinguish "87.50" from "87.5", and - the reason this
	// matters - it cannot decide whether a literal like 87.125 has more than
	// two decimal places without a rounding heuristic. Read as text, the value
	// Postgres stored is the value that comes back, and the "reject more than
	// 2 decimals, do not round" rule (criterion 40) is decided on the digits
	// themselves.
	Score          sql.NullString
	FocusedSubject sql.NullString
	CreatedAt      sql.NullTime
	UpdatedAt      sql.NullTime
}
