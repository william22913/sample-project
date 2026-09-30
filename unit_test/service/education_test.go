package service_test

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nexsoft-git/nexcommon/model"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
)

const educationUUID = "d0de0000-0000-4000-8000-000000000011"

// urlParamEdu is both path segments the education routes are addressed by. The
// keys are the mux's own placeholder names, not the error-message field names -
// getting that wrong yields an empty lookup and a 404 rather than a compile
// error, which is why the service reads them through named constants.
func urlParamEdu(teacherUUID, educationUUID string) model.URLParam {
	return model.URLParam{Path: map[string]string{
		constanta.PathTeacherID:   teacherUUID,
		constanta.PathEducationID: educationUUID,
	}}
}

func newEducationIn() *in.EducationIn {
	return &in.EducationIn{
		InstitutionLevel: "BACHELOR",
		Institution:      "Universitas Indonesia",
		StudyStartDate:   time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC),
		StudyEndDate:     time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
		Score:            "87.50",
		FocusedSubject:   "Mathematics",
	}
}

// ---------------------------------------------------------------------------
// POST /teachers/{id}/educations
// ---------------------------------------------------------------------------

// TestAddEducationWritesTheRow is the add-education route's happy path. The
// teacher_id it writes is the teacher's *internal* id - the uuid_key in the
// path is resolved before the write, and the surrogate never reaches the wire.
func TestAddEducationWritesTheRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(`WITH written AS \(INSERT INTO teacher_education_histories`).
		WithArgs(
			teacherID, int64(2), "Universitas Indonesia",
			newEducationIn().StudyStartDate, newEducationIn().StudyEndDate,
			"87.50", "Mathematics",
		).
		WillReturnRows(educationRows())
	h.mock.ExpectCommit()

	_, payload, err := h.educationService.AddEducation(
		newContext(), urlParamEdu(teacherUUID, ""), newEducationIn(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.EducationOut)
	if !ok {
		t.Fatalf("payload is %T, want out.EducationOut", payload)
	}
	// The teacher's uuid_key, not the bigint that was written to the column.
	if got.TeacherID != teacherUUID {
		t.Errorf("teacher_id: got %q, want the teacher's uuid_key", got.TeacherID)
	}
	if got.InstitutionLevel != "BACHELOR" {
		t.Errorf("institution_level: got %q, want the code", got.InstitutionLevel)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAddEducationReportsAnUnknownTeacher: the path segment addresses a teacher
// that does not exist, so the caller is told which one - and no level lookup or
// write happens.
func TestAddEducationReportsAnUnknownTeacher(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows())

	_, _, err := h.educationService.AddEducation(newContext(), urlParamEdu(teacherUUID, ""), newEducationIn())

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAddEducationRejectsAFutureEndDate is criterion 29 on this route. The
// teacher read happens first because the route is addressed through one; the
// write does not, because the request cannot be written.
func TestAddEducationRejectsAFutureEndDate(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	body := newEducationIn()
	body.StudyEndDate = time.Now().UTC().AddDate(0, 0, 1)
	body.StudyStartDate = body.StudyEndDate.AddDate(-1, 0, 0)

	_, _, err := h.educationService.AddEducation(newContext(), urlParamEdu(teacherUUID, ""), body)

	if got := errorCode(err); got != "E-4-TCH-SRV-004" {
		t.Fatalf("got %q, want E-4-TCH-SRV-004", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("a rejected row reached the database: %v", err)
	}
}

// TestAddEducationRejectsAnUnknownLevel: the level codes are a seeded, closed
// set (architecture A4) and the DTO does not repeat them as an enum, so this
// lookup is where a bad code becomes the caller's error rather than a 500.
func TestAddEducationRejectsAnUnknownLevel(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(sqlmock.NewRows(levelCols()))

	_, _, err := h.educationService.AddEducation(newContext(), urlParamEdu(teacherUUID, ""), newEducationIn())

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestAddEducationWritesNoAuditEntry: architecture A1 registers only `teachers`
// for auditing. An education row is not an audited entity, so this route must
// not publish anything - and must not borrow audit_helper's transaction to do
// it, which is what would have happened had it been routed through
// InitAuditService with an empty audit list.
func TestAddEducationWritesNoAuditEntry(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(`WITH written AS \(INSERT INTO teacher_education_histories`).
		WillReturnRows(educationRows())
	h.mock.ExpectCommit()

	if _, _, err := h.educationService.AddEducation(newContext(), urlParamEdu(teacherUUID, ""), newEducationIn()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := h.stream.published(); n != 0 {
		t.Errorf("audit published %d entries for an education write, want 0", n)
	}
}

// ---------------------------------------------------------------------------
// PUT /teachers/{id}/educations/{eduId}
// ---------------------------------------------------------------------------

// TestUpdateEducationWritesTheRow is the edit route's happy path. Both path
// segments reach the driver: the uuid_key selects the row, and the teacher's
// internal id is what stops a valid key from reaching another teacher's row.
func TestUpdateEducationWritesTheRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qUpdateEducation).
		WithArgs(
			int64(2), "Universitas Indonesia",
			newEducationIn().StudyStartDate, newEducationIn().StudyEndDate,
			"87.50", "Mathematics", educationUUID, teacherID,
		).
		WillReturnRows(educationRows())
	h.mock.ExpectCommit()

	_, payload, err := h.educationService.UpdateEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), newEducationIn(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := payload.(out.EducationOut); !ok {
		t.Fatalf("payload is %T, want out.EducationOut", payload)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateEducationDoesNotReachAnotherTeachersRow is criterion 28. A key that
// exists but belongs to someone else produces the same answer as a key that
// does not exist, so the route cannot be used to discover which keys are real.
func TestUpdateEducationDoesNotReachAnotherTeachersRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectQuery(qLevelByName).
		WithArgs("BACHELOR").
		WillReturnRows(levelRows(2, "BACHELOR"))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qUpdateEducation).
		WillReturnRows(sqlmock.NewRows(educationCols())) // zero rows
	h.mock.ExpectRollback()

	_, _, err := h.educationService.UpdateEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), newEducationIn(),
	)

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestUpdateEducationRejectsAnInvertedRange is the range rule on the edit path,
// where it is as reachable as on create: both dates are editable, so a caller
// can turn a valid row into an invalid one.
func TestUpdateEducationRejectsAnInvertedRange(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	body := newEducationIn()
	body.StudyStartDate = body.StudyEndDate // equal, which the CHECK allows and "precede" does not

	_, _, err := h.educationService.UpdateEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), body,
	)

	if got := errorCode(err); got != "E-4-TCH-SRV-005" {
		t.Fatalf("got %q, want E-4-TCH-SRV-005", got)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("a rejected row reached the database: %v", err)
	}
}

