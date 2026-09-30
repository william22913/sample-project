package teacher

import (
	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/constanta"
	"sample-project/dto/out"
	errors "sample-project/error"
)

// GetDetailTeacher is the GET /teachers/{id} route - the only read of a single
// teacher, and the only place the education array is returned.
//
// `{id}` is the row's uuid_key, not its internal bigint and not teacher_code.
// The bigint never leaves the service boundary; teacher_code is a business
// identifier the trigger assigns and a later feature may let a client search
// by it, which is a different thing from addressing a resource.
func (s *Service) GetDetailTeacher(
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

	teacher, err := s.teacherDAO.GetByUUIDKey(ctx.ToContext(), teacherID)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when get teacher by uuid key")
		return
	}

	if !teacher.ID.Valid {
		err = errors.ErrUnknownData(constanta.FieldID).ContextModel(ctx)
		return
	}

	// No audit helper and no transaction: this is a read. Nothing here goes
	// through audit_helper, which is why the read path in a nexcommon service
	// has no transaction at all.
	educations, err := s.educationDAO.GetByTeacherID(ctx.ToContext(), teacher.ID.Int64)
	if err != nil {
		log.Error().Err(err).Caller().Msg("error when get teacher education histories")
		return
	}

	// A teacher with no education rows reaches this line with an empty, non-nil
	// slice, and renders as `"educations": []` rather than null - criterion 13's
	// deliberate empty state, which a client cannot distinguish from an omitted
	// field. The DAO and the mapper both normalize it; this is the route that
	// makes the difference observable, so it is asserted in the unit tests.
	output = out.NewTeacherDetailOut(teacher, educations)

	return
}
