package server_config

import (
	"sample-project/dao"
)

// InitDAO builds one DAO per table. They hold nothing but the connection, so
// there is no ordering among them and no error to report.
func (s *serverAttribute) InitDAO() {
	s.listDAO = listDAO{
		TeacherDAO:          dao.NewTeacherDAO(s.DBConnection),
		EducationDAO:        dao.NewEducationDAO(s.DBConnection),
		InstitutionLevelDAO: dao.NewInstitutionLevelDAO(s.DBConnection),
	}
}
