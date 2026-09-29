package out

import (
	nexconstanta "github.com/nexsoft-git/nexcommon/constanta"

	"sample-project/repository"
)

// EducationOut is one education-history row.
//
// TeacherID is the teacher's uuid_key, matching the `id` every teacher payload
// carries - the internal bigint the FK uses is never exposed. It is passed in
// rather than read from the row: the row holds only the bigint, and every
// caller is already in the context of a teacher whose uuid_key it resolved to
// get there. Joining teachers back on just to re-derive a value the caller has
// in hand would be a round-trip for nothing.
//
// Score is a string. It is stored as NUMERIC(5,2) and carried as text end to
// end (see repository.TeacherEducationHistoryModel), so it leaves as the digits
// Postgres holds - "87.50", not 87.5. A client that reads a score, edits the
// row and writes it back sends the same literal it received.
type EducationOut struct {
	ID               string `json:"id"`
	TeacherID        string `json:"teacher_id"`
	InstitutionLevel string `json:"institution_level"`
	Institution      string `json:"institution"`
	StudyStartDate   string `json:"study_start_date"`
	StudyEndDate     string `json:"study_end_date"`
	Score            string `json:"score"`
	FocusedSubject   string `json:"focused_subject"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

func NewEducationOut(m repository.TeacherEducationHistoryModel, teacherUUIDKey string) EducationOut {
	return EducationOut{
		ID:               m.UUIDKey.String,
		TeacherID:        teacherUUIDKey,
		InstitutionLevel: m.InstitutionLevel.String,
		Institution:      m.Institution.String,
		StudyStartDate:   m.StudyStartDate.Time.Format(nexconstanta.DateOnlyTimeFormat),
		StudyEndDate:     m.StudyEndDate.Time.Format(nexconstanta.DateOnlyTimeFormat),
		Score:            m.Score.String,
		FocusedSubject:   m.FocusedSubject.String,
		CreatedAt:        m.CreatedAt.Time.Format(TimestampLayout),
		UpdatedAt:        m.UpdatedAt.Time.Format(TimestampLayout),
	}
}
