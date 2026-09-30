// Package endpoint_test walks the router the feature registers, rather than
// sending requests through it.
//
// A route table is worth testing directly because it is the one place where a
// mistake is silent: a typo in a path, a missing verb, or a placeholder spelled
// differently from the constant the service reads it back with all produce a
// 404 on a route that looks present in the code. There is no compile error and
// no test failure anywhere else - which is exactly how a path-param bug reached
// the service layer before this file existed.
package endpoint_test

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	http_validator "github.com/nexsoft-git/nexcommon/controller/http"
	"github.com/nexsoft-git/nexcommon/util/validator/basic_validator"
	"github.com/nexsoft-git/nexcommon/util/validator/get_list_validator"

	"sample-project/constanta"
	"sample-project/dao"
	teacherendpoint "sample-project/endpoint"
	teacher "sample-project/service/teacher"
	projectvalidator "sample-project/validator"
)

// route is one registered method+path pair.
type route struct {
	method string
	path   string
}

// registeredRoutes builds the same controller and endpoint the composition root
// does, then reads back what the router actually holds.
//
// The DAOs and the audit helper are zero values: registration never calls a
// service method, and substituting a live helper would only add a dependency
// this test does not need.
func registeredRoutes(t *testing.T) map[route]bool {
	t.Helper()

	httpValidator := http_validator.NewHTTPController("fixed-token")
	router := mux.NewRouter()

	httpValidator.
		Router(router).
		BasicValidator(basic_validator.BasicValidator{}).
		TagValidator(projectvalidator.NewTagValidator()).
		ListDataValidator(get_list_validator.NewGetListValidator(basic_validator.BasicValidator{}))

	teacherService := teacher.NewService(
		dao.TeacherDAO{}, dao.EducationDAO{}, dao.InstitutionLevelDAO{}, nil,
	)
	educationService := teacher.NewEducationService(
		dao.TeacherDAO{}, dao.EducationDAO{}, dao.InstitutionLevelDAO{}, nil,
	)

	teacherendpoint.NewTeacherEndpoint(httpValidator, teacherService, educationService).
		RegisterEndpoint()

	found := map[route]bool{}
	err := router.Walk(func(r *mux.Route, _ *mux.Router, _ []*mux.Route) error {
		path, err := r.GetPathTemplate()
		if err != nil {
			return nil // a route with no path template (the router's own root)
		}
		methods, _ := r.GetMethods()
		for _, method := range methods {
			found[route{method: method, path: path}] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking the router: %v", err)
	}

	return found
}

// expectedRoutes is architecture.md's HTTP surface table, verbatim.
func expectedRoutes() []route {
	return []route{
		{http.MethodPost, "/teachers"},
		{http.MethodGet, "/teachers"},
		{http.MethodGet, "/teachers/{id}"},
		{http.MethodPut, "/teachers/{id}"},
		{http.MethodPost, "/teachers/{id}/deactivate"},
		{http.MethodPost, "/teachers/{id}/educations"},
		{http.MethodPut, "/teachers/{id}/educations/{eduId}"},
		{http.MethodDelete, "/teachers/{id}/educations/{eduId}"},
		// Every route also answers OPTIONS - the CORS preflight.
		{http.MethodOptions, "/teachers"},
		{http.MethodOptions, "/teachers/{id}"},
		{http.MethodOptions, "/teachers/{id}/deactivate"},
		{http.MethodOptions, "/teachers/{id}/educations"},
		{http.MethodOptions, "/teachers/{id}/educations/{eduId}"},
	}
}

func TestEveryDocumentedRouteIsRegistered(t *testing.T) {
	found := registeredRoutes(t)

	for _, want := range expectedRoutes() {
		if !found[want] {
			t.Errorf("missing route: %s %s", want.method, want.path)
		}
	}
}

// TestNoTeacherDeleteRouteExists is criterion 19, restated as a route-table
// assertion. Deactivation is a status write on its own endpoint; there is no
// route - not a disabled one, not one returning 405 - that removes a teacher.
func TestNoTeacherDeleteRouteExists(t *testing.T) {
	found := registeredRoutes(t)

	for r := range found {
		if r.method == http.MethodDelete && r.path == "/teachers/{id}" {
			t.Error("criterion 19 violated: DELETE /teachers/{id} is registered. " +
				"Hard delete must be unreachable, and a route is reachable whether or not a caller knows about it.")
		}
		if r.method == http.MethodDelete && r.path == "/teachers" {
			t.Error("criterion 19 violated: DELETE /teachers is registered.")
		}
	}
}

// TestPathPlaceholdersMatchTheConstantsTheServiceReads is the guard for the bug
// this file exists because of.
//
// The controller keys model.URLParam.Path by the mux placeholder's own text,
// and the services look their ids up through constanta.PathTeacherID /
// PathEducationID. Two strings that are not the same word produce no compile
// error and no test failure anywhere else - just an empty lookup and a 404 on
// every route that reads one.
func TestPathPlaceholdersMatchTheConstantsTheServiceReads(t *testing.T) {
	found := registeredRoutes(t)

	want := map[route]string{
		{http.MethodGet, "/teachers/" + "{" + constanta.PathTeacherID + "}"}:             constanta.PathTeacherID,
		{http.MethodPut, "/teachers/" + "{" + constanta.PathTeacherID + "}"}:             constanta.PathTeacherID,
		{http.MethodPost, "/teachers/" + "{" + constanta.PathTeacherID + "}/deactivate"}: constanta.PathTeacherID,
		{http.MethodPost, "/teachers/" + "{" + constanta.PathTeacherID + "}/educations"}: constanta.PathTeacherID,
		{http.MethodDelete, educationPath()}:                                             constanta.PathEducationID,
	}

	for r := range want {
		if !found[r] {
			t.Errorf("no route matches %s %s - the placeholder does not match constanta.Path*", r.method, r.path)
		}
	}

	// The education update route carries both, so it is asserted separately
	// rather than through the single-placeholder map above.
	if !found[route{http.MethodPut, educationPath()}] {
		t.Errorf("no route matches PUT %s", educationPath())
	}
}

func educationPath() string {
	return fmt.Sprintf("/teachers/{%s}/educations/{%s}",
		constanta.PathTeacherID, constanta.PathEducationID)
}

// TestRegisteredRoutesAreSortedForReadability is not a behaviour test. It fails
// when the route set changes, and prints the new set - so a reviewer sees the
// surface the code now exposes rather than having to reconstruct it from a
// diff.
func TestRegisteredRoutesAreSortedForReadability(t *testing.T) {
	found := registeredRoutes(t)

	var got []string
	for r := range found {
		got = append(got, fmt.Sprintf("%s %s", r.method, r.path))
	}
	sort.Strings(got)

	t.Logf("registered routes:\n  %s", strings.Join(got, "\n  "))

	if len(found) != len(expectedRoutes()) {
		t.Errorf("the route table has %d entries, want %d - see the list above",
			len(found), len(expectedRoutes()))
	}
}