func TestUpdateEducationReportsAnUnknownTeacher(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows())

	_, _, err := h.educationService.UpdateEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), newEducationIn(),
	)

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
}

// ---------------------------------------------------------------------------
// DELETE /teachers/{id}/educations/{eduId}
// ---------------------------------------------------------------------------

// TestDeleteEducationRemovesOneRow is criterion 27: the row goes, its siblings
// and the teacher stay, and the caller is handed back the row that was removed
// as the only evidence of which one it was.
//
// dtoIn is nil here on purpose. The endpoint is registered with NotReadBody(),
// so the controller never decodes a body for this route and never builds a DTO
// - passing nil is what the controller will actually do.
func TestDeleteEducationRemovesOneRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDeleteEducation).
		WithArgs(educationUUID, teacherID).
		WillReturnRows(educationRows())
	h.mock.ExpectCommit()

	_, payload, err := h.educationService.DeleteEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, ok := payload.(out.EducationOut)
	if !ok {
		t.Fatalf("payload is %T, want out.EducationOut", payload)
	}
	if got.ID != educationUUID {
		t.Errorf("id: got %q, want the key that was addressed", got.ID)
	}
	if got.TeacherID != teacherUUID {
		t.Errorf("teacher_id: got %q, want the teacher's uuid_key", got.TeacherID)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

// TestDeleteEducationReportsAnUnknownRow covers both a key that does not exist
// and a key belonging to another teacher - deliberately the same answer, which
// is what stops the route from confirming that a key is real.
func TestDeleteEducationReportsAnUnknownRow(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))
	h.mock.ExpectBegin()
	h.mock.ExpectQuery(qDeleteEducation).
		WithArgs(educationUUID, teacherID).
		WillReturnRows(sqlmock.NewRows(educationCols())) // zero rows
	h.mock.ExpectRollback()

	_, payload, err := h.educationService.DeleteEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), nil,
	)

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
	if payload != nil {
		t.Errorf("payload: got %v, want nil", payload)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestDeleteEducationReportsAnUnknownTeacher(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(qGetTeacher).
		WithArgs(teacherUUID).
		WillReturnRows(teacherRows())

	_, _, err := h.educationService.DeleteEducation(
		newContext(), urlParamEdu(teacherUUID, educationUUID), nil,
	)

	if got := errorCode(err); got != "E-4-CMD-DTO-004" {
		t.Fatalf("got %q, want E-4-CMD-DTO-004", got)
	}
}
