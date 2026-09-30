// Package endpoint registers the feature's routes with nexcommon's HTTP
// controller.
//
// Everything a route needs beyond its path and handler is a fluent modifier on
// the wrapper, and the ones used here are all load-bearing:
//
//   - WhitelistValidator on every route - architecture A2. There is no
//     authentication in this phase, so the validator is the one that always
//     passes; the controller still runs the rest of the pipeline (DTO decode,
//     tag validation, response formatting).
//   - PathParams names the mux placeholders verbatim ("id", "eduId"), which is
//     what the controller copies out of the router's variable table into
//     model.URLParam.Path. The service reads those same keys through
//     constanta.PathTeacherID/PathEducationID.
//   - Menu is the switch nexcommon's tag validator matches a field's
//     `required:"..."` tag against. It decides which fields of the one shared
//     body DTO are examined on this route.
//   - DTOOut is what the generated swagger documents as the response, and it
//     is wrapped in out.DefaultResponse by the controller.
//
// There is no route for DELETE /teachers/{id} and there will not be one:
// criterion 19 requires hard delete to be unreachable.
package endpoint

import (
	"net/http"

	http_validator "github.com/nexsoft-git/nexcommon/controller/http"

	"sample-project/constanta"
	"sample-project/dto/out"
	teacher "sample-project/service/teacher"
)

// tagTeacher is the swagger grouping for every route in this feature.
const tagTeacher = "Teacher"

type teacherEndpoint struct {
	httpValidator *http_validator.HTTPController
	srv           *teacher.Service
	educationSrv  *teacher.EducationService
}

func NewTeacherEndpoint(
	httpValidator *http_validator.HTTPController,
	srv *teacher.Service,
	educationSrv *teacher.EducationService,
) *teacherEndpoint {
	return &teacherEndpoint{
		httpValidator: httpValidator,
		srv:           srv,
		educationSrv:  educationSrv,
	}
}

func (e *teacherEndpoint) RegisterEndpoint() {
	e.registerTeacherRoutes()
	e.registerEducationRoutes()
}

func (e *teacherEndpoint) registerTeacherRoutes() {
	// POST /teachers - create, with optional embedded education rows. One
	// transaction, so the teacher and its rows commit together (criterion 7).
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.InsertTeacher,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuInsert).
					DTOOut(http_validator.NewDTOParam(out.TeacherOut{})).
					Tags(tagTeacher).Description("Create teacher"),
			),
			http.MethodPost, http.MethodOptions,
		),
	)

	// GET /teachers - list, search, filter, paginate (criteria 9-12).
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers",
			e.httpValidator.WrapServiceListData(
				http_validator.NewWarpGetListServiceParam(
					e.srv,
					e.srv.GetListTeacher,
					e.httpValidator.WhitelistValidator(),
				).DTOOut(http_validator.NewDTOParam([]out.TeacherOut{})).
					Tags(tagTeacher).Description("List teachers"),
			),
			http.MethodGet, http.MethodOptions,
		),
	)

	// GET /teachers/{id} - detail, including the education array (criterion 13
	// makes an empty array a deliberate state, not an error).
	//
	// MenuView is not a no-op even though no field is tagged `required:"view"`:
	// the tag validator only examines a field when the route's menu is one of
	// the values in its required tag, so an unlisted menu is what keeps the
	// detail route from demanding a request body's worth of fields it has no
	// body for.
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.GetDetailTeacher,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuView).
					PathParams(constanta.PathTeacherID).
					DTOOut(http_validator.NewDTOParam(out.TeacherDetailOut{})).
					Tags(tagTeacher).Description("Teacher detail"),
			),
			http.MethodGet, http.MethodOptions,
		),
	)

	// PUT /teachers/{id} - update, guarded by the updated_at optimistic lock
	// (criteria 35/36).
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.UpdateTeacher,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuUpdate).
					PathParams(constanta.PathTeacherID).
					DTOOut(http_validator.NewDTOParam(out.TeacherOut{})).
					Tags(tagTeacher).Description("Update teacher"),
			),
			http.MethodPut, http.MethodOptions,
		),
	)

	// POST /teachers/{id}/deactivate - its own menu, and so its own DTO field
	// set: the body carries only updated_at, not the contact details an update
	// would send.
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}/deactivate",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.srv,
					e.srv.DeactivateTeacher,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuDeactivate).
					PathParams(constanta.PathTeacherID).
					DTOOut(http_validator.NewDTOParam(out.TeacherOut{})).
					Tags(tagTeacher).Description("Deactivate teacher"),
			),
			http.MethodPost, http.MethodOptions,
		),
	)
}

func (e *teacherEndpoint) registerEducationRoutes() {
	// POST /teachers/{id}/educations - a sub-resource rather than a field of
	// the teacher update, so criterion 27 ("deleting a row removes that row
	// only") is a row-scoped statement and not a whole-teacher replace that
	// could clobber siblings by omission.
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}/educations",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.educationSrv,
					e.educationSrv.AddEducation,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuInsert).
					PathParams(constanta.PathTeacherID).
					DTOOut(http_validator.NewDTOParam(out.EducationOut{})).
					Tags(tagTeacher).Description("Add teacher education"),
			),
			http.MethodPost, http.MethodOptions,
		),
	)

	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}/educations/{eduId}",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.educationSrv,
					e.educationSrv.UpdateEducation,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuUpdate).
					PathParams(constanta.PathTeacherID, constanta.PathEducationID).
					DTOOut(http_validator.NewDTOParam(out.EducationOut{})).
					Tags(tagTeacher).Description("Edit teacher education"),
			),
			http.MethodPut, http.MethodOptions,
		),
	)

	// NotReadBody: this is the one route with no request body at all. Without
	// it the controller would try to decode an empty body into EducationIn and
	// fail with ErrReadBody before the handler ever ran, so the service is
	// handed a nil dtoIn - which is what its signature already expects.
	e.httpValidator.HandleFunc(
		http_validator.NewHandleFuncParam(
			"/teachers/{id}/educations/{eduId}",
			e.httpValidator.WrapService(
				http_validator.NewWarpServiceParam(
					e.educationSrv,
					e.educationSrv.DeleteEducation,
					e.httpValidator.WhitelistValidator(),
				).Menu(constanta.MenuDelete).
					PathParams(constanta.PathTeacherID, constanta.PathEducationID).
					NotReadBody().
					DTOOut(http_validator.NewDTOParam(out.EducationOut{})).
					Tags(tagTeacher).Description("Remove teacher education"),
			),
			http.MethodDelete, http.MethodOptions,
		),
	)
}
