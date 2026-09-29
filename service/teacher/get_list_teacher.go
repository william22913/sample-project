package teacher

import (
	"strings"

	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services"

	"sample-project/constanta"
	"sample-project/dto/out"
	errors "sample-project/error"
	"sample-project/repository"
)

// The list endpoint's metadata. nexcommon's get-list validator reads all four
// off the service before it touches the request, and each answers a different
// question:
//
//   - GetListValidLimit - which page sizes are legal. The validator clamps an
//     absent or too-small limit up to the first entry, so 20 first is criterion
//     12's default rather than a coincidence.
//   - GetListValidOrderBy - which columns may be sorted on, and what to sort by
//     when the caller asks for nothing. The validator takes [0] as that
//     default, so `id` leads: it is unique, which is what keeps page 2 from
//     repeating a row that page 1 already returned when two teachers share a
//     name.
//   - GetListValidSearch - the filter keys. Built from the operator map below
//     so the two cannot drift.
//   - GetDefaultOperator - which operator each key accepts, and its data type.
func (s *Service) GetListValidLimit() []int { return []int{20, 50, 100} }

func (s *Service) GetListValidOrderBy() []string {
	return []string{"id", "first_name", "last_name", "teacher_code", "created_at", "updated_at"}
}

func (s *Service) GetListValidSearch() []string { return s.validSearchBy }

func (s *Service) GetDefaultOperator() services.DefaultOperators { return s.operator }

// GetListTeacher is criterion 9: search by Name and by Teacher_ID, plus the
// status filter of criteria 11/33/34 and the pagination of criterion 12.
//
// Nearly all of it is nexcommon's - the request is parsed and validated before
// this is called, and the DAO hands the filters to the shared query builder.
// The one thing this method adds is expandActiveFilter, because that is the one
// place the framework's default reading of the request is not this feature's.
func (s *Service) GetListTeacher(
	ctx *context.ContextModel,
	searchParam []model.SearchParam,
	dto in.GetListRequest,
) (
	header map[string]string,
	output interface{},
	err error,
) {
	if err = validateOrder(dto); err != nil {
		return
	}

	dbResult, err := s.teacherDAO.GetListTeacher(ctx, dto, expandActiveFilter(searchParam))
	if err != nil {
		return
	}

	// Never nil: a page with no rows is an empty array, matching the detail
	// view's treatment of an empty education list. A nil slice marshals to JSON
	// null, which a client renders as an error rather than as "no matches".
	result := []out.TeacherOut{}
	for i := range dbResult {
		repo, ok := dbResult[i].(repository.TeacherModel)
		if !ok {
			// Unreachable while parseTeacherRow is the only parser this query
			// uses. Skipping rather than panicking keeps a mismatched parser a
			// missing row instead of a 500 with no response body (the
			// controller's recover does not write one).
			continue
		}
		result = append(result, out.NewTeacherOut(repo))
	}

	output = result

	return
}

// validateOrder closes the one hole the framework's get-list validator leaves
// open, and it is a real one: the `order` parameter's sort *direction* reaches
// SQL as text.
//
// The validator does validate the column - it splits each comma-separated entry
// on spaces, rejects anything longer than two tokens, and checks token[0]
// against GetListValidOrderBy(). It never looks at token[1]. The DAO builder
// then does fmt.Sprintf("%s %s", field, tokens[len-1]) straight into
// `ORDER BY`. So `?order=id ;--` passes validation (two tokens) and renders
// `ORDER BY id ;-- LIMIT $1 OFFSET $2`, where the `--` comments the pagination
// out of the statement.
//
// It is bounded - the statement is a SELECT, and the framework's two-token cap
// means a second statement cannot be smuggled in - but it is unparameterized
// caller input in a statement, and its effect is to turn one page into the whole
// table. The framework will not close it, so it is closed here, at the boundary
// this service owns.
//
// Rejecting rather than coercing: a direction that is neither ASC nor DESC is a
// malformed request, and answering it with a silent ASC would hide that. The
// error is nexcommon's own malformed-field code, so the caller sees the same
// 400 it would get for a bad column name.
//
// Note the two-token cap is also why `DESC NULLS LAST` never arrives: the
// validator already rejects it. ASC and DESC are the whole legal set.
func validateOrder(dto in.GetListRequest) error {
	if dto.Order == "" {
		// The validator replaces an empty order with the first entry of
		// GetListValidOrderBy, so this is unreachable through the controller.
		// Checked anyway because it is the value the DAO would interpolate.
		return nil
	}

	for _, entry := range strings.Split(dto.Order, ",") {
		tokens := strings.Fields(entry)

		if len(tokens) == 0 || len(tokens) > 2 {
			return errors.ErrFormatField(constanta.FieldOrderBy)
		}
		if len(tokens) == 2 && !isSortDirection(tokens[1]) {
			return errors.ErrFormatField(constanta.FieldOrderBy)
		}
	}

	return nil
}

