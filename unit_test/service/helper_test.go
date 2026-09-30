// Package service_test exercises the teacher service layer the way the HTTP
// controller does: a validated DTO and a URLParam in, a response payload and an
// error code out.
//
// The real audit_helper is used, not a fake. It cannot be faked: its
// InitAuditService returns *auditBuilder, an unexported type, so no type
// outside the library can satisfy the AuditHelper interface. Driving the real
// one is also the more honest test - it is the component that owns the
// transaction, and every assertion about "nothing was written" below is really
// an assertion about what it did with that transaction.
//
// What sits underneath it is a sqlmock *sql.DB, so the SQL is observable, and a
// JetStreamContext that records instead of sending (see recordingJetStream).
package service_test

import (
	"database/sql/driver"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/nats-io/nats.go"
	nexctx "github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services/audit_helper"

	"sample-project/constanta"
	"sample-project/dao"
	teacher "sample-project/service/teacher"
)

// ---------------------------------------------------------------------------
// harness
// ---------------------------------------------------------------------------

// recordingJetStream stands in for the NATS connection the audit helper
// publishes to after a commit.
//
// It embeds the interface, left nil, so the dozen methods this test never calls
// are satisfied without being written out; Publish is the only one overridden.
// A nil embedded interface would panic if anything else were reached, which is
// the right failure for a test: it means the helper started using a part of
// NATS this harness does not model.
type recordingJetStream struct {
	nats.JetStreamContext

	mu       sync.Mutex
	subjects []string
	payloads [][]byte
}

func (j *recordingJetStream) Publish(subj string, data []byte, _ ...nats.PubOpt) (*nats.PubAck, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.subjects = append(j.subjects, subj)
	j.payloads = append(j.payloads, data)

	return &nats.PubAck{}, nil
}

func (j *recordingJetStream) published() int {
	j.mu.Lock()
	defer j.mu.Unlock()

	return len(j.payloads)
}

const auditSubject = "audit.teacher"

type harness struct {
	// service and educationService are two values in the same package, matching
	// what the composition root registers: nexcommon reads one body DTO per
	// service value, so a teacher edit and an education edit cannot share one.
	service          *teacher.Service
	educationService *teacher.EducationService
	mock             sqlmock.Sqlmock
	stream           *recordingJetStream
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	stream := &recordingJetStream{}

	// enable=true, and there is no test here for the disabled case on purpose:
	// a disabled helper makes every write return ErrAuditIsDisabled (404), and
	// the composition root hardcodes it on for exactly that reason. Testing the
	// off branch would be testing a deployment this service never ships.
	helper := audit_helper.NewAuditHelper(true, db, stream, auditSubject, "public")

	teacherDAO := dao.NewTeacherDAO(db)
	educationDAO := dao.NewEducationDAO(db)
	levelDAO := dao.NewInstitutionLevelDAO(db)

	return &harness{
		service:          teacher.NewService(teacherDAO, educationDAO, levelDAO, helper),
		educationService: teacher.NewEducationService(teacherDAO, educationDAO, levelDAO, db),
		mock:             mock,
		stream:           stream,
	}
}

// newContext is the zero ContextModel the controller would have built for a
// whitelisted route. It carries no actor, so audit_helper omits created_by from
// its snapshot query and every column it fills is NULL - which is this phase's
// documented state (architecture A2).
func newContext() *nexctx.ContextModel {
	return nexctx.NewContextModel()
}

func urlParam(id string) model.URLParam {
	return model.URLParam{Path: map[string]string{constanta.PathTeacherID: id}}
}

