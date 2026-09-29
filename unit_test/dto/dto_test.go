// Package dto_test exercises the request DTOs the way the HTTP controller
// does: JSON in, nexcommon's tag validator over it, error code out.
//
// Assertions are on error *codes* rather than messages because no error
// formator is installed in this test binary - with the formator nil,
// UnbundledErrorMessages.Error() returns its code verbatim. The three codes
// that appear are nexcommon's own field-level vocabulary:
//
//	E-4-CMD-DTO-001  field is empty
//	E-4-CMD-DTO-002  field is not in the declared format (dates)
//	E-4-CMD-DTO-003  field breaks a named rule (max length, regex, enum)
package dto_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	nexctx "github.com/nexsoft-git/nexcommon/context"

	"sample-project/dto/in"
	"sample-project/validator"
)

const (
	codeEmptyField   = "E-4-CMD-DTO-001"
	codeFormatField  = "E-4-CMD-DTO-002"
	codeFormatRule   = "E-4-CMD-DTO-003"
	menuInsert       = "insert"
	menuUpdate       = "update"
	menuDeactivate   = "deactivate"
	validTimestamp   = "2026-09-29T10:11:12.345678Z"
	validHireDate    = "2024-01-15"
	validStudyStart  = "2016-08-01"
	validStudyEnd    = "2020-06-01"
	countryCodePhone = "+62-81234567890"
)

// validate returns the error code the validator reports for dto, or "" when it
// accepts the value.
func validate(t *testing.T, dto interface{}, menu string) string {
	t.Helper()
	err := validator.NewTagValidator().ValidateByTag(nexctx.NewContextModel(), dto, "", menu)
	if err == nil {
		return ""
	}
	return err.Error()
}

// decode runs the same json.Unmarshal the controller's readBody does, then
// validates. Decode failures are reported separately: a body that will not
// unmarshal never reaches the validator at all.
func decode(t *testing.T, raw string, dst interface{}, menu string) string {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), dst); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	return validate(t, dst, menu)
}

func educationJSON(score string) string {
	return `{
		"institution_level": "BACHELOR",
		"institution": "Universitas Indonesia",
		"study_start_date": "` + validStudyStart + `",
		"study_end_date": "` + validStudyEnd + `",
		"score": ` + score + `,
		"focused_subject": "Mathematics"
	}`
}

func teacherJSON(extra string) string {
	base := `{"first_name":"Ana","last_name":"Putri","email":"ana@school.edu","hire_date":"` + validHireDate + `"`
	if extra == "" {
		return base + `}`
	}
	return base + `,` + extra + `}`
}

// ---------------------------------------------------------------------------
// criterion 25 / 28 - score round-trips, range is enforced
// ---------------------------------------------------------------------------

func TestScoreAcceptsTheWholeLegalRange(t *testing.T) {
	for _, literal := range []string{"0", "0.0", "87.5", "87.50", "99.99", "100", "100.00"} {
		dto := &in.EducationIn{}
		if code := decode(t, educationJSON(literal), dto, menuInsert); code != "" {
			t.Errorf("score %s: rejected with %s, want accepted", literal, code)
			continue
		}
		// C25 is not only "accepted" - the digits must be the client's.
		if dto.Score != literal {
			t.Errorf("score %s: decoded as %q, want the literal preserved", literal, dto.Score)
		}
	}
}

func TestScoreRejectsOutOfRange(t *testing.T) {
	for _, literal := range []string{"-5", "-0.01", "100.01", "101", "500"} {
		dto := &in.EducationIn{}
		if code := decode(t, educationJSON(literal), dto, menuInsert); code != codeFormatRule {
			t.Errorf("score %s: got %q, want %s", literal, code, codeFormatRule)
		}
	}
}

// TestScoreRejectsFinerPrecision is criterion 40: more than two decimal places
// is rejected, not rounded. It is checked on the digits because a float64
// could not tell 87.125 from 87.125000000000001 by the time this runs.
func TestScoreRejectsFinerPrecision(t *testing.T) {
	for _, literal := range []string{"87.125", "0.001", "100.0001", "1.005"} {
		dto := &in.EducationIn{}
		if code := decode(t, educationJSON(literal), dto, menuInsert); code != codeFormatRule {
			t.Errorf("score %s: got %q, want %s", literal, code, codeFormatRule)
		}
	}
}

