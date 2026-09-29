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

// updateInput is what the transaction needs and the request body cannot supply:
// the row's internal id, resolved from the uuid_key before the transaction
// opened.
//
// It is passed to InitAuditService as the opaque inputStruct, which nexcommon
// hands straight back to the serve function. The body is carried along rather
// than re-asserted from the path a second time.
type updateInput struct {
	id   int64
	body *in.TeacherIn
}

// UpdateTeacher is the PUT /teachers/{id} route.
//
// The order of the checks below is the order of the criteria they serve, and it
// is not arbitrary - each one is cheaper and more specific than the next, and
// every one of them refuses the whole request rather than applying part of it.
func (s *Service) UpdateTeacher(
	ctx *context.ContextModel,
	param model.URLParam,
	dtoIn interface{},
) (
	header map[string]string,
	output interface{},
	err error,
) {
	body := dtoIn.(*in.TeacherIn)

	// Criterion 15, and the reason these two fields exist on the DTO at all:
	// an attempt to edit Teacher_ID or Hire_Date is an explicit rejection, not
	// a silently-dropped field. A caller who renames a teacher and is told the
	// save succeeded has been lied to.
	//
	// Checked before anything is read, because the answer does not depend on
	// the row - the request is invalid whatever the stored state is.
	if body.TeacherCode != nil {
		err = errors.ErrImmutableField(constanta.FieldTeacherCode).ContextModel(ctx)
		return
	}
	if body.HireDatePresent() {
		err = errors.ErrImmutableField(constanta.FieldHireDate).ContextModel(ctx)
		return
	}

	teacherID, err := uuidPath(ctx, param, constanta.PathTeacherID, constanta.FieldID)
	if err != nil {
		return
	}

	current, err := s.teacherDAO.GetByUUIDKey(ctx.ToContext(), teacherID)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when get teacher by uuid key")
		return
	}
	if !current.ID.Valid {
		err = errors.ErrUnknownData(constanta.FieldID).ContextModel(ctx)
		return
	}

	// Criterion 21. Inactive is terminal, so no update reaches an inactive row -
	// including one that would leave the status alone, which is what makes this
	// a property of the state rather than of the value being written. The only
	// reason a caller can see an inactive teacher is to read it.
	if current.Status.String == constanta.StatusInactive {
		err = errors.ErrInactiveTeacher().ContextModel(ctx)
		return
	}

	// Criterion 35/36, the optimistic lock (architecture A5).
	//
	// Compared here as well as in the UPDATE's WHERE clause, and the duplication
	// is deliberate: this read is what lets the two failure modes be told apart.
	// The conditional UPDATE can only say "nothing matched", which is also what
	// a row deleted a moment ago produces; this says "someone else has written
	// since you read". The UPDATE remains the guard that actually holds, because
	// another writer can commit between this line and the transaction opening.
	//
	// Equal, not After: a client that echoes a *newer* value than the stored one
	// is as stale as one that echoes an older value - both mean its copy is not
	// the row's current state, and only exact equality proves it read the row it
	// thinks it is editing.
	if !current.UpdatedAt.Time.Equal(body.UpdatedAt) {
		err = errors.ErrDataLocked().ContextModel(ctx)
		return
	}

	// Criterion 17: updated_at changes only when a field actually changed, so a
	// save that changes nothing must write nothing. Returning the row unchanged
	// is not an empty response - it is the record as it stands, with the
	// updated_at the client already has, so its next save still works.
	//
	// No transaction is opened on this path and no audit entry is written: a
	// no-op save is not a successful update, and criterion 16 counts entries per
	// successful update.
	if !teacherChanged(current, body) {
		output = out.NewTeacherOut(current)
		return
	}

	audit, err := s.auditHelper.InitAuditService(ctx, updateInput{id: current.ID.Int64, body: body}, s.doUpdateTeacher)
	if err != nil {
		return
	}

	output, err = audit.CompletedAuditData()

	return
}

