// Package constanta holds the string values that appear in more than one
// package - DB enum values, request field names - so a typo cannot silently
// create a second spelling of the same concept.
package constanta

// Employment status values. These are the literal strings in the
// ck_teachers_status CHECK constraint (sql_migrations/01_init_schema.sql), so
// changing one here without a migration makes every write fail the check.
const (
	StatusActive   = "ACTIVE"
	StatusOnLeave  = "ON_LEAVE"
	StatusInactive = "INACTIVE"
)

// ActiveFilterValues is what the list endpoint's "Active" filter actually
// matches. ON_LEAVE is deliberately in this set: spec criterion 33 states the
// status has no behavioural effect on filtering, and criterion 11 requires the
// Active/Inactive counts to partition the set. Filtering on status = 'ACTIVE'
// literally would put On Leave teachers in neither bucket and break both.
var ActiveFilterValues = []string{StatusActive, StatusOnLeave}

// Menus are what nexcommon's tag validator matches a field's `required` tag
// against. A field is validated only when one of the values in its `required`
// tag equals the menu its route was registered with - so `required:"insert"`
// means "check this field on the insert route", not "this field must be
// non-empty". Emptiness is a separate concern (the validator's own zero check,
// or `empty:"allowed"` to opt out).
const (
	MenuInsert = "insert"
	MenuUpdate = "update"
	// MenuDeactivate is its own menu rather than reusing MenuUpdate. The
	// deactivate body carries only the optimistic-lock value, so registering it
	// as "update" would demand first_name/last_name/email/phone/status in a
	// request that has no business sending them.
	MenuDeactivate = "deactivate"
	// MenuView and MenuDelete are named even though no DTO field is tagged
	// `required:"view"` or `required:"delete"`, and naming them is the point:
	// the tag validator examines a field only when the route's menu appears in
	// the field's required tag, so a menu nothing lists is what makes a route
	// read no body fields at all. A typo here would silently fall back to
	// "validate nothing", which is why they are constants and not literals.
	MenuView   = "view"
	MenuDelete = "delete"
)

// Request field names, exactly as they appear in the
// `required:"..."`/`max:"..."` struct tags on the DTOs. They exist so a service
// can name the offending field in an error without a string literal, and so the
// bundle key the error converter looks up cannot drift from the tag.
const (
	FieldID               = "ID"
	FieldEmail            = "EMAIL"
	FieldPhone            = "PHONE"
	FieldTeacher          = "TEACHER"
	FieldTeacherCode      = "TEACHER_CODE"
	FieldHireDate         = "HIRE_DATE"
	FieldInstitution      = "INSTITUTION"
	FieldInstitutionLevel = "INSTITUTION_LEVEL"
	FieldStudyStartDate   = "STUDY_START_DATE"
	FieldStudyEndDate     = "STUDY_END_DATE"
	FieldScore            = "SCORE"
	FieldFocusedSubject   = "FOCUSED_SUBJECT"
	FieldEducation        = "EDUCATION"
	FieldUpdatedAt        = "UPDATED_AT"
	// FieldOrderBy is the constanta bundle key for the list's `order` parameter
	// (i18n/common/constanta/*.json -> "ORDER_BY"). Note the framework's own
	// malformed-order rejection passes the bare literal "ORDER", which is not a
	// bundle key and so never resolves to a label; this one does.
	FieldOrderBy = "ORDER_BY"
)

// Table names, as they appear in sql_migrations/01_init_schema.sql and in the
// audit_helper registration. Spelled once so an audit entry's table_name and
// the table it describes cannot disagree.
const (
	TableTeachers = "teachers"
)

// URL path placeholder names, exactly as they appear in the route pattern.
//
// These are keys into model.URLParam.Path, which the controller builds by
// reading the mux's own variable table - so they must match the pattern's
// placeholder text character for character. They are deliberately NOT the
// Field* constants above: those are bundle keys in SCREAMING_CASE because they
// name a field in an error message, and this is a lowercase URL segment. Using
// one for the other yields an empty lookup rather than a compile error, which
// is a silent 404 on every route that trusts it.
const (
	// PathTeacherID is `{id}` in /teachers/{id}. It carries the teacher's
	// uuid_key.
	PathTeacherID = "id"
	// PathEducationID is `{eduId}` in /teachers/{id}/educations/{eduId}. It
	// carries the education row's uuid_key.
	PathEducationID = "eduId"
)

// List search keys and filter operators.
//
// These are the names the list endpoint accepts in its `filter` query
// parameter, and nexcommon's get-list validator both validates them against
// GetListValidSearch() and hands them back as model.SearchParam.SearchKey. The
// DAO then turns a key straight into a column expression, so a key and its
// column must stay the same word - which is why they are spelled here rather
// than in the service.
//
// FieldStatus is the one key the service rewrites rather than passing through;
// see service/teacher/get_list_teacher.go for why.
const (
	SearchFirstName   = "first_name"
	SearchLastName    = "last_name"
	SearchTeacherCode = "teacher_code"
	SearchStatus      = "status"
)

// Filter operators, as nexcommon's get-list validator names them. `lk` is what
// the DAO renders as LOWER(column) LIKE '%value%' - the case-insensitive
// substring match criteria 9 and 10 ask for.
const (
	OperatorEqual = "eq"
	OperatorLike  = "lk"
	OperatorIn    = "in"
)