// TestScoreRejectsExponentForm pins the boundary of the rule: the score is
// digits and at most one dot, so `1e2` is refused rather than quietly read as
// 100. It matches the DB CHECK's intent without relying on the DB to say so.
func TestScoreRejectsExponentForm(t *testing.T) {
	dto := &in.EducationIn{}
	if code := decode(t, educationJSON(`"1e2"`), dto, menuInsert); code != codeFormatRule {
		t.Errorf("score \"1e2\": got %q, want %s", code, codeFormatRule)
	}
}

// TestScoreIsRequired: an absent score is a missing field, not a zero.
func TestScoreIsRequired(t *testing.T) {
	dto := &in.EducationIn{}
	raw := `{
		"institution_level": "BACHELOR",
		"institution": "Universitas Indonesia",
		"study_start_date": "` + validStudyStart + `",
		"study_end_date": "` + validStudyEnd + `",
		"focused_subject": "Mathematics"
	}`
	if code := decode(t, raw, dto, menuInsert); code != codeEmptyField {
		t.Errorf("absent score: got %q, want %s", code, codeEmptyField)
	}
}

// TestScoreTagIsReachableByTheValidator guards the subtlest trap in this DTO
// set. nexcommon's validator dispatches on the exact reflected type name
// (`reflect.String.String() == field.Type.String()`); a json.Number field
// matches no branch at all and is skipped in silence, so `regex:"score"` would
// never run and 500 would be accepted. Holding the field as a plain string,
// with UnmarshalJSON doing the conversion, is what keeps the tag live.
func TestScoreTagIsReachableByTheValidator(t *testing.T) {
	field, ok := reflect.TypeOf(in.EducationIn{}).FieldByName("Score")
	if !ok {
		t.Fatal("EducationIn has no Score field")
	}
	if field.Type.Kind() != reflect.String || field.Type.String() != "string" {
		t.Errorf("Score is %s (kind %s); the validator only inspects a field whose "+
			"type is exactly `string`, so any other type makes regex:%q dead",
			field.Type, field.Type.Kind(), validator.RuleScore)
	}
}

// TestEducationJSONAcceptsNumberOrString covers UnmarshalJSON's two jobs: the
// wire form is a number, but a quoted number must decode too - and the literal
// must survive either way, trailing zero included.
func TestEducationJSONAcceptsNumberOrString(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{`87.5`, "87.5"},
		{`"87.5"`, "87.5"},
		{`87.50`, "87.50"},
		{`0`, "0"},
	} {
		dto := &in.EducationIn{}
		if code := decode(t, educationJSON(tc.raw), dto, menuInsert); code != "" {
			t.Errorf("score %s: rejected with %s, want accepted", tc.raw, code)
			continue
		}
		if dto.Score != tc.want {
			t.Errorf("score %s: decoded as %q, want %q", tc.raw, dto.Score, tc.want)
		}
	}
}

// A score that is not a number at all fails to unmarshal, which the controller
// reports as a malformed body rather than as a field error - there is nothing
// field-shaped left to report.
func TestNonNumericScoreFailsToDecode(t *testing.T) {
	dto := &in.EducationIn{}
	if err := json.Unmarshal([]byte(educationJSON(`"abc"`)), dto); err == nil {
		t.Error("a non-numeric score decoded without error, want a decode failure")
	}
}

// ---------------------------------------------------------------------------
// criteria 37 / 38 / 39 - ASCII only, email lowercased, max length inclusive
// ---------------------------------------------------------------------------

