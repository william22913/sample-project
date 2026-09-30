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

// UpdateEducation is the PUT /teachers/{id}/educations/{eduId} route.
//
// Every field of an education row is editable, start and end dates included -
// unlike the teacher's Hire_Date, which criterion 15 freezes. Criterion 15 is
// specific to `Teacher_ID` and `Hire_Date`; nothing extends it to a study
// period, and the spec calls education rows freely editable.
//
// The row is addressed by its own uuid_key AND scoped by the teacher it belongs
// to. The second half is not decoration: without it, a caller holding a valid
// row key could edit another teacher's history by guessing the path, and
// criterion 28 requires an edit aimed at another teacher's row not to reach it.
func (s *EducationService) UpdateEducation(
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

	educations := []in.EducationIn{*body}
	if err = validateEducationDates(ctx, educations); err != nil {
		return
	}

	levelIDs, err := resolveInstitutionLevels(ctx, s.levelDAO, educations)
	if err != nil {
		return
	}

	model := educationModel(*body, levelIDs[0])

	err = s.inTransaction(ctx, func(tx *sql.Tx) error {
		written, updateErr := s.educationDAO.Update(
			ctx.ToContext(), tx, educationID, teacher.ID.Int64, model,
		)
		if updateErr != nil {
			log.Error().Err(updateErr).Caller().Msg("error when update teacher education history")
			return updateErr
		}

		// Nothing matched. The row does not exist, or it belongs to another
		// teacher - deliberately the same answer, so this route cannot be used
		// to probe for keys the caller has no business seeing. The error names
		// the education rather than the teacher, because the teacher is the
		// part of the path that was correct.
		if !written.ID.Valid {
			return errors.ErrUnknownData(constanta.FieldEducation).ContextModel(ctx)
		}

		output = out.NewEducationOut(written, teacher.UUIDKey.String)

		return nil
	})
	if err != nil {
		return
	}

	return
}
