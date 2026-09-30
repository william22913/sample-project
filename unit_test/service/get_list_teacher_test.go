package service_test

import (
	"testing"

	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"

	"sample-project/constanta"
	"sample-project/dto/out"
)

// listRequest is what the get-list validator would have produced by the time
// the service sees it: page 1 at the default 20 per page, ordered by the
// default (the first entry of GetListValidOrderBy), limit resolved.
func listRequest() in.GetListRequest {
	return in.GetListRequest{Page: 1, Limit: 20, Order: "id ASC"}
}

// statusFilter is the parsed form of `filter=status eq ACTIVE`.
func statusFilter(value string) []model.SearchParam {
	return []model.SearchParam{{
		DataType:       "char",
		SearchKey:      constanta.SearchStatus,
		SearchOperator: constanta.OperatorEqual,
		SearchValue:    value,
	}}
}

// TestGetListActiveFilterIncludesOnLeave is criteria 33 and 34, and it is the
// one piece of the list this service does not take from the framework.
//
// The framework renders a filter as `status = $1`, so a literal reading of
// `status=ACTIVE` would omit every On Leave teacher - who, per 33, keeps access
// and stays in the Active filter. Both halves are asserted here, and they are
// separate claims: the statement must use the `in` operator (not two `=`
// comparisons and not `<>`), and the two values bound to it must be the active
// pair. The bound values alone would also match a query that rendered
// `status = $1 OR status = $2`.
func TestGetListActiveFilterIncludesOnLeave(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`status IN \( \$1 , \$2 \)`).
		WithArgs(constanta.StatusActive, constanta.StatusOnLeave, 20, 0).
		WillReturnRows(teacherRows())

	_, _, err := h.service.GetListTeacher(newContext(), statusFilter(constanta.StatusActive), listRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the Active filter did not reach the driver as an IN over both statuses: %v", err)
	}
}

// TestGetListInactiveFilterIsUnchanged is criterion 34 from the other side: the
// rewrite is specific to the ACTIVE value and must not widen anything else. An
// Inactive filter that quietly grew an extra value would still satisfy the
// active-side tests.
func TestGetListInactiveFilterIsUnchanged(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`FROM teachers WHERE`).
		WithArgs(constanta.StatusInactive, 20, 0).
		WillReturnRows(teacherRows())

	if _, _, err := h.service.GetListTeacher(newContext(), statusFilter(constanta.StatusInactive), listRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the Inactive filter was altered: %v", err)
	}
}

// TestGetListNameSearchIsCaseInsensitiveSubstring is criteria 9 and 10.
//
// The assertion is on the SQL because that is where both halves live: the
// framework lowers the column for `lk` and wraps the value in %...%, and
// neither is visible from the returned rows.
func TestGetListNameSearchIsCaseInsensitiveSubstring(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`LOWER\(first_name\) like \$1`).
		WithArgs("%ada%", 20, 0).
		WillReturnRows(teacherRows())

	searchBy := []model.SearchParam{{
		DataType:       "char",
		SearchKey:      constanta.SearchFirstName,
		SearchOperator: constanta.OperatorLike,
		SearchValue:    "ADA",
	}}

	if _, _, err := h.service.GetListTeacher(newContext(), searchBy, listRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("criteria 9/10: the name search is no longer a lowered substring match: %v", err)
	}
}

// TestGetListTeacherCodeSearchIsExact is criterion 9's other half: Teacher_ID
// is matched exactly, not as a substring, so a partial code finds nothing.
func TestGetListTeacherCodeSearchIsExact(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`teacher_code = \$1`).
		WithArgs("TCH-2024-007", 20, 0).
		WillReturnRows(teacherRows())

	searchBy := []model.SearchParam{{
		DataType:       "char",
		SearchKey:      constanta.SearchTeacherCode,
		SearchOperator: constanta.OperatorEqual,
		SearchValue:    "TCH-2024-007",
	}}

	if _, _, err := h.service.GetListTeacher(newContext(), searchBy, listRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("criterion 9: the Teacher_ID search is no longer an exact match: %v", err)
	}
}