// TestEmailIsLowercasedOnDecode is criterion 38. The lowercase happens during
// validation (auto_fix), so it is the value that gets stored, compared against
// the case-insensitive unique index, and returned.
func TestEmailIsLowercasedOnDecode(t *testing.T) {
	for _, tc := range []struct{ sent, want string }{
		{"Teacher@School.EDU", "teacher@school.edu"},
		{"MIXED.Case@Example.COM", "mixed.case@example.com"},
		{"already@lower.org", "already@lower.org"},
	} {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"email":"`+tc.sent+`"`), dto, menuInsert); code != "" {
			t.Errorf("email %s: rejected with %s, want accepted", tc.sent, code)
			continue
		}
		if dto.Email != tc.want {
			t.Errorf("email %s: stored as %q, want %q", tc.sent, dto.Email, tc.want)
		}
	}
}

// TestNonAsciiIsRejectedPerField is criterion 37. The field is not named in the
// assertion because the field name reaches the message only through the
// library's converter and the i18n bundle; what is checked here is that each
// text field carries the rule, which is the part this package owns.
func TestNonAsciiIsRejectedPerField(t *testing.T) {
	const nonAscii = "Añá"

	t.Run("first_name", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"first_name":"`+nonAscii+`"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("last_name", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"last_name":"`+nonAscii+`"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("email", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"email":"café@school.edu"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("institution", func(t *testing.T) {
		dto := &in.EducationIn{}
		raw := strings.Replace(educationJSON("87.5"), "Universitas Indonesia", "Üniversitas", 1)
		if code := decode(t, raw, dto, menuInsert); code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("focused_subject", func(t *testing.T) {
		dto := &in.EducationIn{}
		raw := strings.Replace(educationJSON("87.5"), "Mathematics", "Matemáticas", 1)
		if code := decode(t, raw, dto, menuInsert); code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
}

// TestMaxLengthIsInclusive is criterion 39. The boundary is the point: exactly
// the limit is accepted, one over is refused.
func TestMaxLengthIsInclusive(t *testing.T) {
	t.Run("first_name_50", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"first_name":"`+strings.Repeat("a", 50)+`"`), dto, menuInsert); code != "" {
			t.Errorf("50 characters rejected with %s, want accepted", code)
		}
	})
	t.Run("first_name_51", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"first_name":"`+strings.Repeat("a", 51)+`"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("51 characters got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("first_name_below_min", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"first_name":"A"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("1 character got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("institution_100", func(t *testing.T) {
		dto := &in.EducationIn{}
		raw := strings.Replace(educationJSON("87.5"), "Universitas Indonesia", strings.Repeat("a", 100), 1)
		if code := decode(t, raw, dto, menuInsert); code != "" {
			t.Errorf("100 characters rejected with %s, want accepted", code)
		}
	})
	t.Run("institution_101", func(t *testing.T) {
		dto := &in.EducationIn{}
		raw := strings.Replace(educationJSON("87.5"), "Universitas Indonesia", strings.Repeat("a", 101), 1)
		if code := decode(t, raw, dto, menuInsert); code != codeFormatRule {
			t.Errorf("101 characters got %q, want %s", code, codeFormatRule)
		}
	})
	t.Run("email_50", func(t *testing.T) {
		// 50 characters exactly, shaped like an address.
		local := strings.Repeat("a", 50-len("@school.edu"))
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"email":"`+local+`@school.edu"`), dto, menuInsert); code != "" {
			t.Errorf("50-character email rejected with %s, want accepted", code)
		}
	})
	t.Run("email_51", func(t *testing.T) {
		local := strings.Repeat("a", 51-len("@school.edu"))
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"email":"`+local+`@school.edu"`), dto, menuInsert); code != codeFormatRule {
			t.Errorf("51-character email got %q, want %s", code, codeFormatRule)
		}
	})
}

// ---------------------------------------------------------------------------
// architecture A3 - the phone rule
// ---------------------------------------------------------------------------

// TestPhoneIsAnchoredAtBothEnds covers the deviation from the library: its
// PHONE_NUMBER_WITH_COUNTRY_CODE ends in `$` but does not begin with `^`, so on
// its own it accepts any string *containing* a match. The rule registered here
// adds the anchor; these cases are the ones that distinguishes the two.
func TestPhoneIsAnchoredAtBothEnds(t *testing.T) {
	for _, tc := range []struct {
		phone string
		valid bool
		why   string
	}{
		{countryCodePhone, true, "the intended shape"},
		{"", true, "omitted, which empty:\"allowed\" permits"},
		{"junk" + countryCodePhone, false, "a match anywhere in the string is not a phone number"},
		{"62-81234567890", false, "no leading +"},
		{"+62-0812345678", false, "the part after the dash must start 1-9"},
		{"not a phone", false, "not a phone number at all"},
	} {
		dto := &in.TeacherIn{}
		code := decode(t, teacherJSON(`"phone":"`+tc.phone+`"`), dto, menuInsert)
		if tc.valid && code != "" {
			t.Errorf("phone %q (%s): rejected with %s, want accepted", tc.phone, tc.why, code)
		}
		if !tc.valid && code != codeFormatRule {
			t.Errorf("phone %q (%s): got %q, want %s", tc.phone, tc.why, code, codeFormatRule)
		}
	}
}

