package teacher

import (
	"database/sql"
	"time"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexlogger/log"

	"sample-project/constanta"
	"sample-project/dao"
	in "sample-project/dto/in"
	errors "sample-project/error"
	"sample-project/repository"
)

// resolveInstitutionLevels turns each education row's level *code* (BACHELOR)
// into the institution_levels.id the foreign key needs.
//
// This is the one place the seeded lookup is read (architecture A4). The code
// travels on the wire and the surrogate stays inside - nothing outside this
// function ever sees institution_level_id - which is what keeps the codes
// stable if the table is ever re-seeded with different ids.
//
// The returned ids are positionally aligned with educations: ids[i] belongs to
// educations[i]. That is what lets the caller build rows without a second lookup
// per row.
func resolveInstitutionLevels(
	ctx *context.ContextModel,
	levelDAO dao.InstitutionLevelDAO,
	educations []in.EducationIn,
) (ids []int64, err error) {
	ids = make([]int64, 0, len(educations))

	for _, education := range educations {
		level, lookupErr := levelDAO.GetByName(ctx.ToContext(), education.InstitutionLevel)
		if lookupErr != nil {
			log.Error().Err(lookupErr).Caller().Msg("error when resolve institution level")
			err = lookupErr
			return
		}

		// An unknown code is the caller's mistake, not a server fault, so it
		// travels as a business error naming the field rather than as a 500.
		// The seeded set is fixed (A4), and the DTO deliberately does not
		// duplicate it as an enum - so this lookup is where an unknown code is
		// caught, and it is the only place that needs to know.
		if !level.ID.Valid {
			err = errors.ErrUnknownData(constanta.FieldInstitutionLevel).ContextModel(ctx)
			return
		}

		ids = append(ids, level.ID.Int64)
	}

	return
}

// validateEducationDates applies the spec's two date rules to every row in the
// request before any of them is written:
//
//   - a row's study start must precede its study end
//   - a study period cannot extend past today
//
// Both are checked here rather than left to the database, and the reason is not
// laziness - it is that the database *cannot* check the second one. A Postgres
// CHECK must be immutable and CURRENT_DATE is not, so criterion 29 has no
// constraint behind it. The range rule does have one
// (ck_teachereducationhistories_daterange), but leaving it to fire would turn a
// caller's typo into a generic 500.
//
// Note that this is stricter than the constraint by one case: the CHECK accepts
// start == end, and "must precede" does not. The spec's word wins.
func validateEducationDates(ctx *context.ContextModel, educations []in.EducationIn) error {
	today := todayUTC()

	for _, education := range educations {
		if !education.StudyStartDate.Before(education.StudyEndDate) {
			return errors.ErrInvalidDateRange(
				constanta.FieldStudyStartDate,
				constanta.FieldStudyEndDate,
			).ContextModel(ctx)
		}

		if education.StudyEndDate.After(today) {
			return errors.ErrDateInFuture(constanta.FieldStudyEndDate).ContextModel(ctx)
		}
	}

	return nil
}

// todayUTC is the server's calendar date at UTC midnight.
//
// The comparison is against a date, not an instant, because the DTO's date
// fields arrive that way: the validator parses them with time.Parse, so
// "2026-09-29" becomes 2026-09-29T00:00:00Z whatever the server's timezone is.
// Comparing that against time.Now() instead would be wrong for part of every
// day - a study period ending today would look like it ended in the past once
// the clock passed UTC midnight. Truncating *today* to the same shape compares
// like with like, and takes the date from the server's local clock, which is
// what a user means by today.
func todayUTC() time.Time {
	now := time.Now()

	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

// educationModel builds the row for one request entry. levelID comes from
// resolveInstitutionLevels and is aligned by index with the same slice.
//
// TeacherID is left for the caller: on the create path the teacher does not
// exist yet when this is called.
func educationModel(education in.EducationIn, levelID int64) repository.TeacherEducationHistoryModel {
	return repository.TeacherEducationHistoryModel{
		InstitutionLevelID: sql.NullInt64{Int64: levelID, Valid: true},
		Institution:        sql.NullString{String: education.Institution, Valid: true},
		StudyStartDate:     sql.NullTime{Time: education.StudyStartDate, Valid: true},
		StudyEndDate:       sql.NullTime{Time: education.StudyEndDate, Valid: true},
		Score:              sql.NullString{String: education.Score, Valid: true},
		FocusedSubject:     sql.NullString{String: education.FocusedSubject, Valid: true},
	}
}

// insertEducations writes the request's education rows against a teacher that
// is already inserted, inside the caller's transaction so both halves commit or
// roll back together (criterion 7).
func (s *Service) insertEducations(
	ctx *context.ContextModel,
	tx *sql.Tx,
	teacherID int64,
	educations []in.EducationIn,
	levelIDs []int64,
) error {
	for i, education := range educations {
		model := educationModel(education, levelIDs[i])
		model.TeacherID = sql.NullInt64{Int64: teacherID, Valid: true}

		if _, err := s.educationDAO.Insert(ctx.ToContext(), tx, model); err != nil {
			log.Error().Err(err).Caller().Msg("error when insert teacher education history")
			return err
		}
	}

	return nil
}