// TestGetListMapsRowsForTheListShape checks the response is the list DTO and
// not the detail one - the list view has no education array to fill.
func TestGetListMapsRowsForTheListShape(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`FROM teachers`).
		WithArgs(20, 0).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	_, payload, err := h.service.GetListTeacher(newContext(), nil, listRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, ok := payload.([]out.TeacherOut)
	if !ok {
		t.Fatalf("payload is %T, want []out.TeacherOut", payload)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	if rows[0].ID != teacherUUID {
		t.Errorf("id: got %q, want the uuid_key", rows[0].ID)
	}
	// NULL phone renders as JSON null, not "".
	if rows[0].Phone != nil {
		t.Errorf("phone: got %v, want nil for a NULL column", *rows[0].Phone)
	}
}

// TestGetListRendersNoMatchesAsAnEmptyArray: a page with nothing on it is a
// normal result, and a nil slice would serialize to JSON null - which a client
// renders as an error rather than as "no matches".
func TestGetListRendersNoMatchesAsAnEmptyArray(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`FROM teachers`).
		WithArgs(20, 0).
		WillReturnRows(teacherRows())

	_, payload, err := h.service.GetListTeacher(newContext(), nil, listRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rows, ok := payload.([]out.TeacherOut)
	if !ok {
		t.Fatalf("payload is %T, want []out.TeacherOut", payload)
	}
	if rows == nil {
		t.Error("got a nil slice, which serializes to JSON null - want an empty array")
	}
}

// TestGetListMetadataIsWhatTheValidatorNeeds covers the four values the
// get-list validator reads before the service is ever called.
//
// The first entry of each is the one that matters: the validator takes
// GetListValidLimit()[0] as the default page size (criterion 12's 20) and
// GetListValidOrderBy()[0] as the default sort, so a reordering of either list
// silently changes the endpoint's defaults rather than failing.
func TestGetListMetadataIsWhatTheValidatorNeeds(t *testing.T) {
	h := newHarness(t)

	t.Run("default page size is 20", func(t *testing.T) {
		limits := h.service.GetListValidLimit()
		if len(limits) == 0 || limits[0] != 20 {
			t.Errorf("criterion 12: the first valid limit is %v, want 20", limits)
		}
	})

	t.Run("default ordering is unique", func(t *testing.T) {
		orderBy := h.service.GetListValidOrderBy()
		if len(orderBy) == 0 || orderBy[0] != "id" {
			t.Errorf("the default ordering is %v, want id first - "+
				"a non-unique default lets page 2 repeat a row page 1 returned", orderBy)
		}
	})

	t.Run("every search key has an operator and vice versa", func(t *testing.T) {
		valid := h.service.GetListValidSearch()
		operators := h.service.GetDefaultOperator()

		if len(valid) != len(operators) {
			t.Fatalf("GetListValidSearch has %d keys, GetDefaultOperator has %d", len(valid), len(operators))
		}
		for _, key := range valid {
			if _, ok := operators[key]; !ok {
				t.Errorf("search key %q is offered with no operator - it would be accepted and then never matched", key)
			}
		}
	})

	t.Run("status is filterable by equality", func(t *testing.T) {
		op, ok := h.service.GetDefaultOperator()[constanta.SearchStatus]
		if !ok {
			t.Fatal("status is not a searchable key, so criteria 11/33/34 have no filter to work on")
		}
		if len(op.Operator) != 1 || op.Operator[0] != constanta.OperatorEqual {
			t.Errorf("status operators: got %v, want exactly [eq] - "+
				"the rewrite only fires on eq, so any other operator would reach the DAO unexpanded", op.Operator)
		}
	})
}

// TestExpandActiveFilterLeavesOtherKeysAlone guards the rewrite against the
// mistake it is most likely to make: rewriting on the value alone, so a
// teacher_code search for the literal string "ACTIVE" would be turned into an
// IN over two statuses.
func TestExpandActiveFilterLeavesOtherKeysAlone(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`teacher_code = \$1`).
		WithArgs(constanta.StatusActive, 20, 0).
		WillReturnRows(teacherRows())

	searchBy := []model.SearchParam{{
		DataType:       "char",
		SearchKey:      constanta.SearchTeacherCode,
		SearchOperator: constanta.OperatorEqual,
		SearchValue:    constanta.StatusActive,
	}}

	if _, _, err := h.service.GetListTeacher(newContext(), searchBy, listRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the rewrite fired on a non-status key: %v", err)
	}
}

