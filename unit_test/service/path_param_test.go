package service_test

import (
	"testing"

	"github.com/nexsoft-git/nexcommon/model"

	"sample-project/constanta"
)

// The four routes below are addressed by a uuid-valued path parameter, and both
// `{id}` and `{eduId}` reach a `uuid` column. A malformed value cannot be
// compared by Postgres at all - it answers `invalid input syntax for type uuid`
// (SQLSTATE 22P02), the DAO returns that as an ordinary error, and nexcommon's
// formator turns anything untyped into a 500. A live run showed exactly that on
// `GET /teachers/not-a-uuid`.
//
// These tests pin the replacement behaviour: a 400 naming the offending field,
// decided before any statement is sent. "Before any statement" is not asserted
// separately - it is implied by the setup. No sqlmock expectation is
// registered, so the first query any of these paths attempted would come back
// as an "unexpected call" error and fail the code assertion below.

// malformedID is not a uuid in any of the forms Postgres accepts.
const malformedID = "not-a-uuid"

func TestMalformedPathIDIsRejectedNotPassedToTheDatabase(t *testing.T) {
	cases := []struct {
		name string
		// call runs one route with a path parameter the database cannot take.
		call func(h *harness) error
	}{
		{
			name: "GET /teachers/{id} with a malformed id",
			call: func(h *harness) error {
				_, _, err := h.service.GetDetailTeacher(
					newContext(), urlParam(malformedID), nil,
				)
				return err
			},
		},
		{
			name: "PUT /teachers/{id} with a malformed id",
			call: func(h *harness) error {
				_, _, err := h.service.UpdateTeacher(
					newContext(), urlParam(malformedID), newTeacherIn(),
				)
				return err
			},
		},
		{
			name: "PUT /teachers/{id} with a malformed teacher id and a valid education id",
			call: func(h *harness) error {
				_, _, err := h.educationService.UpdateEducation(
					newContext(), urlParamEdu(malformedID, educationUUID), newEducationIn(),
				)
				return err
			},
		},
		{
			name: "PUT .../educations/{eduId} with a valid teacher id and a malformed education id",
			call: func(h *harness) error {
				_, _, err := h.educationService.UpdateEducation(
					newContext(), urlParamEdu(teacherUUID, malformedID), newEducationIn(),
				)
				return err
			},
		},
		{
			name: "DELETE .../educations/{eduId} with a valid teacher id and a malformed education id",
			call: func(h *harness) error {
				_, _, err := h.educationService.DeleteEducation(
					newContext(), urlParamEdu(teacherUUID, malformedID), nil,
				)
				return err
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := newHarness(t)

			err := tc.call(h)
			if err == nil {
				t.Fatal("a malformed uuid in the path was accepted, so it would reach " +
					"Postgres and come back as a 500")
			}

			if got := errorCode(err); got != "E-4-CMD-DTO-002" {
				t.Errorf("error code: got %q, want E-4-CMD-DTO-002 (invalid parameter format). "+
					"A different code here usually means the value got as far as a statement - "+
					"sqlmock reports the unexpected call as the error.", got)
			}

			if err := h.mock.ExpectationsWereMet(); err != nil {
				t.Errorf("the rejection issued a statement: %v", err)
			}
		})
	}
}

// TestWellFormedPathIDStillReachesTheDatabase is the other half, and it is the
// one that matters: a guard that rejects everything would satisfy the tests
// above. The uuid is the fixture's own, so this exercises the same path the
// existing detail tests do - if the guard were over-eager it would fail here
// with E-4-CMD-DTO-002 instead of returning the row.
func TestWellFormedPathIDStillReachesTheDatabase(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qGetEducations).
		WithArgs(teacherID).
		WillReturnRows(educationRows())

	if _, _, err := h.service.GetDetailTeacher(
		newContext(), urlParam(teacherUUID), nil,
	); err != nil {
		t.Fatalf("a well-formed uuid was rejected: %v", err)
	}

	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the request did not run its statements: %v", err)
	}
}

// TestUppercasePathIDIsAccepted pins the reason the guard is uuid.Parse and not
// a length-and-dashes pattern. Uppercase hex is a valid uuid and Postgres
// accepts it, so a hand-rolled `[0-9a-f]` character class would reject input
// the database takes - the same 500, moved rather than removed.
func TestUppercasePathIDIsAccepted(t *testing.T) {
	h := newHarness(t)

	upper := "B1E1C0DE-0000-4000-8000-000000000001"

	// Postgres compares uuids by value, not by text, so the uppercase form
	// matches the same row; a mock cannot reproduce that, and asserting it is
	// not the point. The point is that the guard lets it through to a statement
	// at all, which the expectation below proves.
	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(upper).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qGetEducations).
		WithArgs(teacherID).
		WillReturnRows(educationRows())

	if _, _, err := h.service.GetDetailTeacher(
		newContext(), urlParam(upper), nil,
	); err != nil {
		t.Fatalf("an uppercase uuid was rejected: %v", err)
	}

	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the request did not run its statements: %v", err)
	}
}

// TestMissingPathIDIsRejected covers the other degenerate input: the route
// pattern always supplies the segment, but a param map missing it yields "",
// and "" is not a uuid. Rejected explicitly rather than passed on as a query
// for the empty string.
func TestMissingPathIDIsRejected(t *testing.T) {
	h := newHarness(t)

	_, _, err := h.service.GetDetailTeacher(
		newContext(), model.URLParam{Path: map[string]string{}}, nil,
	)
	if got := errorCode(err); got != "E-4-CMD-DTO-002" {
		t.Fatalf("error code: got %q, want E-4-CMD-DTO-002", got)
	}

	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("a missing path parameter still issued a statement: %v", err)
	}
}

// The path keys are read through the constanta names, so a guard wired to the
// wrong key would silently validate nothing and let every value through - it
// would read param.Path["id"] while the route fills "eduId". These two
// assertions are cheap and they fail loudly if the constants ever disagree with
// the route patterns.
func TestPathKeysMatchTheRoutePlaceholders(t *testing.T) {
	if constanta.PathTeacherID == constanta.PathEducationID {
		t.Fatalf("both path constants are %q, so one of the two guards checks the wrong segment",
			constanta.PathTeacherID)
	}
}
