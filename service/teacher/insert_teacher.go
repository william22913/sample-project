package teacher

import (
	"database/sql"
	"time"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services/audit_helper"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
	errors "sample-project/error"
	"sample-project/repository"
)

// InsertTeacher is the POST /teachers route.
//
// The teacher row and its education rows are one transaction (criterion 7), and
// that transaction is the audit builder's - nexcommon's audit_helper owns it so
// the audit entry and the data it describes commit together. There is no second
// Begin() here and there must not be one.
func (s *Service) InsertTeacher(
	ctx *context.ContextModel,
	_ model.URLParam,
	dtoIn interface{},
) (
	header map[string]string,
	output interface{},
	err error,
) {
	body := dtoIn.(*in.TeacherIn)

	// Checked before the transaction opens. Starting one for a request that
	// cannot be written would be wasted work, and there is nothing to roll back.
	if err = validateEducationDates(ctx, body.Educations); err != nil {
		return
	}

	// InitAuditService errors are returned as-is, with no fallback path. When
	// the helper is constructed with enable=false it returns
	// ErrAuditIsDisabled (404) and a nil builder, and every other service in
	// this ecosystem lets that travel - so a deployment that turns auditing off
	// has no working writes rather than silently unaudited ones. The
	// composition root therefore constructs it enabled.
	audit, err := s.auditHelper.InitAuditService(ctx, dtoIn, s.doInsertTeacher)
	if err != nil {
		return
	}

	output, err = audit.CompletedAuditData()

	return
}

// doInsertTeacher runs inside the audit builder's transaction.
func (s *Service) doInsertTeacher(
	ctx *context.ContextModel,
	tx *sql.Tx,
	dtoIn interface{},
	_ time.Time,
) (
	output interface{},
	audit []audit_helper.AuditSystemModel,
	err error,
) {
	body := dtoIn.(*in.TeacherIn)

	// Criterion 3: a duplicate email is a field-level error naming Email, and
	// zero rows are written. Running the check inside the transaction is what
	// makes the second half true - it and the INSERT see one snapshot, so a
	// failure here means the INSERT never ran.
	//
	// It is a check, not the enforcement. uq_teachers_email_lower on
	// lower(email) is what actually makes email unique (criterion 30): two
	// concurrent requests can both pass this read, and the loser fails the
	// index with a generic 500. That is the correct outcome for a genuine
	// conflict, and it is the reason this pre-check does not pretend to be a
	// lock.
	duplicate, err := s.teacherDAO.CheckDuplicateEmail(ctx.ToContext(), tx, body.Email, 0)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when check duplicate teacher email")
		return
	}
	if duplicate.Int64 > 0 {
		err = errors.ErrDataUsed(constanta.FieldEmail).ContextModel(ctx)
		return
	}

	levelIDs, err := resolveInstitutionLevels(ctx, s.levelDAO, body.Educations)
	if err != nil {
		return
	}

	actor := actorID(ctx)

	// teacher_code is not in this model, and cannot be: the
	// trg_teachers_setcode BEFORE INSERT trigger derives it from the
	// sequence-assigned id and hire_date, and the INSERT ... RETURNING brings
	// it back. Supplying a value would make the trigger skip its branch, which
	// is the one way a client could choose its own Teacher_ID (criterion 2).
	teacher, err := s.teacherDAO.Insert(ctx.ToContext(), tx, repository.TeacherModel{
		FirstName: sql.NullString{String: body.FirstName, Valid: true},
		LastName:  sql.NullString{String: body.LastName, Valid: true},
		Email:     sql.NullString{String: body.Email, Valid: true},
		Phone:     sql.NullString{String: body.Phone, Valid: body.Phone != ""},
		HireDate:  sql.NullTime{Time: body.HireDate, Valid: true},
		CreatedBy: actor,
		UpdatedBy: actor,
	})
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when insert teacher")
		return
	}

	if err = s.insertEducations(ctx, tx, teacher.ID.Int64, body.Educations, levelIDs); err != nil {
		return
	}

	// Built from the INSERT ... RETURNING row, not re-read: the row is
	// uncommitted, so a read on the pool would not see it and a read on this
	// transaction would be a second query for values already in hand.
	output = out.NewTeacherOut(teacher)

	// Only teachers is registered for auditing (architecture A1). Education rows
	// are deliberately not: one entry per changed row would break criterion 16's
	// "exactly one" audit entry on any save that touched education.
	//
	// Only the identity of the row is given - audit_helper back-fills Data with
	// the committed row itself after the serve function returns, so the snapshot
	// is the database's version rather than anything this code asserts.
	audit = append(audit, audit_helper.AuditSystemModel{
		TableName:  constanta.TableTeachers,
		PrimaryKey: teacher.ID.Int64,
		Action:     audit_helper.AuditServiceActionInsert,
	})

	return
}