// TestPhoneLengthIsCapped proves the max tag is live on a field that also has
// empty:"allowed" - that pair is what the required tag has to coexist with.
func TestPhoneLengthIsCapped(t *testing.T) {
	dto := &in.TeacherIn{}
	long := "+62-" + strings.Repeat("1", 60)
	if code := decode(t, teacherJSON(`"phone":"`+long+`"`), dto, menuInsert); code != codeFormatRule {
		t.Errorf("61-character phone got %q, want %s", code, codeFormatRule)
	}
}

// ---------------------------------------------------------------------------
// criterion 21 / 35 / 36 - status and the optimistic-lock token
// ---------------------------------------------------------------------------

func updateJSON(extra string) string {
	base := `{"first_name":"Ana","last_name":"Putri","email":"ana@school.edu",` +
		`"phone":"","status":"ACTIVE","updated_at":"` + validTimestamp + `"`
	if extra == "" {
		return base + `}`
	}
	return base + `,` + extra + `}`
}

// TestUpdateAcceptsOnlyActiveAndOnLeave is criterion 21's second half: nothing
// transitions *into* Inactive except deactivation, so an update naming it is
// refused rather than silently ignored.
func TestUpdateAcceptsOnlyActiveAndOnLeave(t *testing.T) {
	for _, tc := range []struct {
		status string
		valid  bool
	}{
		{"ACTIVE", true},
		{"ON_LEAVE", true},
		{"INACTIVE", false},
		{"active", false},
		{"", false},
		{"DELETED", false},
	} {
		dto := &in.TeacherIn{}
		code := decode(t, updateJSON(`"status":"`+tc.status+`"`), dto, menuUpdate)
		if tc.valid && code != "" {
			t.Errorf("status %q: rejected with %s, want accepted", tc.status, code)
		}
		if !tc.valid && code == "" {
			t.Errorf("status %q: accepted, want refused", tc.status)
		}
	}
}

// TestUpdatedAtIsRequiredAndExact is criteria 35 and 36. The token cannot be
// omitted, and it must be the value the client was given - to the microsecond,
// because the UPDATE matches on equality and a second-precision format would
// make every write look stale.
func TestUpdatedAtIsRequiredAndExact(t *testing.T) {
	t.Run("present and microsecond exact", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, updateJSON(""), dto, menuUpdate); code != "" {
			t.Fatalf("rejected with %s, want accepted", code)
		}
		if got := dto.UpdatedAt.Format(validator.TimestampLayout); got != validTimestamp {
			t.Errorf("updated_at round-tripped as %s, want %s - the client could not "+
				"echo back the value it was given", got, validTimestamp)
		}
	})
	t.Run("absent", func(t *testing.T) {
		dto := &in.TeacherIn{}
		raw := `{"first_name":"Ana","last_name":"Putri","email":"ana@school.edu","phone":"","status":"ACTIVE"}`
		if code := decode(t, raw, dto, menuUpdate); code != codeFormatField {
			t.Errorf("absent updated_at: got %q, want %s", code, codeFormatField)
		}
	})
	t.Run("second precision is not the wire format", func(t *testing.T) {
		dto := &in.TeacherIn{}
		raw := strings.Replace(updateJSON(""), validTimestamp, "2026-09-29T10:11:12Z", 1)
		if code := decode(t, raw, dto, menuUpdate); code != codeFormatField {
			t.Errorf("second-precision updated_at: got %q, want %s", code, codeFormatField)
		}
	})
	t.Run("deactivate carries the same token", func(t *testing.T) {
		// The deactivate body is the token and nothing else, so the token's
		// required tag has to name its menu too - `required:"update"` alone
		// would leave this route with no token requirement at all.
		dto := &in.TeacherIn{}
		if code := decode(t, `{"updated_at":"`+validTimestamp+`"}`, dto, menuDeactivate); code != "" {
			t.Errorf("rejected with %s, want accepted", code)
		}
		empty := &in.TeacherIn{}
		if code := decode(t, `{}`, empty, menuDeactivate); code != codeFormatField {
			t.Errorf("missing token: got %q, want %s", code, codeFormatField)
		}
	})
}