// doUpdateTeacher runs inside the audit builder's transaction.
func (s *Service) doUpdateTeacher(
	ctx *context.ContextModel,
	tx *sql.Tx,
	dtoIn interface{},
	_ time.Time,
) (
	output interface{},
	audit []audit_helper.AuditSystemModel,
	err error,
) {
	input := dtoIn.(updateInput)
	body := input.body

	// Re-checked, not re-used. The check in UpdateTeacher ran on the pool before
	// this transaction existed, so between the two another request can take the
	// address. excludeID is this row, so re-saving a teacher without touching
	// their email is not a duplicate of itself.
	duplicate, err := s.teacherDAO.CheckDuplicateEmail(ctx.ToContext(), tx, body.Email, input.id)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when check duplicate teacher email")
		return
	}
	if duplicate.Int64 > 0 {
		err = errors.ErrDataUsed(constanta.FieldEmail).ContextModel(ctx)
		return
	}

	// The before-snapshot, and it has to be read here: before the UPDATE, and
	// inside this transaction.
	//
	// audit_helper does this by itself for an insert and only for an insert -
	// there the row is brand new, so reading it after the write is the same
	// thing. An update is not symmetric, and the library has no way to read the
	// row it is about to overwrite. Calling this explicitly is the house
	// pattern.
	//
	// What it captures is the row as it *was*. The after-image is not recorded,
	// and cannot be: AuditSystemModel carries a single Data field
	// (audit_helper/type.go), so there is no second slot to put the post-update
	// values in. Criterion 16 is satisfied in that exactly one entry is appended
	// per successful update; it is not satisfied in the sense of a before/after
	// pair, and a reader wanting the new values reads the row or the next entry.
	//
	// Data is left empty on purpose - this call fills it from the row.
	audit, err = audit_helper.GetDataForAuditByIDTx(
		ctx,
		tx,
		audit_helper.AuditServiceActionUpdate,
		constanta.TableTeachers,
		input.id,
		ctx.Limitation.UserID,
		s.auditHelper.GetDefaultSchema(),
	)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when get teacher data for audit")
		return
	}

	teacher, err := s.teacherDAO.Update(ctx.ToContext(), tx, input.id, repository.TeacherModel{
		FirstName: sql.NullString{String: body.FirstName, Valid: true},
		LastName:  sql.NullString{String: body.LastName, Valid: true},
		Email:     sql.NullString{String: body.Email, Valid: true},
		Phone:     sql.NullString{String: body.Phone, Valid: body.Phone != ""},
		Status:    sql.NullString{String: body.Status, Valid: true},
		UpdatedBy: actorID(ctx),
	}, body.UpdatedAt)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when update teacher")
		return
	}

	// Nothing matched. The row was there when this transaction opened and its
	// updated_at matched then, so the only thing that can have happened is
	// another writer committing in between - which is exactly what the lock
	// exists to catch.
	if !teacher.ID.Valid {
		err = errors.ErrDataLocked().ContextModel(ctx)
		return
	}

	output = out.NewTeacherOut(teacher)

	return
}

// teacherChanged reports whether the request would alter any writable column.
//
// It covers exactly the set the UPDATE's SET clause writes. A column that
// neither this function nor the UPDATE mentions - teacher_code, hire_date,
// deleted - cannot make a save "changed", which is what keeps criterion 17 and
// the statement in step.
//
// The comparison is on the values as stored and as validated, and both sides
// have already been through the same normalisation: email is lowercased by the
// DTO's auto_fix before it reaches here, so a client re-sending
// "Teacher@School.EDU" against a stored "teacher@school.edu" is correctly a
// no-op rather than a write.
func teacherChanged(current repository.TeacherModel, body *in.TeacherIn) bool {
	return current.FirstName.String != body.FirstName ||
		current.LastName.String != body.LastName ||
		current.Email.String != body.Email ||
		current.Phone.String != body.Phone ||
		current.Status.String != body.Status
}
