package service_test

import (
	"database/sql/driver"
	"testing"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
)

// newDeactivateIn is the deactivate body as the validator would leave it: the
// optimistic-lock token and nothing else. MenuDeactivate is its own menu for
// exactly this reason - registered as "update", the validator would demand
// first_name, last_name, email, phone and status in a request that has no
// business sending them.
func newDeactivateIn() *in.TeacherIn {
	return &in.TeacherIn{UpdatedAt: rowUpdatedAt}
}

// TestDeactivateTeacherSetsInactive is criterion 18. Deactivation is a status
// write, not a delete: the row survives with every column but status/updated_*
// intact, and it stays readable afterwards - which is what criterion 19's
// unreachable hard-delete path is protecting.
//
// Audited as an update, because at the row level that is what happened. The
// before-snapshot is read before the write so the entry records the status the
// teacher held as well as the one they were moved to.
func TestDeactivateTeacherSetsInactive(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qDeactivate).
		WithArgs(nil, teacherID, rowUpdatedAt).
		WillReturnRows(teacherRows().AddRow(inactiveTeacherValues()...))
	h.mock.ExpectCommit()

	_, payload, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), newDeactivateIn())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.TeacherOut)
	if !ok {
		t.Fatalf("payload is %T, want out.TeacherOut", payload)
	}
	if got.Status != constanta.StatusInactive {
		t.Errorf("status: got %q, want %q", got.Status, constanta.StatusInactive)
	}
	// Everything else survives - this is the "not a delete" half of criterion
	// 18, asserted rather than assumed.
	if got.Email != "ada@school.edu" || got.TeacherCode != "TCH-2024-007" {
		t.Errorf("deactivation dropped data: %+v", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestDeactivateTeacherRefusesAnAlreadyInactiveRow is criterion 21 read the
// other way round: nothing transitions into Inactive except this route, and
// nothing transitions out of it - so a second deactivation is a terminal-state
// rejection, not a stale write.
//
// The token sent here is deliberately stale as well. Both reasons apply, and
// this test pins which one wins: the more specific, actionable one.
func TestDeactivateTeacherRefusesAnAlreadyInactiveRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(inactiveTeacherValues()...))

	body := newDeactivateIn()
	body.UpdatedAt = rowUpdatedAt.Add(-1)

	_, _, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), body)

	if got := errorCode(err); got != "E-4-TCH-SRV-003" {
		t.Fatalf("got %q, want E-4-TCH-SRV-003 - the terminal state outranks the stale token", got)
	}
	// No transaction: a request refused before the lock check never reached one.
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestDeactivateTeacherRefusesAStaleToken is criterion 36 applied to this
// route. Leaving it unchecked would be a second, weaker door into the same row:
// a deactivation built on a stale read is as wrong as a stale edit.
func TestDeactivateTeacherRefusesAStaleToken(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	body := newDeactivateIn()
	body.UpdatedAt = rowUpdatedAt.Add(-1)

	_, _, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), body)

	if got := errorCode(err); got != "E-4-CMD-DTO-007" {
		t.Fatalf("got %q, want E-4-CMD-DTO-007", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
	if n := h.stream.published(); n != 0 {
		t.Errorf("audit published %d entries for a refused deactivation, want 0", n)
	}
}

// TestDeactivateTeacherReportsAConcurrentWriteAsLocked. The DAO's WHERE also
// carries `status <> 'INACTIVE'`, so a zero-row result is ambiguous - it could
// be the other writer or another deactivation. Only the pre-read can tell them
// apart, and this is the case where it said "still active", so the honest
// answer is the optimistic lock.
func TestDeactivateTeacherReportsAConcurrentWriteAsLocked(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qDeactivate).
		WithArgs(nil, teacherID, rowUpdatedAt).
		WillReturnRows(teacherRows()) // zero rows: the WHERE matched nothing
	h.mock.ExpectRollback()

	_, payload, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), newDeactivateIn())

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

// TestDeactivateTeacherReportsAnUnknownID: the caller gets the field-level
// error, not a 500 and not a silent success.
func TestDeactivateTeacherReportsAnUnknownID(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows())

	_, _, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), newDeactivateIn())

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
}

// TestDeactivateTeacherRecordsTheActorAsNull: no authentication yet, so
// updated_by is NULL (architecture A2). The argument is asserted because that
// is where NULL and an unset int64 differ.
func TestDeactivateTeacherRecordsTheActorAsNull(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qAuditSnapshot).
		WithArgs(false, teacherID).
		WillReturnRows(auditRows(teacherID))
	h.mock.ExpectQuery(qDeactivate).
		WithArgs(driver.Value(nil), teacherID, rowUpdatedAt).
		WillReturnRows(teacherRows().AddRow(inactiveTeacherValues()...))
	h.mock.ExpectCommit()

	if _, _, err := h.service.DeactivateTeacher(newContext(), urlParam(teacherUUID), newDeactivateIn()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