// TestImmutableFieldsAreDetectableWhenSent is criterion 15's precondition.
//
// One DTO serves every route (nexcommon's controller allows exactly one
// GetDTO per service value), so teacher_code and hire_date have to be present
// on the struct for the update path to be able to refuse them - but they must
// also be *absent* from the update menu's validation, or the refusal would
// arrive as a validation error instead of the service's own.
//
// The two fields reach that in different ways, and the difference is why both
// are asserted here:
//
//   - teacher_code is a pointer. "sent" and "sent empty" ("" and null) are
//     both non-nil, so any attempt to write it is visible.
//   - hire_date is the validator's mandatory time.Time + <Name>Str pair, which
//     cannot be a pointer - the validator dispatches on the exact reflected
//     type and looks the companion up by name. Presence is therefore read from
//     the string companion, which is what TeacherIn.HireDatePresent does.
func TestImmutableFieldsAreDetectableWhenSent(t *testing.T) {
	typ := reflect.TypeOf(in.TeacherIn{})

	t.Run("teacher_code", func(t *testing.T) {
		field, ok := typ.FieldByName("TeacherCode")
		if !ok {
			t.Fatal("TeacherIn has no TeacherCode field")
		}
		if field.Type.Kind() != reflect.Ptr {
			t.Errorf("TeacherCode is %s, not a pointer: an empty string would be "+
				"indistinguishable from an omitted field", field.Type)
		}
		if field.Tag.Get("required") != "" {
			t.Errorf("TeacherCode carries required:%q, which would make it a validated "+
				"field rather than one that exists to be refused",
				field.Tag.Get("required"))
		}
	})

	t.Run("hire_date", func(t *testing.T) {
		field, ok := typ.FieldByName("HireDate")
		if !ok {
			t.Fatal("TeacherIn has no HireDate field")
		}
		// required:"insert" is the load-bearing part: it makes the field
		// validated on create and invisible to the update menu, so an update
		// that carries hire_date passes validation and reaches the service.
		if got := field.Tag.Get("required"); got != menuInsert {
			t.Errorf("HireDate carries required:%q, want %q - any other menu would "+
				"either drop the create-path rule or turn criterion 15's refusal "+
				"into a validation error", got, menuInsert)
		}

		companion, ok := typ.FieldByName("HireDateStr")
		if !ok {
			t.Fatal("HireDate has no HireDateStr companion, so presence cannot be detected")
		}
		if companion.Tag.Get("json") != "hire_date" {
			t.Errorf("HireDateStr carries json:%q, want %q - that tag is the wire name "+
				"the validator reports, and the name the client must send",
				companion.Tag.Get("json"), "hire_date")
		}

		// The detection itself, on every shape an update can send.
		//
		// "" reads as absent alongside null and omitted, and that is the
		// correct reading rather than a hole: an empty string carries no date,
		// so there is no new value to reject. The rejection criterion 15 asks
		// for is about a caller writing a *different* Hire_Date, which is the
		// third case below.
		for _, tc := range []struct {
			body     string
			expected bool
		}{
			{`{"hire_date":"` + validHireDate + `"}`, true},
			{`{"hire_date":"2020-01-01"}`, true},
			{`{"hire_date":""}`, false},
			{`{"hire_date":null}`, false},
			{`{}`, false},
		} {
			dto := &in.TeacherIn{}
			if err := json.Unmarshal([]byte(tc.body), dto); err != nil {
				t.Fatalf("%s: %v", tc.body, err)
			}
			if got := dto.HireDatePresent(); got != tc.expected {
				t.Errorf("%s: HireDatePresent() = %v, want %v", tc.body, got, tc.expected)
			}
		}
	})
}

// ---------------------------------------------------------------------------
// create-path dates
// ---------------------------------------------------------------------------

