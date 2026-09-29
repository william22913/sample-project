package service_test

import (
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
)

// newTeacherIn is a create body that passes the DTO's own rules. The values are
// what the validator would have left behind, not what a client typed - notably
// hireDate, which arrives as a parsed date rather than the string on the wire.
func newTeacherIn() *in.TeacherIn {
	return &in.TeacherIn{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@school.edu",
		Phone:     "",
		HireDate:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Educations: []in.EducationIn{{
			InstitutionLevel: "BACHELOR",
			Institution:      "Universitas Indonesia",
			StudyStartDate:   time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC),
			StudyEndDate:     time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			Score:            "87.50",
			FocusedSubject:   "Mathematics",
		}},
	}
}

// ---------------------------------------------------------------------------
// criterion 1 / 7 / 13 - create writes both halves atomically
// ---------------------------------------------------------------------------

// TestInsertTeacherWritesTheTeacherAndItsEducations is criteria 1 and 7: the
// create is one transaction spanning the teacher row and every education row,
// and the caller gets back the row the database actually wrote - including
// teacher_code, which no Go code supplies.
func TestInsertTeacherWritesTheTeacherAndItsEducations(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	// teacher_code is absent from the argument list on purpose: the trigger
	// derives it, and a value supplied here would let a client choose its own
	// Teacher_ID (criterion 2).
	h.mock.ExpectQuery(qInsertTeacher).
		WithArgs("Ada", "Lovelace", "ada@school.edu", nil, newTeacherIn().HireDate, nil, nil).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qInsertEdu).
		WithArgs(
			teacherID, int64(2), "Universitas Indonesia",
			newTeacherIn().Educations[0].StudyStartDate,
			newTeacherIn().Educations[0].StudyEndDate,
			"87.50", "Mathematics",
		).
		WillReturnRows(educationRows())
	// audit_helper's own after-write read, run for the insert action only.
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectCommit()

	_, payload, err := h.service.InsertTeacher(newContext(), urlParam(""), newTeacherIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.TeacherOut)
	if !ok {
		t.Fatalf("payload is %T, want out.TeacherOut", payload)
	}
	if got.TeacherCode != "TCH-2024-007" {
		t.Errorf("teacher_code: got %q, want the value the trigger wrote back", got.TeacherCode)
	}
	if got.Status != constanta.StatusActive {
		t.Errorf("status: got %q, want %q - the column DEFAULT decides it, not the request",
			got.Status, constanta.StatusActive)
	}
	if got.Phone != nil {
		t.Errorf("phone: got %v, want nil for an omitted number", *got.Phone)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestInsertTeacherWritesNothingWhenTheEmailIsTaken is criterion 3's second
// half, and the stronger half: not only is the caller told which field is at
// fault, the teacher row is never written. That is what running the check
// inside the transaction buys - the check and the INSERT see one snapshot.
//
// The INSERT expectation is deliberately absent rather than asserted-not-met.
// sqlmock fails the INSERT itself if the code reaches for it, so the absence is
// the assertion.
func TestInsertTeacherWritesNothingWhenTheEmailIsTaken(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	h.mock.ExpectRollback()

	_, payload, err := h.service.InsertTeacher(newContext(), urlParam(""), newTeacherIn())

	if code := errorCode(err); code != "E-4-TCH-SRV-001" {
		t.Fatalf("got %q, want E-4-TCH-SRV-001", code)
	}
	if payload != nil {
		t.Errorf("payload: got %v, want nil on a rejected create", payload)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestInsertTeacherRejectsAnUnknownInstitutionLevel: the level codes are a
// seeded, closed set (architecture A4) and the DTO does not duplicate them as
// an enum, so this lookup is the only place a bad code is caught - and it has
// to be the caller's error, not a 500.
func TestInsertTeacherRejectsAnUnknownInstitutionLevel(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(sqlmock.NewRows(levelCols())) // no such code
	h.mock.ExpectRollback()

	_, _, err := h.service.InsertTeacher(newContext(), urlParam(""), newTeacherIn())

	if code := errorCode(err); code != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", code)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// ---------------------------------------------------------------------------
// criteria 29 / 37 - the date rules, checked before anything is opened
// ---------------------------------------------------------------------------

// TestInsertTeacherRejectsAFutureStudyEnd is criterion 29. The rule is
// application-layer only - a Postgres CHECK cannot call CURRENT_DATE, which is
// not immutable - so this test is the whole enforcement, not a duplicate of a
// constraint.
//
// Asserted against a period ending tomorrow rather than a hardcoded date, so
// the test does not quietly stop exercising the rule once that date passes.
func TestInsertTeacherRejectsAFutureStudyEnd(t *testing.T) {
	h := newHarness(t)

	body := newTeacherIn()
	tomorrow := time.Now().UTC().AddDate(0, 0, 1)
	body.Educations[0].StudyEndDate = tomorrow
	body.Educations[0].StudyStartDate = tomorrow.AddDate(-1, 0, 0)

	_, _, err := h.service.InsertTeacher(newContext(), urlParam(""), body)

	if code := errorCode(err); code != "E-4-TCH-SRV-004" {
		t.Fatalf("got %q, want E-4-TCH-SRV-004", code)
	}
	// No Begin, no expectations: the rejection happens before the transaction
	// exists, so there is nothing to roll back. ExpectationsWereMet passes with
	// an empty set only if the service really did not touch the database.
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the database was touched for a request rejected by a pure date rule: %v", err)
	}
	if n := h.stream.published(); n != 0 {
		t.Errorf("audit published %d times for a rejected create, want 0", n)
	}
}

// TestInsertTeacherRejectsAnInvertedDateRange. The CHECK constraint accepts
// start == end; "must precede" does not. The stricter reading is the spec's, so
// equality has to be refused here rather than allowed through to the database.
func TestInsertTeacherRejectsAnInvertedDateRange(t *testing.T) {
	h := newHarness(t)

	for _, tc := range []struct {
		name  string
		start time.Time
		end   time.Time
	}{
		{
			name:  "start after end",
			start: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "start equals end, which the CHECK would allow",
			start: time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
			end:   time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := newTeacherIn()
			body.Educations[0].StudyStartDate = tc.start
			body.Educations[0].StudyEndDate = tc.end

			_, _, err := h.service.InsertTeacher(newContext(), urlParam(""), body)

			if code := errorCode(err); code != "E-4-TCH-SRV-005" {
				t.Fatalf("got %q, want E-4-TCH-SRV-005", code)
			}
			if err := h.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("the database was touched: %v", err)
			}
		})
	}
}

// TestInsertTeacherAcceptsNoEducations is criterion 13: an empty history is a
// deliberate state, not an error, so the create must succeed with no level
// lookups and no education INSERT at all.
func TestInsertTeacherAcceptsNoEducations(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qInsertTeacher).
		WithArgs("Ada", "Lovelace", "ada@school.edu", nil, newTeacherIn().HireDate, nil, nil).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectCommit()

	body := newTeacherIn()
	body.Educations = nil

	if _, _, err := h.service.InsertTeacher(newContext(), urlParam(""), body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestInsertTeacherRecordsTheActorAsNull: there is no authentication yet
// (architecture A2), so created_by and updated_by are NULL on every row this
// phase writes. Asserted through the arguments the driver receives, because
// that is the only place the difference between "NULL" and "an unset int64"
// shows.
func TestInsertTeacherRecordsTheActorAsNull(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", int64(0)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	h.mock.ExpectQuery(qInsertTeacher).
		WithArgs(
			"Ada", "Lovelace", "ada@school.edu", nil, newTeacherIn().HireDate,
			sql.NullInt64{}, sql.NullInt64{},
		).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qInsertEdu).
		WithArgs(
			teacherID, int64(2), "Universitas Indonesia",
			newTeacherIn().Educations[0].StudyStartDate,
			newTeacherIn().Educations[0].StudyEndDate,
			"87.50", "Mathematics",
		).
		WillReturnRows(educationRows())
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectCommit()

	if _, _, err := h.service.InsertTeacher(newContext(), urlParam(""), newTeacherIn()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
