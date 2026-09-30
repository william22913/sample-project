package service_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"

	"sample-project/dto/out"
)

// TestGetDetailTeacherReturnsTheEducations is criterion 1's read half and
// criterion 26's ordering: the detail view carries the education rows, each
// already resolved to its level code rather than the surrogate id the FK
// holds.
func TestGetDetailTeacherReturnsTheEducations(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qGetEducations).
		WithArgs(teacherID).
		WillReturnRows(educationRows())

	_, payload, err := h.service.GetDetailTeacher(newContext(), urlParam(teacherUUID), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.TeacherDetailOut)
	if !ok {
		t.Fatalf("payload is %T, want out.TeacherDetailOut", payload)
	}
	if got.ID != teacherUUID {
		t.Errorf("id: got %q, want the uuid_key, never the internal bigint", got.ID)
	}
	if len(got.Educations) != 1 {
		t.Fatalf("educations: got %d, want 1", len(got.Educations))
	}

	education := got.Educations[0]
	if education.InstitutionLevel != "BACHELOR" {
		t.Errorf("institution_level: got %q, want the code", education.InstitutionLevel)
	}
	if education.TeacherID != teacherUUID {
		t.Errorf("teacher_id: got %q, want the teacher's uuid_key", education.TeacherID)
	}
	if education.StudyEndDate != "2020-06-01" {
		t.Errorf("study_end_date: got %q, want date-only", education.StudyEndDate)
	}
	// The digits Postgres holds, not a re-formatted float - "87.50", not 87.5
	// (criterion 25).
	if education.Score != "87.50" {
		t.Errorf("score: got %q, want the stored digits", education.Score)
	}
}

// TestGetDetailTeacherRendersNoEducationsAsAnEmptyArray is criterion 13. A
// client cannot tell "no history" from "field omitted" if the array is null,
// so the empty state has to be a real empty array - asserted through JSON,
// because that is the layer where null and [] differ.
func TestGetDetailTeacherRendersNoEducationsAsAnEmptyArray(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qGetEducations).
		WithArgs(teacherID).
		WillReturnRows(sqlmock.NewRows(educationCols())) // no rows

	_, payload, err := h.service.GetDetailTeacher(newContext(), urlParam(teacherUUID), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"educations":[]`) {
		t.Errorf("educations rendered as something other than an empty array: %s", encoded)
	}
}

// TestGetDetailTeacherReportsAnUnknownID: the DAO reports a missing row as a
// found-but-empty model, and this is the layer that has to turn that into a
// decision - 404-shaped, naming the field the caller got wrong.
func TestGetDetailTeacherReportsAnUnknownID(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows()) // no such row

	_, payload, err := h.service.GetDetailTeacher(newContext(), urlParam(teacherUUID), nil)

	if code := errorCode(err); code != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", code)
	}
	if payload != nil {
		t.Errorf("payload: got %v, want nil", payload)
	}
	// The education read must not have run: there is no teacher to scope it to.
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestGetDetailTeacherResolvesThePathByUUIDKey pins the routing decision this
// service made: `{id}` is uuid_key. Addressed by the internal bigint or by
// teacher_code it would still return 200 for this request, which is why the
// argument is asserted rather than just the outcome.
func TestGetDetailTeacherResolvesThePathByUUIDKey(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qGetEducations).
		WithArgs(teacherID).
		WillReturnRows(sqlmock.NewRows(educationCols()))

	if _, _, err := h.service.GetDetailTeacher(newContext(), urlParam(teacherUUID), nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}