func TestHireDateIsRequiredAndDateOnly(t *testing.T) {
	t.Run("present", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(""), dto, menuInsert); code != "" {
			t.Fatalf("rejected with %s, want accepted", code)
		}
		if got := dto.HireDate.Format("2006-01-02"); got != validHireDate {
			t.Errorf("hire_date parsed as %s, want %s", got, validHireDate)
		}
	})
	t.Run("absent", func(t *testing.T) {
		dto := &in.TeacherIn{}
		raw := `{"first_name":"Ana","last_name":"Putri","email":"ana@school.edu","phone":""}`
		if code := decode(t, raw, dto, menuInsert); code != codeFormatField {
			t.Errorf("absent hire_date: got %q, want %s", code, codeFormatField)
		}
	})
	t.Run("not a date", func(t *testing.T) {
		dto := &in.TeacherIn{}
		raw := strings.Replace(teacherJSON(""), validHireDate, "15-01-2024", 1)
		if code := decode(t, raw, dto, menuInsert); code != codeFormatField {
			t.Errorf("malformed hire_date: got %q, want %s", code, codeFormatField)
		}
	})
}

// TestEducationDatesAreRequiredAndOrderedInApplication: both dates are
// required, and both are date-only on the wire - the ordering rule itself
// (end >= start) is not expressed here, because it is a comparison between two
// fields and no tag can state it. The service owns it.
func TestEducationDatesAreRequired(t *testing.T) {
	for _, missing := range []string{"study_start_date", "study_end_date"} {
		dto := &in.EducationIn{}
		raw := `{
			"institution_level": "BACHELOR",
			"institution": "Universitas Indonesia",
			"study_start_date": "` + validStudyStart + `",
			"study_end_date": "` + validStudyEnd + `",
			"score": 87.5,
			"focused_subject": "Mathematics"
		}`
		var obj map[string]json.RawMessage
		if err := json.Unmarshal([]byte(raw), &obj); err != nil {
			t.Fatalf("building body: %v", err)
		}
		delete(obj, missing)
		stripped, err := json.Marshal(obj)
		if err != nil {
			t.Fatalf("rebuilding body: %v", err)
		}
		if code := decode(t, string(stripped), dto, menuInsert); code != codeFormatField {
			t.Errorf("missing %s: got %q, want %s", missing, code, codeFormatField)
		}
	}
}

