package dao_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"sample-project/dao"
	"sample-project/repository"
)

func educationRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "uuid_key", "teacher_id", "institution_level_id", "name",
		"institution", "study_start_date", "study_end_date", "score",
		"focused_subject", "created_at", "updated_at",
	})
}

// score is a []byte, not a float64, because that is genuinely what lib/pq
// hands over for a NUMERIC column - it arrives in text format. Feeding the
// test a float64 would both fail to scan into the string-backed Score field
// and test a path production never takes.
func educationRowValues(id int64, institution string, start, end time.Time, score string) []driver.Value {
	return []driver.Value{
		id, "b1e1c0de-0000-4000-8000-00000000000" + string(rune('0'+id%10)), 7, 5, "BACHELOR",
		institution, start, end, []byte(score), "Mathematics",
		time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC),
	}
}

// TestGetByTeacherIDOrdersNewestFirstOnQuery is criterion 26.
//
// The assertion is on the emitted SQL, not on the order sqlmock hands rows
// back - sqlmock returns rows in insertion order regardless of the ORDER BY,
// so a test that appeared to check the ordering that way would pass even with
// the clause deleted. The clause is the thing that has to be there.
func TestGetByTeacherIDOrdersNewestFirstOnQuery(t *testing.T) {
	src := sourceOf(t, "dao/education_dao.go")
	m := regexp.MustCompile(`ORDER BY teh\.study_end_date DESC`).FindString(src)
	if m == "" {
		t.Error("criterion 26: the education read lost its `ORDER BY study_end_date DESC`, " +
			"so rows come back in whatever order the planner chooses")
	}
}

func TestGetByTeacherIDScansJoinedLevelName(t *testing.T) {
	db, mock := newMock(t)
	end := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectQuery(`FROM teacher_education_histories teh`).
		WithArgs(int64(7)).
		WillReturnRows(educationRow().AddRow(educationRowValues(1, "Universitas Indonesia", start, end, "3.75")...))

	got, err := dao.NewEducationDAO(db).GetByTeacherID(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d rows, want 1", len(got))
	}
	// The caller must receive the code, never the surrogate id.
	if got[0].InstitutionLevel.String != "BACHELOR" {
		t.Errorf("institution_level: got %q, want the joined code BACHELOR", got[0].InstitutionLevel.String)
	}
	// Compared as text, not as a number: criterion 25 wants the digits
	// round-tripped exactly, and "3.75" is the literal the column returns.
	if got[0].Score.String != "3.75" {
		t.Errorf("score: got %q, want the exact literal %q", got[0].Score.String, "3.75")
	}
}

