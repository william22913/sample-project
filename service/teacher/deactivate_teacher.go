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
)

// deactivateInput is what the transaction needs and the request body cannot
// supply - the same idea as updateInput, minus the body, because deactivation
// writes no field the caller chooses.
type deactivateInput struct {
	id         int64
	expectedAt time.Time
}

// DeactivateTeacher is the POST /teachers/{id}/deactivate route.
//
// A status write, not a delete (criterion 18): the row and every column but
// status/updated_* survive, and the record stays readable afterwards. There is
// no DELETE /teachers/{id} route at all (criterion 19).
//
// It carries the optimistic lock for the same reason the update route does. A
// deactivation built on a stale read is exactly as wrong as a stale edit -
// leaving it unchecked would be a second, weaker door into the same row - and
// criterion 36 does not exempt it.
func (s *Service) DeactivateTeacher(
	ctx *context.ContextModel,
	param model.URLParam,
	dtoIn interface{},
) (
	header map[string]string,
	output interface{},
	err error,
) {
	body := dtoIn.(*in.TeacherIn)

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

	// Criterion 21, read this way round on purpose: deactivating an already
	// inactive teacher is a terminal-state rejection, not a stale write, so it
	// is reported as one even when the caller's updated_at is also out of date.
	// The more specific reason is the more useful one.
	//
	// The DAO's `status <> 'INACTIVE'` predicate covers the same case again. The
	// difference is what a failure means: here it is known to be terminal, there
	// it could be either, so only here can the honest message be sent.
	if current.Status.String == constanta.StatusInactive {
		err = errors.ErrInactiveTeacher().ContextModel(ctx)
		return
	}

	if !current.UpdatedAt.Time.Equal(body.UpdatedAt) {
		err = errors.ErrDataLocked().ContextModel(ctx)
		return
	}

	audit, err := s.auditHelper.InitAuditService(
		ctx,
		deactivateInput{id: current.ID.Int64, expectedAt: body.UpdatedAt},
		s.doDeactivateTeacher,
	)
	if err != nil {
		return
	}

	output, err = audit.CompletedAuditData()

	return
}

// doDeactivateTeacher runs inside the audit builder's transaction.
func (s *Service) doDeactivateTeacher(
	ctx *context.ContextModel,
	tx *sql.Tx,
	dtoIn interface{},
	_ time.Time,
) (
	output interface{},
	audit []audit_helper.AuditSystemModel,
	err error,
) {
	input := dtoIn.(deactivateInput)

	// Audited as an update, because at the row level that is what happened:
	// criterion 18 keeps the row and criterion 19 removes every path that could
	// make this a delete. The before-snapshot is read first, so the entry
	// records the status the teacher held and the one they were moved to.
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

	teacher, err := s.teacherDAO.Deactivate(ctx.ToContext(), tx, input.id, actorID(ctx), input.expectedAt)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when deactivate teacher")
		return
	}

	if !teacher.ID.Valid {
		err = errors.ErrDataLocked().ContextModel(ctx)
		return
	}

	output = out.NewTeacherOut(teacher)

	return
}
