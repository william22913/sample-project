package dao_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"sample-project/dao"
	"sample-project/repository"
)

func newMock(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}

func teacherRow() *sqlmock.Rows {
	return sqlmock.NewRows([]string{
		"id", "uuid_key", "teacher_code", "first_name", "last_name", "email", "phone",
		"hire_date", "status", "created_by", "updated_by", "created_at", "updated_at",
	})
}

// Null-valued columns are passed as nil so the scan exercises the sql.Null*
// wrappers rather than a comfortable all-present row.
func teacherRowValues() []driver.Value {
	hire := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	return []driver.Value{
		7, "b1e1c0de-0000-4000-8000-000000000001", "TCH-2024-007", "Ada", "Lovelace",
		"ada@school.edu", nil, hire, "ACTIVE", nil, nil,
		time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC),
	}
}

// TestGetByUUIDKeyScansNullColumns proves a row with a NULL phone and NULL
// audit actors scans without error and that the Null* wrappers report
// Valid=false, which is the state every row is in this phase (no auth means no
// actor to record).
func TestGetByUUIDKeyScansNullColumns(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM teachers WHERE uuid_key = \$1`).
		WithArgs("b1e1c0de-0000-4000-8000-000000000001").
		WillReturnRows(teacherRow().AddRow(teacherRowValues()...))

	got, err := dao.NewTeacherDAO(db).GetByUUIDKey(context.Background(), "b1e1c0de-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.ID.Valid || got.ID.Int64 != 7 {
		t.Errorf("id: got %+v, want 7", got.ID)
	}
	if got.Phone.Valid {
		t.Errorf("phone: got %+v, want invalid (column was NULL)", got.Phone)
	}
	if got.CreatedBy.Valid || got.UpdatedBy.Valid {
		t.Errorf("audit actors: got %+v / %+v, want both invalid", got.CreatedBy, got.UpdatedBy)
	}
	if got.TeacherCode.String != "TCH-2024-007" {
		t.Errorf("teacher_code: got %q", got.TeacherCode.String)
	}
}

// TestGetByUUIDKeyTreatsNoRowsAsAbsent pins the convention every caller leans
// on: a missing row is a found-but-empty model, not an error. A service checks
// ID.Valid to decide between 404 and success.
func TestGetByUUIDKeyTreatsNoRowsAsAbsent(t *testing.T) {
	db, mock := newMock(t)
	mock.ExpectQuery(`FROM teachers WHERE uuid_key = \$1`).
		WithArgs("nope").
		WillReturnRows(teacherRow()) // zero rows

	got, err := dao.NewTeacherDAO(db).GetByUUIDKey(context.Background(), "nope")
	if err != nil {
		t.Fatalf("ErrNoRows must not surface as an error, got: %v", err)
	}
	if got.ID.Valid {
		t.Errorf("expected an invalid ID for a missing row, got %+v", got.ID)
	}
}

// TestInsertReturnsTriggerAssignedCode is criterion 1/2: teacher_code is never
// sent by the client, it comes back from the INSERT ... RETURNING. If someone
// adds teacher_code to the column list, this test still passes - but the
// trigger would then be bypassed - so the statement itself is checked below in
// TestInsertDoesNotSendTeacherCode.
func TestInsertReturnsTriggerAssignedCode(t *testing.T) {
	db, mock := newMock(t)
	hire := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO teachers`).
		WithArgs("Ada", "Lovelace", "ada@school.edu", nil, hire, nil, nil).
		WillReturnRows(teacherRow().AddRow(teacherRowValues()...))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).Insert(context.Background(), tx, repository.TeacherModel{
		FirstName: sql.NullString{String: "Ada", Valid: true},
		LastName:  sql.NullString{String: "Lovelace", Valid: true},
		Email:     sql.NullString{String: "ada@school.edu", Valid: true},
		HireDate:  sql.NullTime{Time: hire, Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TeacherCode.String != "TCH-2024-007" {
		t.Errorf("teacher_code: got %q, want the trigger-assigned TCH-2024-007", got.TeacherCode.String)
	}
	// The DAO does not commit - the caller owns the transaction. Committing
	// here is what proves the DAO issued no commit of its own.
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateIsConditionalOnExpectedTimestamp is criterion 35/36 and
// architecture A5, the single most important assertion in this file: the
// UPDATE carries the client's expected updated_at in its WHERE clause, and a
// stale value comes back as a found-but-empty row rather than an error.
func TestUpdateIsConditionalOnExpectedTimestamp(t *testing.T) {
	db, mock := newMock(t)
	expected := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)

	// A stale write matches nothing: the RETURNING clause yields zero rows.
	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE teachers SET`).
		WithArgs("Ada", "Lovelace", "ada@school.edu", nil, "ACTIVE", nil, int64(7), expected).
		WillReturnRows(teacherRow())
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).Update(context.Background(), tx, 7, repository.TeacherModel{
		FirstName: sql.NullString{String: "Ada", Valid: true},
		LastName:  sql.NullString{String: "Lovelace", Valid: true},
		Email:     sql.NullString{String: "ada@school.edu", Valid: true},
		Status:    sql.NullString{String: "ACTIVE", Valid: true},
	}, expected)

	if err != nil {
		t.Fatalf("a stale write must not be an error at the DAO layer: %v", err)
	}
	if got.ID.Valid {
		t.Errorf("expected an invalid ID for a stale write, got %+v - the service reads "+
			"this as ErrDataLocked", got.ID)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateReturnsThePostWriteRow is what keeps the caller's next lock value
// honest. If Update returned anything but the row Postgres just wrote, a client
// would echo a stale updated_at on its next save and be told it was locked
// forever.
func TestUpdateReturnsThePostWriteRow(t *testing.T) {
	db, mock := newMock(t)
	expected := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)

	row := teacherRowValues()
	// The stored updated_at is CURRENT_TIMESTAMP - a value the application
	// never computes - so the returned model has to come from the database.
	row[12] = time.Date(2024, 3, 2, 8, 30, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE teachers SET`).
		WithArgs("Ada", "Lovelace", "ada@school.edu", nil, "ACTIVE", nil, int64(7), expected).
		WillReturnRows(teacherRow().AddRow(row...))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).Update(context.Background(), tx, 7, repository.TeacherModel{
		FirstName: sql.NullString{String: "Ada", Valid: true},
		LastName:  sql.NullString{String: "Lovelace", Valid: true},
		Email:     sql.NullString{String: "ada@school.edu", Valid: true},
		Status:    sql.NullString{String: "ACTIVE", Valid: true},
	}, expected)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := time.Date(2024, 3, 2, 8, 30, 0, 0, time.UTC)
	if !got.UpdatedAt.Valid || !got.UpdatedAt.Time.Equal(want) {
		t.Errorf("updated_at: got %+v, want %v - the caller hands this value back as the next lock value",
			got.UpdatedAt, want)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestUpdateQueryShape(t *testing.T) {
	src := sourceOf(t, "dao/teacher_dao.go")

	for _, want := range []string{
		`updated_at = CURRENT_TIMESTAMP`,
		`WHERE id = $7 AND updated_at = $8`,
	} {
		if !regexp.MustCompile(regexp.QuoteMeta(want)).MatchString(src) {
			t.Errorf("teacher_dao.go no longer contains %q", want)
		}
	}

	updateSQL := queryLiteralAfter(t, src, "UPDATE teachers SET")
	// The column list itself, not the RETURNING clause. teacher_code must be
	// absent so the trg_teachers_setcode trigger derives it: supplying a value
	// makes the trigger skip its branch, which is the one way a client could
	// choose its own Teacher_ID.
	insertSQL := queryLiteralAfter(t, src, "INSERT INTO teachers")
	insertCols := regexp.MustCompile(`(?s)INSERT INTO teachers(.*?)VALUES`).FindStringSubmatch(insertSQL)
	if insertCols == nil {
		t.Fatal("could not isolate the INSERT column list in teacher_dao.go")
	}
	if regexp.MustCompile(`teacher_code`).MatchString(insertCols[1]) {
		t.Error("criterion 2: teacher_code is in the INSERT column list, so the trigger " +
			"skips generation and the client picks its own Teacher_ID")
	}
	if !regexp.MustCompile(`hire_date`).MatchString(insertCols[1]) {
		t.Error("the INSERT column list no longer carries hire_date, which the trigger reads to build the code")
	}

	setClause := regexp.MustCompile(`(?s)UPDATE teachers SET(.*?)WHERE`).FindStringSubmatch(updateSQL)
	if setClause == nil {
		t.Fatal("could not isolate the UPDATE SET clause in teacher_dao.go")
	}
	for _, forbidden := range []string{"teacher_code", "hire_date"} {
		if regexp.MustCompile(forbidden).MatchString(setClause[1]) {
			t.Errorf("criterion 15: %q is in the UPDATE SET list - editing it must be impossible, not ignored", forbidden)
		}
	}
	for _, required := range []string{"first_name", "last_name", "email", "phone", "status"} {
		if !regexp.MustCompile(required).MatchString(setClause[1]) {
			t.Errorf("the UPDATE SET list no longer writes %q", required)
		}
	}
}

// TestDeactivateRefusesAnAlreadyInactiveRow is criterion 21: Inactive is
// terminal. The `status <> 'INACTIVE'` predicate is the whole mechanism - drop
// it and a second deactivate silently succeeds, which reads as reactivation
// having a path.
func TestDeactivateRefusesAnAlreadyInactiveRow(t *testing.T) {
	db, mock := newMock(t)
	expected := time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)

	mock.ExpectBegin()
	mock.ExpectQuery(`UPDATE teachers SET`).
		WithArgs(nil, int64(7), expected).
		WillReturnRows(teacherRow())
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).Deactivate(context.Background(), tx, 7, sql.NullInt64{}, expected)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID.Valid {
		t.Errorf("expected an invalid ID for a row that is already INACTIVE, got %+v", got.ID)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestCheckDuplicateEmailExcludesTheRowBeingUpdated covers the half of the
// duplicate check that a naive `WHERE lower(email) = lower($1)` gets wrong:
// re-saving a teacher without touching their email would report that email as
// taken by the very row being saved.
func TestCheckDuplicateEmailExcludesTheRowBeingUpdated(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT count\(id\) FROM teachers`).
		WithArgs("ada@school.edu", int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).CheckDuplicateEmail(context.Background(), tx, "ada@school.edu", 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Valid || got.Int64 != 0 {
		t.Errorf("count: got %+v, want a valid 0", got)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}

	// The exclusion has to be in the statement, not just in this test's
	// expectations: sqlmock matches the query by regex, so a DAO that dropped
	// `id <> $2` would still satisfy WithArgs above and the test would pass
	// while the bug shipped.
	dupSQL := queryLiteralAfter(t, sourceOf(t, "dao/teacher_dao.go"), "SELECT count(id) FROM teachers")
	if !regexp.MustCompile(regexp.QuoteMeta("id <> $2")).MatchString(dupSQL) {
		t.Error("CheckDuplicateEmail no longer excludes the row being updated, so " +
			"re-saving a teacher without changing their email reports their own address as taken")
	}
}

// TestCheckDuplicateEmailFindsATakenAddress is criterion 3's precondition.
func TestCheckDuplicateEmailFindsATakenAddress(t *testing.T) {
	db, mock := newMock(t)

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT count\(id\) FROM teachers`).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectCommit()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	got, err := dao.NewTeacherDAO(db).CheckDuplicateEmail(context.Background(), tx, "ada@school.edu", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Valid || got.Int64 != 1 {
		t.Errorf("count: got %+v, want a valid 1", got)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("commit: %v", err)
	}
}

func TestDeactivateQueryShape(t *testing.T) {
	// Scoped to the statement, not the file. Two things go wrong otherwise:
	// the comment above Deactivate quotes this predicate in prose, so a
	// file-wide search still finds it after the guard has been deleted from
	// the SQL - which is how this test once passed a mutation that removed the
	// guard - and `UPDATE teachers SET` opens both this statement and Update's.
	// The marker is the SET value, which no comment repeats.
	deactivateSQL := queryLiteralAfter(t, sourceOf(t, "dao/teacher_dao.go"), "= 'INACTIVE',")
	if !regexp.MustCompile(`status <> 'INACTIVE'`).MatchString(deactivateSQL) {
		t.Error("criterion 21: the deactivate UPDATE lost its `status <> 'INACTIVE'` guard, " +
			"so a second deactivate would report success")
	}
}

// TestTeacherDAOHasNoHardDelete is criterion 19, which asks for unreachability
// rather than discouragement. A reflection walk over the method set is the
// only check that keeps holding when someone adds a method six months from
// now - a comment cannot fail a build, this can.
func TestTeacherDAOHasNoHardDelete(t *testing.T) {
	bad := regexp.MustCompile(`(?i)delete|remove|drop|purge|destroy`)
	typ := reflect.TypeOf(dao.TeacherDAO{})

	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if bad.MatchString(name) {
			t.Errorf("criterion 19 violated: TeacherDAO has a %q method. Hard delete must be "+
				"unreachable - no route and no DAO method - or deactivation is not the only exit.", name)
		}
	}
}

// TestListQueryIsSpaceTerminated guards the one thing about the list query that
// sqlmock cannot see and that no clause-level assertion covers: the get-list
// builder appends its WHERE clause to this string with no separator of its
// own, so a missing trailing space produces `FROM teachersWHERE ...`, which
// Postgres rejects as a single identifier.
//
// Every other statement in this package is spelled out whole at its call site
// and so cannot have this failure mode. This one is assembled.
func TestListQueryIsSpaceTerminated(t *testing.T) {
	src := sourceOf(t, "dao/teacher_dao.go")

	const marker = "const teacherListQuery = `SELECT ` + teacherColumns + ` FROM teachers"
	at := strings.Index(src, marker)
	if at < 0 {
		t.Fatal("could not find the list query's declaration - it has been renamed or restructured")
	}

	rest := src[at+len(marker):]
	if !strings.HasPrefix(rest, " ") {
		t.Errorf("the list query ends without whitespace before the builder's WHERE clause: "+
			"next characters are %q, so the emitted SQL would read `FROM teachersWHERE`", firstLine(rest))
	}
}

// firstLine trims a fragment to something readable in a failure message.
func firstLine(s string) string {
	if i := strings.IndexAny(s, "\n`"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 40 {
		s = s[:40]
	}
	return s
}