// TestGetByTeacherIDReturnsEmptySliceNotNull is criterion 13: the detail view
// renders with zero education rows as a deliberate empty state. A nil slice
// marshals to JSON null, which a client renders as an error rather than as an
// empty list - so the DAO must hand back a non-nil empty slice.
func TestGetByTeacherIDReturnsEmptySliceNotNull(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM teacher_education_histories teh`).
		WithArgs(int64(7)).
		WillReturnRows(educationRow()) // zero rows

	got, err := dao.NewEducationDAO(db).GetByTeacherID(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Error("criterion 13: got a nil slice, which serializes to JSON null - want an empty array")
	}
	if len(got) != 0 {
		t.Errorf("got %d rows, want 0", len(got))
	}
}

// educationWriteCols is the column list every education write returns - the
// same joined list a read produces, so a write response and a later read cannot
// disagree.
func educationWriteCols() []string {
	return []string{
		"id", "uuid_key", "teacher_id", "institution_level_id", "name",
		"institution", "study_start_date", "study_end_date", "score",
		"focused_subject", "created_at", "updated_at",
	}
}

// TestEducationWritesAreTeacherScoped is criteria 27 and 28: an edit or delete
// aimed at another teacher's row must change nothing, and must look identical
// to aiming at a row that does not exist.
//
// Both are addressed by uuid_key now, and a zero-row result is what a
// mismatched teacher produces. The same answer covers "no such row", which is
// the point - the caller cannot tell the two apart, so the route cannot be used
// to probe for keys.
func TestEducationWritesAreTeacherScoped(t *testing.T) {
	db, mock := newMock(t)
	end := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	start := time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC)

	const eduUUID = "d0de0000-0000-4000-8000-000000000011"

	// A row belonging to teacher 7, addressed by a caller working on teacher 8.
	// Both statements return nothing, which is what the CTE yields when its
	// WHERE matches no row.
	mock.ExpectBegin()
	mock.ExpectQuery(`WITH written AS \(UPDATE teacher_education_histories SET`).
		WithArgs(int64(5), "Universitas Indonesia", start, end, "3.75", "Mathematics", eduUUID, int64(8)).
		WillReturnRows(sqlmock.NewRows(educationWriteCols()))
	mock.ExpectQuery(`WITH written AS \(DELETE FROM teacher_education_histories`).
		WithArgs(eduUUID, int64(8)).
		WillReturnRows(sqlmock.NewRows(educationWriteCols()))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	edu := dao.NewEducationDAO(db)

	updated, err := edu.Update(context.Background(), tx, eduUUID, 8, repository.TeacherEducationHistoryModel{
		InstitutionLevelID: sql.NullInt64{Int64: 5, Valid: true},
		Institution:        sql.NullString{String: "Universitas Indonesia", Valid: true},
		StudyStartDate:     sql.NullTime{Time: start, Valid: true},
		StudyEndDate:       sql.NullTime{Time: end, Valid: true},
		Score:              sql.NullString{String: "3.75", Valid: true},
		FocusedSubject:     sql.NullString{String: "Mathematics", Valid: true},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updated.ID.Valid {
		t.Errorf("update returned a row, want a found-but-empty result - " +
			"a foreign teacher's row must not be reachable")
	}

	deleted, err := edu.Delete(context.Background(), tx, eduUUID, 8)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleted.ID.Valid {
		t.Errorf("delete returned a row, want a found-but-empty result - " +
			"a foreign teacher's row must not be reachable")
	}

	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestEducationWriteReturnsTheStoredRow is criterion 25's write-side half: the
// response carries the row the database wrote, not the request echoed back.
// NUMERIC(5,2) rewrites "87.5" to "87.50", and only RETURNING can show that.
func TestEducationWriteReturnsTheStoredRow(t *testing.T) {
	db, mock := newMock(t)
	start := time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC)
	now := time.Date(2026, 9, 29, 10, 11, 12, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`WITH written AS \(UPDATE teacher_education_histories SET`).
		WillReturnRows(sqlmock.NewRows(educationWriteCols()).AddRow(
			11, "d0de0000-0000-4000-8000-000000000011", int64(7), int64(2), "BACHELOR",
			"Universitas Indonesia", start, end, "87.50", "Mathematics", now, now,
		))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}

	got, err := dao.NewEducationDAO(db).Update(context.Background(), tx, "d0de...", 7,
		repository.TeacherEducationHistoryModel{
			InstitutionLevelID: sql.NullInt64{Int64: 2, Valid: true},
			Institution:        sql.NullString{String: "Universitas Indonesia", Valid: true},
			StudyStartDate:     sql.NullTime{Time: start, Valid: true},
			StudyEndDate:       sql.NullTime{Time: end, Valid: true},
			Score:              sql.NullString{String: "87.5", Valid: true},
			FocusedSubject:     sql.NullString{String: "Mathematics", Valid: true},
		})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if !got.ID.Valid {
		t.Fatal("no row returned")
	}
	if got.Score.String != "87.50" {
		t.Errorf("score: got %q, want the stored digits rather than the request's", got.Score.String)
	}
	if got.InstitutionLevel.String != "BACHELOR" {
		t.Errorf("institution_level: got %q, want the joined code", got.InstitutionLevel.String)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

// TestEducationDeleteIsHardDelete guards the other half of criterion 27. The
// teachers table soft-deletes via status, so "delete an education row" could
// plausibly drift into an UPDATE on a `deleted` column - except this table has
// no such column, which is exactly why it must never be registered with
// audit_helper either.
func TestEducationDeleteIsHardDelete(t *testing.T) {
	src := sourceOf(t, "dao/education_dao.go")
	deleteSQL := statementLiteralAfter(t, src, "DELETE FROM teacher_education_histories")
	if !strings.Contains(deleteSQL, "DELETE FROM") {
		t.Error("criterion 27: the education delete is no longer a real DELETE")
	}

	updateSQL := statementLiteralAfter(t, src, "UPDATE teacher_education_histories SET")
	setClause := regexp.MustCompile(`(?s)UPDATE teacher_education_histories SET(.*?)WHERE`).FindStringSubmatch(updateSQL)
	if setClause == nil {
		t.Fatal("could not isolate the education UPDATE SET clause")
	}
	if regexp.MustCompile(`\bdeleted\b`).MatchString(setClause[1]) {
		t.Error("education rows must hard-delete - and this table has no `deleted` column to write")
	}

	// sqlmock cannot catch a dropped teacher_id predicate: the statement still
	// matches its regex, and the args it was told to expect still match too -
	// the mock never checks that the placeholders it was handed are the ones
	// the SQL actually uses. Only reading the statement can.
	if !regexp.MustCompile(`AND teacher_id = \$2`).MatchString(deleteSQL) {
		t.Error("criteria 27/28: the education delete is no longer scoped to its teacher, " +
			"so a caller could remove another teacher's row")
	}
	if !regexp.MustCompile(`AND teacher_id = \$8`).MatchString(updateSQL) {
		t.Error("criteria 27/28: the education update is no longer scoped to its teacher")
	}
}

func TestInstitutionLevelGetByNameTreatsUnknownCodeAsAbsent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM institution_levels WHERE name = \$1`).
		WithArgs("DIPLOMA").
		WillReturnRows(sqlmock.NewRows([]string{"id", "uuid_key", "name", "created_at", "updated_at"}))

	got, err := dao.NewInstitutionLevelDAO(db).GetByName(context.Background(), "DIPLOMA")
	if err != nil {
		t.Fatalf("an unknown level code must not surface as a DAO error: %v", err)
	}
	if got.ID.Valid {
		t.Errorf("expected an invalid ID for an unseeded code, got %+v", got.ID)
	}
}