// TestGetListPassesPaginationThrough is criterion 12 at the service boundary:
// page 3 of 20 must reach the driver as LIMIT 20 OFFSET 40, not as the page
// number.
func TestGetListPassesPaginationThrough(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`LIMIT \$1 OFFSET \$2`).
		WithArgs(20, 40).
		WillReturnRows(teacherRows())

	request := listRequest()
	request.Page = 3

	if _, _, err := h.service.GetListTeacher(newContext(), nil, request); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("criterion 12: pagination did not reach the driver: %v", err)
	}
}

// TestGetListRejectsAMalformedOrderDirection is the ORDER BY injection guard.
//
// The framework's get-list validator checks the order *column* against
// GetListValidOrderBy() but never the direction token, and nexcommon's DAO
// builder then interpolates that token straight into `ORDER BY`. So
// `?order=id ;--` passes validation and renders
//
//	ORDER BY id ;-- LIMIT $1 OFFSET $2
//
// where `--` comments the pagination out of the statement and one page becomes
// the whole table. Two tokens is under the framework's three-token cap, so
// nothing upstream stops it.
//
// The assertion is that the DAO is never reached, and it has to be phrased as
// an *unmet* expectation rather than "the error is non-nil". With no mock
// expectation set, a DAO call would make sqlmock return an error of its own -
// so `err != nil` is satisfied by the very thing the guard exists to prevent,
// and the test passes with the guard deleted. Setting the expectation and
// requiring it to go unmet is what makes the rejection observable.
func TestGetListRejectsAMalformedOrderDirection(t *testing.T) {
	for _, order := range []string{
		"id ;--",
		"first_name DESC; DROP TABLE teachers",
		"id ASCENDING",
		"last_name asc nulls last",
	} {
		t.Run(order, func(t *testing.T) {
			h := newHarness(t)

			// Deliberately satisfiable: if the service lets this through, the
			// query runs, the expectation is consumed, and the check below fails.
			h.mock.ExpectQuery(`FROM teachers`).
				WithArgs(20, 0).
				WillReturnRows(teacherRows())

			request := listRequest()
			request.Order = order

			if _, _, err := h.service.GetListTeacher(newContext(), nil, request); err == nil {
				t.Fatalf("order %q was accepted; it reaches SQL as text", order)
			}
			if err := h.mock.ExpectationsWereMet(); err == nil {
				t.Errorf("the DAO was reached despite the rejection, so order %q was "+
					"interpolated into the statement", order)
			}
		})
	}
}

// TestGetListAcceptsEveryLegalOrder is the other half - a guard that rejected
// everything would pass the test above and break sorting entirely.
func TestGetListAcceptsEveryLegalOrder(t *testing.T) {
	for _, order := range []string{
		"id ASC",
		"first_name DESC",
		"last_name asc",
		"teacher_code DESC, id ASC",
		"created_at DESC, updated_at ASC",
	} {
		t.Run(order, func(t *testing.T) {
			h := newHarness(t)

			h.mock.ExpectQuery(`FROM teachers`).
				WithArgs(20, 0).
				WillReturnRows(teacherRows())

			request := listRequest()
			request.Order = order

			if _, _, err := h.service.GetListTeacher(newContext(), nil, request); err != nil {
				t.Fatalf("order %q was rejected: %v", order, err)
			}
		})
	}
}

// TestGetListDoesNotWriteAnything is a cheap invariant with a real failure
// mode: the get-list flow shares a query builder with nothing, but it does
// share the connection, and a stray ExpectBegin would go unmet here.
func TestGetListDoesNotWriteAnything(t *testing.T) {
	h := newHarness(t)

	h.mock.ExpectQuery(`FROM teachers`).
		WithArgs(20, 0).
		WillReturnRows(teacherRows().AddRow(teacherValues()...))

	if _, _, err := h.service.GetListTeacher(newContext(), nil, listRequest()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n := h.stream.published(); n != 0 {
		t.Errorf("the list published %d audit entries, want 0 - reads are not audited", n)
	}
	if err := h.mock.ExpectationsWereMet(); err != nil {
		t.Errorf("the list opened a transaction or wrote a row: %v", err)
	}
}
