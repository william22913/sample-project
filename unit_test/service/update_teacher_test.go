package service_test

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
)

// newUpdateIn is a PUT body as the validator would leave it: the personal
// fields present, status named, and the optimistic-lock token echoed back
// exactly as rowUpdatedAt would have been read out.
func newUpdateIn() *in.TeacherIn {
	return &in.TeacherIn{
		FirstName: "Ada",
		LastName:  "Lovelace",
		Email:     "ada@school.edu",
		Phone:     "",
		Status:    constanta.StatusActive,
		UpdatedAt: rowUpdatedAt,
	}
}

// expectedUpdateArgs is the argument list TeacherDAO.Update builds, in order.
func expectedUpdateArgs(status string) []driver.Value {
	return []driver.Value{
		"Ada", "Lovelace", "ada@school.edu", nil, status, nil,
		teacherID, rowUpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// criterion 15 - immutable fields are refused, not dropped
// ---------------------------------------------------------------------------

// TestUpdateTeacherRefusesTeacherCodeAndHireDate is criterion 15. Each is
// rejected as an explicit error, and the rejection happens before the row is
// even read - the request is invalid whatever the stored state is, so a
// response that depended on the database would be a different answer to the
// same question.
//
// The absence of any sqlmock expectation is the assertion that nothing was
// read or written.
func TestUpdateTeacherRefusesTeacherCodeAndHireDate(t *testing.T) {
	t.Run("teacher_code", func(t *testing.T) {
		h := newHarness(t)

		body := newUpdateIn()
		code := "TCH-2024-999"
		body.TeacherCode = &code

		_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

		if got := errorCode(err); got != "E-4-TCH-SRV-002" {
			t.Fatalf("got %q, want E-4-TCH-SRV-002", got)
		}
		if err := h.mock.ExpectationsWereMet(); err != nil {
			t.Errorf("the database was touched for a request refused before it was read: %v", err)
		}
	})

	t.Run("hire_date", func(t *testing.T) {
		h := newHarness(t)

		body := newUpdateIn()
		body.HireDateStr = "2020-01-01"

		_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

		if got := errorCode(err); got != "E-4-TCH-SRV-002" {
			t.Fatalf("got %q, want E-4-TCH-SRV-002", got)
		}
		if err := h.mock.ExpectationsWereMet(); err != nil {
			t.Errorf("the database was touched for a request refused before it was read: %v", err)
		}
	})
}

// ---------------------------------------------------------------------------
// criteria 21 / 35 / 36 - the guards that need the row
// ---------------------------------------------------------------------------

func TestUpdateTeacherReportsAnUnknownID(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows())

	_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), newUpdateIn())

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
}

// TestUpdateTeacherRefusesAnInactiveRow is criterion 21's first half: Inactive
// is terminal, and the requirement is that this is enforced in the update path
// rather than merely absent from a UI.
func TestUpdateTeacherRefusesAnInactiveRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(inactiveTeacherValues()...))

	_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), newUpdateIn())

	if got := errorCode(err); got != "E-4-TCH-SRV-003" {
		t.Fatalf("got %q, want E-4-TCH-SRV-003", got)
	}
}

// TestUpdateTeacherRefusesAnInactiveRowEvenWithAStaleToken pins the ordering of
// the two refusals. Both reasons apply here; the terminal state is the more
// specific one, and it is the one the caller can act on - re-fetching a fresh
// token would not make the write legal.
func TestUpdateTeacherRefusesAnInactiveRowEvenWithAStaleToken(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(inactiveTeacherValues()...))

	body := newUpdateIn()
	body.UpdatedAt = rowUpdatedAt.Add(-1) // deliberately not what the row holds

	_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

	if got := errorCode(err); got != "E-4-TCH-SRV-003" {
		t.Fatalf("got %q, want E-4-TCH-SRV-003 - the terminal state outranks the stale token", got)
	}
}