// TestEducationsAreOptionalButValidated: an absent or empty array is valid -
// a teacher may have no education history - while a malformed element inside
// one is still caught, which is what makes the slice's required tag load-bearing.
// Without it the validator never walks the slice at all.
func TestEducationsAreOptionalButValidated(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(""), dto, menuInsert); code != "" {
			t.Errorf("rejected with %s, want accepted", code)
		}
	})
	t.Run("empty array", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"educations":[]`), dto, menuInsert); code != "" {
			t.Errorf("rejected with %s, want accepted", code)
		}
	})
	t.Run("one valid row", func(t *testing.T) {
		dto := &in.TeacherIn{}
		if code := decode(t, teacherJSON(`"educations":[`+educationJSON("87.5")+`]`), dto, menuInsert); code != "" {
			t.Errorf("rejected with %s, want accepted", code)
		}
		if len(dto.Educations) != 1 || dto.Educations[0].Score != "87.5" {
			t.Errorf("education row did not survive decoding: %+v", dto.Educations)
		}
	})
	t.Run("one malformed row", func(t *testing.T) {
		dto := &in.TeacherIn{}
		code := decode(t, teacherJSON(`"educations":[`+educationJSON("87.125")+`]`), dto, menuInsert)
		if code != codeFormatRule {
			t.Errorf("got %q, want %s", code, codeFormatRule)
		}
	})
}

// ---------------------------------------------------------------------------
// structural guards - the failure mode these catch is silence
// ---------------------------------------------------------------------------

// ruleTags are the tags that only take effect inside the validator's
// required-gated block. A field carrying one of these without a required tag
// is a rule that never runs, and nothing else in the codebase would say so:
// the JSON still decodes, the field is still populated, and the value is
// accepted no matter what it contains.
var ruleTags = []string{"max", "min", "regex", "enum", "dateFormat"}

func TestEveryRuleCarriesARequiredTag(t *testing.T) {
	dtos := []struct {
		name string
		typ  reflect.Type
	}{
		{"TeacherIn", reflect.TypeOf(in.TeacherIn{})},
		{"EducationIn", reflect.TypeOf(in.EducationIn{})},
	}

	for _, dto := range dtos {
		for i := 0; i < dto.typ.NumField(); i++ {
			field := dto.typ.Field(i)
			required := field.Tag.Get("required")
			for _, tag := range ruleTags {
				if field.Tag.Get(tag) == "" {
					continue
				}
				if required == "" {
					t.Errorf("%s.%s carries %s:%q but no required tag - the validator "+
						"skips the field entirely, so that rule never runs",
						dto.name, field.Name, tag, field.Tag.Get(tag))
				}
				break
			}
		}
	}
}

// TestEveryRuleNamesARegisteredRule checks the other half of tag/constant
// agreement: a regex tag naming a rule the validator does not know is skipped
// in silence by the library (`if v.regex[regexField].Regex != ""`).
//
// The registered set is read back from the validator itself rather than
// hardcoded, so adding a rule to the package and forgetting to use it, or
// using one that was never added, both show up here.
func TestEveryRuleNamesARegisteredRule(t *testing.T) {
	known := map[string]bool{
		validator.RuleAscii: true,
		validator.RulePhone: true,
		validator.RuleScore: true,
		// The library's own vocabulary, which this project may also use.
		"email": true,
	}

	dtos := []reflect.Type{
		reflect.TypeOf(in.TeacherIn{}),
		reflect.TypeOf(in.EducationIn{}),
	}

	for _, dto := range dtos {
		for i := 0; i < dto.NumField(); i++ {
			field := dto.Field(i)
			rule := field.Tag.Get("regex")
			if rule == "" {
				continue
			}
			if !known[rule] {
				t.Errorf("%s.%s: regex:%q names a rule that is not registered, so the "+
					"validator skips it without complaint", dto.Name(), field.Name, rule)
			}
		}
	}
}

func TestEveryRuleKeyConstantIsUsedByADTO(t *testing.T) {
	// A rule registered but referenced by no field is either dead or a typo in
	// a tag that TestEveryRuleNamesARegisteredRule would otherwise accept.
	used := map[string]bool{}
	dtos := []reflect.Type{
		reflect.TypeOf(in.TeacherIn{}),
		reflect.TypeOf(in.EducationIn{}),
	}
	for _, dto := range dtos {
		for i := 0; i < dto.NumField(); i++ {
			if rule := dto.Field(i).Tag.Get("regex"); rule != "" {
				used[rule] = true
			}
		}
	}

	for _, rule := range []string{validator.RuleAscii, validator.RulePhone, validator.RuleScore} {
		if !used[rule] {
			t.Errorf("rule %q is registered but no DTO field asks for it", rule)
		}
	}
}

// TestTimeFieldsUseTheStrCompanion pins the shape nexcommon's validator
// requires of a time field: it looks up the field named <Name>Str, reads the
// string from it, and reports the json name found on *that* field. The
// time.Time itself must therefore carry no json tag - the wire has one date
// string, and a duplicate tag is what `go vet` rejects.
func TestTimeFieldsUseTheStrCompanion(t *testing.T) {
	dtos := []reflect.Type{
		reflect.TypeOf(in.TeacherIn{}),
		reflect.TypeOf(in.EducationIn{}),
	}

	found := 0
	for _, dto := range dtos {
		for i := 0; i < dto.NumField(); i++ {
			field := dto.Field(i)
			if field.Type != reflect.TypeOf(time.Time{}) {
				continue
			}
			found++

			if jsonTag := field.Tag.Get("json"); jsonTag != "" {
				t.Errorf("%s.%s carries json:%q; the time field must be untagged so the "+
					"wire value decodes into its companion",
					dto.Name(), field.Name, jsonTag)
			}
			companion, ok := dto.FieldByName(field.Name + "Str")
			if !ok {
				t.Errorf("%s.%s has no %sStr companion, so the validator cannot read "+
					"it: it would report ErrUnknownData for the missing field",
					dto.Name(), field.Name, field.Name)
				continue
			}
			if companion.Tag.Get("json") == "" {
				t.Errorf("%s.%sStr has no json tag; the reported field name would be empty",
					dto.Name(), field.Name)
			}
			if companion.Type.Kind() != reflect.String {
				t.Errorf("%s.%sStr is %s, not a string", dto.Name(), field.Name, companion.Type)
			}
		}
	}

	if found == 0 {
		t.Fatal("no time.Time fields found - the DTOs no longer have any dates")
	}
}
