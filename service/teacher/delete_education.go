package teacher

import (
	"database/sql"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/constanta"
	"sample-project/dto/out"
	errors "sample-project/error"
)

// DeleteEducation is the DELETE /teachers/{id}/educations/{eduId} route.
//
// This is a real delete, and it is the only one in the feature. Criterion 19
// makes a teacher hard-delete unreachable; this table is the opposite case -
// it has no `deleted` column, is deliberately not registered with audit_helper,
// and criterion 27 requires the row to be gone while its siblings and the
// teacher survive.
//
// The route takes no body: the row is fully addressed by its path, so the
// endpoint is registered with NotReadBody() and this method ignores dtoIn
// entirely. That is why there is no optimistic-lock token here - there is
// nothing for a client to send one in, and no field of a deleted row for a
// stale write to corrupt. The scoping predicate is what makes the delete safe,
// and it is evaluated by the same statement that deletes.
//
// The deleted row is returned. It is the only evidence the caller gets that
// they removed the row they meant to, and every other route in this feature
// answers with the row it acted on.
func (s *EducationService) DeleteEducation(
	ctx *context.ContextModel,
	param model.URLParam,
	_ interface{},
) (
	header map[string]string,
	output interface{},
	err error,
) {
	teacherID, err := uuidPath(ctx, param, constanta.PathTeacherID, constanta.FieldID)
	if err != nil {
		return
	}
	educationID, err := uuidPath(ctx, param, constanta.PathEducationID, constanta.FieldEducation)
	if err != nil {
		return
	}

	teacher, err := s.teacherDAO.GetByUUIDKey(ctx.ToContext(), teacherID)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when get teacher by uuid key")
		return
	}
	if !teacher.ID.Valid {
		err = errors.ErrUnknownData(constanta.FieldID).ContextModel(ctx)
		return
	}

	err = s.inTransaction(ctx, func(tx *sql.Tx) error {
		deleted, deleteErr := s.educationDAO.Delete(
			ctx.ToContext(), tx, educationID, teacher.ID.Int64,
		)
		if deleteErr != nil {
			log.Error().Err(deleteErr).Caller().Msg("error when delete teacher education history")
			return deleteErr
		}

		// Zero rows: the key matched no row *for this teacher*. Reported as an
		// unknown education rather than as a forbidden one, so a caller cannot
		// learn that a key exists by getting a different error for it.
		if !deleted.ID.Valid {
			return errors.ErrUnknownData(constanta.FieldEducation).ContextModel(ctx)
		}

		output = out.NewEducationOut(deleted, teacher.UUIDKey.String)

		return nil
	})
	if err != nil {
		return
	}

	return
}