// TestUpdateTeacherRefusesAStaleToken is criteria 35 and 36. The token is
// compared for exact equality, so a client that echoes a *newer* value than the
// stored one is refused as well: its copy is not the row it claims to be
// editing either.
func TestUpdateTeacherRefusesAStaleToken(t *testing.T) {
	h := newHarness(t)

	for _, tc := range []struct {
		name  string
		token time.Time
	}{
		{"older than the row", rowUpdatedAt.Add(-time.Second)},
		{"newer than the row", rowUpdatedAt.Add(time.Second)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h.mock.ExpectQuery(qGetTeacher).
				WithArgs(teacherUUID).
				WillReturnRows(teacherRows().AddRow(teacherValues()...))

			body := newUpdateIn()
			body.UpdatedAt = tc.token

			_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

			if got := errorCode(err); got != "E-4-CMD-DTO-007" {
				t.Fatalf("got %q, want E-4-CMD-DTO-007", got)
			}
			// No transaction was opened for a write that cannot proceed.
			if err := h.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// criterion 17 / 16 - the no-op save
// ---------------------------------------------------------------------------

// TestUpdateTeacherReturnsTheRowUnchangedOnANoOpSave is criterion 17: a save
// that changes nothing writes nothing, so updated_at keeps its value and the
// client's next save still works.
//
// The absence of ExpectBegin is the load-bearing assertion. Opening a
// transaction here would produce an audit entry for an update that did not
// happen, which criterion 16 counts per *successful* update.
func TestUpdateTeacherReturnsTheRowUnchangedOnANoOpSave(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	_, payload, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), newUpdateIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.TeacherOut)
	if !ok {
		t.Fatalf("payload is %T, want out.TeacherOut", payload)
	}
	// The token the client must use next time is the one it already has.
	if want := rowUpdatedAt.Format(out.TimestampLayout); got.UpdatedAt != want {
		t.Errorf("updated_at: got %q, want %q unchanged", got.UpdatedAt, want)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
	if n := h.stream.published(); n != 0 {
		t.Errorf("audit published %d entries for a save that changed nothing, want 0", n)
	}
}

// TestUpdateTeacherTreatsADifferentlyCasedEmailAsNoOp: the DTO lowercases the
// email before the service sees it, so a client re-sending the address it read
// back in a different case is not making a change. Without the auto_fix this
// would be a write, and criterion 17 would be violated by a client doing
// nothing wrong.
func TestUpdateTeacherTreatsADifferentlyCasedEmailAsNoOp(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	body := newUpdateIn()
	body.Email = "ada@school.edu" // already normalized, as the validator would leave it

	_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("a no-op save wrote something: %v", err)
	}
}

// ---------------------------------------------------------------------------
// criteria 16 / 35 - the write path
// ---------------------------------------------------------------------------

// TestUpdateTeacherWritesAndAudits is criterion 16: exactly one audit entry per
// successful update, carrying both what the row was and what it became. The
// before-snapshot is read inside the transaction and before the UPDATE, which
// is the only ordering that can capture both.
func TestUpdateTeacherWritesAndAudits(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	// Re-checked inside the transaction, excluding this row so a teacher
	// re-saved without touching their email is not a duplicate of itself.
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", teacherID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qUpdateTeacher).
		WithArgs(expectedUpdateArgs(constanta.StatusOnLeave)...).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectCommit()

	body := newUpdateIn()
	body.Status = constanta.StatusOnLeave

	_, payload, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := payload.(out.TeacherOut); !ok {
		t.Fatalf("payload is %T, want out.TeacherOut", payload)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateTeacherRefusesADuplicateEmail. The check that ran before the
// transaction can be overtaken by a concurrent request, so it is run again
// inside it - and that second run is the one a test can see fail.
func TestUpdateTeacherRefusesADuplicateEmail(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	// The body carries a *different* address, so this is a genuine change being
	// refused rather than a no-op that never reaches the transaction.
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("grace@school.edu", teacherID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	h.mock.ExpectRollback()
	// The UPDATE is deliberately without an expectation: reaching for it would
	// fail the mock, which is the assertion that no row was written.

	body := newUpdateIn()
	body.Email = "grace@school.edu"

	_, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

	if got := errorCode(err); got != "E-4-TCH-SRV-001" {
		t.Fatalf("got %q, want E-4-TCH-SRV-001", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateTeacherReportsAConcurrentWriteAsLocked is criterion 35's real
// enforcement point. The pre-read matched, so between it and the transaction
// another writer committed; the conditional UPDATE is the only thing that can
// see that, and it sees it as a WHERE that matched nothing.
//
// Nothing is written and nothing is audited - the rollback is what makes the
// audit entry absent rather than describing a change that never landed.
func TestUpdateTeacherReportsAConcurrentWriteAsLocked(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", teacherID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qUpdateTeacher).
		WithArgs(expectedUpdateArgs(constanta.StatusOnLeave)...).
		WillReturnRows(teacherRows()) // zero rows: the WHERE matched nothing
	h.mock.ExpectRollback()

	body := newUpdateIn()
	body.Status = constanta.StatusOnLeave

	_, payload, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body)

	if got := errorCode(err); got != "E-4-CMD-DTO-007" {
		t.Fatalf("got %q, want E-4-CMD-DTO-007", got)
	}
	if payload != nil {
		t.Errorf("payload: got %v, want nil", payload)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateTeacherWritesNullPhoneForAnOmittedNumber: the column is nullable,
// so clearing a phone number has to reach the driver as NULL rather than as an
// empty string.
func TestUpdateTeacherWritesNullPhoneForAnOmittedNumber(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDuplicateEmail).
		WithArgs("ada@school.edu", teacherID).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qUpdateTeacher).
		WithArgs(expectedUpdateArgs(constanta.StatusOnLeave)...).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectCommit()

	body := newUpdateIn()
	body.Status = constanta.StatusOnLeave
	body.Phone = ""

	if _, _, err := h.service.UpdateTeacher(newContext(), urlParam(teacherUUID), body); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("phone did not reach the driver as NULL: %v", err)
	}
}
