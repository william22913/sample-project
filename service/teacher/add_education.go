package teacher

import (
	"database/sql"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/constanta"
	in "sample-project/dto/in"
	"sample-project/dto/out"
	errors "sample-project/error"
)

// AddEducation is the POST /teachers/{id}/educations route: one new education
// row against an existing teacher.
//
// There is no audit entry, because `teachers` is the only table registered for
// auditing (architecture A1) - an education row is not an audited entity. That
// is also why the teacher's updated_at deliberately does not move: a save that
// changed no teacher field must leave the lock value alone (criterion 17), and
// bumping it here would invalidate every client's copy of the record for a
// change that did not touch the record.
//
// No Inactive check either. Criterion 21 states its rule for the teacher's own
// status transition and names the Update path as where it is enforced; the
// spec describes education rows as "freely editable and deletable after
// creation". Adding a gate here would be inventing a rule the feature docs do
// not state.
func (s *EducationService) AddEducation(
	ctx *context.ContextModel,
	param model.URLParam,
	dtoIn interface{},
) (
	header map[string]string,
	output interface{},
	err error,
) {
	body := dtoIn.(*in.EducationIn)

	teacherID, err := uuidPath(ctx, param, constanta.PathTeacherID, constanta.FieldID)
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

	// Both rules run before the transaction opens: a request that cannot be
	// written must not be able to write anything, and there is nothing to roll
	// back if the failure precedes the work.
	educations := []in.EducationIn{*body}
	if err = validateEducationDates(ctx, educations); err != nil {
		return
	}

	levelIDs, err := resolveInstitutionLevels(ctx, s.levelDAO, educations)
	if err != nil {
		return
	}

	model := educationModel(*body, levelIDs[0])
	model.TeacherID = sql.NullInt64{Int64: teacher.ID.Int64, Valid: true}

	err = s.inTransaction(ctx, func(tx *sql.Tx) error {
		written, insertErr := s.educationDAO.Insert(ctx.ToContext(), tx, model)
		if insertErr != nil {
			log.Error().Err(insertErr).Caller().Msg("error when insert teacher education history")
			return insertErr
		}

		// Built from the row the database wrote, not from the request. Score is
		// the case that makes this matter: NUMERIC(5,2) turns an input of
		// "87.5" into a stored "87.50", and echoing the request would hand the
		// client back a value that disagrees with what a later read returns.
		output = out.NewEducationOut(written, teacher.UUIDKey.String)

		return nil
	})
	if err != nil {
		return
	}

	return
}
