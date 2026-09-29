// Package out holds response payloads and the one mapping from a database row
// to each. The mapping lives here rather than in the services because five
// routes return a teacher and every one of them must agree on how a NULL phone
// or a timestamp is rendered; a second copy is how they stop agreeing.
package out

import (
	nexconstanta "github.com/nexsoft-git/nexcommon/constanta"

	"sample-project/repository"
	"sample-project/validator"
)

// TeacherOut is one teacher record.
//
// Phone is a *string so an absent number renders as JSON null rather than "",
// which is the difference the spec draws when it calls the column nullable.
// Every other field is non-nullable in the schema and renders as a string.
type TeacherOut struct {
	ID          string  `json:"id"`
	TeacherCode string  `json:"teacher_code"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Email       string  `json:"email"`
	Phone       *string `json:"phone"`
	HireDate    string  `json:"hire_date"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// TeacherDetailOut is TeacherOut plus the education history.
//
// Educations is never nil. A teacher with no education rows must render as an
// empty array, not JSON null - criterion 13 makes that a stated behaviour of
// this view, and a client cannot tell "no history" from "field omitted".
type TeacherDetailOut struct {
	TeacherOut
	Educations []EducationOut `json:"educations"`
}

// TimestampLayout is the wire format for every *_at field, and it must be the
// same layout the request DTOs accept for updated_at (validator.TimestampLayout)
// or a client could not echo the value it was given back into an update.
const TimestampLayout = validator.TimestampLayout

// NewTeacherOut maps a row to its response shape. The caller is responsible for
// having checked ID.Valid - a teacher that was not found is a 404 the service
// decides on, not something this mapper can express.
func NewTeacherOut(m repository.TeacherModel) TeacherOut {
	out := TeacherOut{
		ID:          m.UUIDKey.String,
		TeacherCode: m.TeacherCode.String,
		FirstName:   m.FirstName.String,
		LastName:    m.LastName.String,
		Email:       m.Email.String,
		Status:      m.Status.String,
		HireDate:    m.HireDate.Time.Format(nexconstanta.DateOnlyTimeFormat),
		CreatedAt:   m.CreatedAt.Time.Format(TimestampLayout),
		UpdatedAt:   m.UpdatedAt.Time.Format(TimestampLayout),
	}

	if m.Phone.Valid {
		phone := m.Phone.String
		out.Phone = &phone
	}

	return out
}

// NewTeacherDetailOut pairs a teacher with its education rows. `educations` is
// expected to be the non-nil slice the DAO returns; it is normalized here as
// well so a nil from any other source still renders as an empty array.
func NewTeacherDetailOut(m repository.TeacherModel, educations []repository.TeacherEducationHistoryModel) TeacherDetailOut {
	detail := TeacherDetailOut{
		TeacherOut: NewTeacherOut(m),
		Educations: []EducationOut{},
	}

	for _, e := range educations {
		detail.Educations = append(detail.Educations, NewEducationOut(e, m.UUIDKey.String))
	}

	return detail
}
