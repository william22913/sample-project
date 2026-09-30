package server_config

import (
	nexendpoint "github.com/nexsoft-git/nexcommon/http/endpoint"

	teacherendpoint "sample-project/endpoint"
)

// InitEndpoint registers every route. Called from main after the router exists
// and before GenerateSwaggerDocs, which reads the metadata these calls
// accumulate.
func (s *serverAttribute) InitEndpoint() {
	endpoints := nexendpoint.NewEndpoint()

	endpoints.AddEndpoint(
		teacherendpoint.NewTeacherEndpoint(
			s.Validator.HttpController,
			s.Services.Teacher,
			s.Services.TeacherEducation,
		),
	)

	endpoints.ServeEndpoint()
}
