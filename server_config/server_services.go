package server_config

import (
	teacher "sample-project/service/teacher"
)

// InitServices wires the two service values.
//
// Two values, not one with more methods, because nexcommon reads a service's
// request body type from a single GetDTO() with no per-endpoint override - a
// teacher edit and an education edit genuinely have different bodies. See
// service/teacher/type.go.
//
// EducationService is given the *sql.DB rather than the audit helper: its three
// routes are single-row statements that own their own transaction, and
// architecture A1 registers only `teachers` for auditing.
func (s *serverAttribute) InitServices() {
	s.Services = services{
		Teacher: teacher.NewService(
			s.listDAO.TeacherDAO,
			s.listDAO.EducationDAO,
			s.listDAO.InstitutionLevelDAO,
			s.auditHelper,
		),
		TeacherEducation: teacher.NewEducationService(
			s.listDAO.TeacherDAO,
			s.listDAO.EducationDAO,
			s.listDAO.InstitutionLevelDAO,
			s.DBConnection,
		),
	}
}