func isSortDirection(token string) bool {
	return strings.EqualFold(token, "ASC") || strings.EqualFold(token, "DESC")
}

// expandActiveFilter rewrites the caller's `status` filter so that asking for
// Active includes teachers who are On Leave.
//
// This is the one request reading this feature does NOT take from the
// framework, and it is not cosmetic. The DAO renders a filter as
// `status = $1`, and the validator rejects anything outside the enum, so
// `status=ACTIVE` would literally mean the ACTIVE rows. Criteria 11 and 34 need
// it to mean {ACTIVE, ON_LEAVE}: On Leave teachers keep access and stay in the
// Active filter (33), and the Active/Inactive counts have to partition the set
// with nothing falling between them (11). INACTIVE is not in this set, so
// `status=INACTIVE` still matches only the deactivated rows and 34 holds.
//
// Rewriting the parsed SearchParam rather than the SQL keeps the whole thing
// inside the framework's `in` operator, which is already parameterized - the
// values become $n placeholders like any other filter, so nothing here is
// string-concatenated into a statement.
//
// The rewrite is in place, on the caller's slice. That is safe because the
// controller hands this method a freshly parsed slice per request and keeps no
// reference to it.
func expandActiveFilter(searchParam []model.SearchParam) []model.SearchParam {
	for i := range searchParam {
		param := &searchParam[i]

		if param.SearchKey != constanta.SearchStatus ||
			param.SearchOperator != constanta.OperatorEqual ||
			param.SearchValue != constanta.StatusActive {
			continue
		}

		param.SearchOperator = constanta.OperatorIn
		param.SearchValue = []interface{}{constanta.StatusActive, constanta.StatusOnLeave}
	}

	return searchParam
}

// listOperators is the filter vocabulary. Built once in NewService so
// GetDefaultOperator and GetListValidSearch are two views of one map - the
// second is literally the first's key set, so a key can never be offered
// without an operator, or accepted without being offered.
//
// The operators are per the architecture's criterion mapping: `lk` on the name
// fields for criteria 9/10's case-insensitive substring search, `eq` on
// teacher_code (C9) and on status (the filter the expandActiveFilter above
// rewrites).
//
// DataType "char" on every key, including status. It is not the enum
// DataType - that one makes the builder wrap the column in a CAST, which is
// for integer-backed enum columns, and status is a VARCHAR already.
func listOperators() services.DefaultOperators {
	operator := make(services.DefaultOperators)

	operator[constanta.SearchFirstName] = model.DefaultOperator{
		DataType: "char",
		Operator: []string{constanta.OperatorLike, constanta.OperatorEqual},
	}
	operator[constanta.SearchLastName] = model.DefaultOperator{
		DataType: "char",
		Operator: []string{constanta.OperatorLike, constanta.OperatorEqual},
	}
	operator[constanta.SearchTeacherCode] = model.DefaultOperator{
		DataType: "char",
		Operator: []string{constanta.OperatorEqual},
	}
	operator[constanta.SearchStatus] = model.DefaultOperator{
		DataType: "char",
		Operator: []string{constanta.OperatorEqual},
	}

	return operator
}