// errorCode reads the code out of a returned error. With no formator installed
// in this test binary, UnbundledErrorMessages.Error() is the code verbatim -
// the same reading the DTO tests rely on.
func errorCode(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// ---------------------------------------------------------------------------
// SQL fragments
//
// Loose on purpose: the DAO tests own statement shape (they read the source and
// assert on clauses a regex cannot honestly express). These match only far
// enough to order the expectations, which is what sqlmock needs.
// ---------------------------------------------------------------------------

var (
	qDuplicateEmail = regexp.QuoteMeta(`SELECT count(id) FROM teachers WHERE lower(email) = lower($1) AND id <> $2`)
	qLevelByName    = regexp.QuoteMeta(`FROM institution_levels WHERE name = $1`)
	qInsertTeacher  = regexp.QuoteMeta(`INSERT INTO teachers`)
	qInsertEdu      = regexp.QuoteMeta(`INSERT INTO teacher_education_histories`)
	// The two are the same statement prefix, so each is anchored on the first
	// column it sets. Left as one matcher, a deactivate test would also accept
	// the update statement - the argument lists differ, so it would still fail,
	// but for the wrong reason.
	qUpdateTeacher = `UPDATE teachers SET\s+first_name = \$1`
	qDeactivate    = `UPDATE teachers SET\s+status     = 'INACTIVE'`
	qGetTeacher    = regexp.QuoteMeta(`FROM teachers WHERE uuid_key = $1`)
	qGetEducations = regexp.QuoteMeta(`WHERE teh.teacher_id = $1`)
	qAuditSnapshot = `SELECT a\.id, uuid_key, row_to_json\(a\) FROM \(SELECT \* FROM .* FOR UPDATE\) a`
)

// ---------------------------------------------------------------------------
// fixtures
// ---------------------------------------------------------------------------

const (
	teacherUUID = "b1e1c0de-0000-4000-8000-000000000001"
	teacherID   = int64(7)
)

func teacherCols() []string {
	return []string{
		"id", "uuid_key", "teacher_code", "first_name", "last_name", "email", "phone",
		"hire_date", "status", "created_by", "updated_by", "created_at", "updated_at",
	}
}

var (
	hireAt       = time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	rowCreatedAt = time.Date(2024, 3, 1, 10, 0, 0, 0, time.UTC)
	rowUpdatedAt = time.Date(2026, 9, 29, 10, 11, 12, 345678000, time.UTC)
)

// teacherValues is a live ACTIVE teacher with a NULL phone and NULL audit
// actors, which is every row this phase writes.
func teacherValues() []driver.Value {
	return []driver.Value{
		teacherID, teacherUUID, "TCH-2024-007", "Ada", "Lovelace",
		"ada@school.edu", nil, hireAt, constanta.StatusActive, nil, nil,
		rowCreatedAt, rowUpdatedAt,
	}
}

func teacherRows() *sqlmock.Rows {
	return sqlmock.NewRows(teacherCols())
}

// inactiveTeacherValues is the same row after deactivation.
func inactiveTeacherValues() []driver.Value {
	v := teacherValues()
	v[8] = constanta.StatusInactive

	return v
}

func levelCols() []string {
	return []string{"id", "uuid_key", "name", "created_at", "updated_at"}
}

func levelRows(id int64, name string) *sqlmock.Rows {
	return sqlmock.NewRows(levelCols()).AddRow(
		id, "c0de0000-0000-4000-8000-00000000000"+string(rune('0'+id%10)), name,
		rowCreatedAt, rowUpdatedAt,
	)
}

func educationCols() []string {
	return []string{
		"id", "uuid_key", "teacher_id", "institution_level_id", "name",
		"institution", "study_start_date", "study_end_date", "score",
		"focused_subject", "created_at", "updated_at",
	}
}

func educationRows() *sqlmock.Rows {
	return sqlmock.NewRows(educationCols()).AddRow(
		11, "d0de0000-0000-4000-8000-000000000011", teacherID, 2, "BACHELOR",
		"Universitas Indonesia",
		time.Date(2016, 8, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2020, 6, 1, 0, 0, 0, 0, time.UTC),
		"87.50", "Mathematics", rowCreatedAt, rowUpdatedAt,
	)
}

// auditRows is what audit_helper's before/after snapshot query returns: the
// primary key, the uuid, and the row as JSON.
func auditRows(id int64) *sqlmock.Rows {
	return sqlmock.NewRows([]string{"id", "uuid_key", "row_to_json"}).
		AddRow(id, teacherUUID, `{"id":7,"email":"ada@school.edu"}`)
}

// The education routes' statements, each anchored on the verb inside the
// educationWrite CTE so one cannot be satisfied by another.
var (
	qUpdateEducation = `WITH written AS \(UPDATE teacher_education_histories SET`
	qDeleteEducation = `WITH written AS \(DELETE FROM teacher_education_histories`
)
