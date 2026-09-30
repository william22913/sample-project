// Package error holds this feature's business errors - the ones a caller is
// meant to read and act on. Anything else (a DB failure, a marshal failure)
// travels as a raw error and nexcommon's formator turns it into a generic 500,
// which is correct: there is nothing actionable to say about it.
//
// Shape note, because it differs from nexlegal's error/error.go deliberately.
// There the sentinels are package-level vars mutated in place by Param(...)
// before being returned - which is how nexcommon's own error/message.go does it
// too. That is a data race with an observable symptom: two requests failing on
// different fields at the same time both mutate the same struct, and the
// message one caller receives can carry the other's field name. Every error
// here is constructed per call instead, so no state is shared. The codes and
// the wire shape are identical - a client cannot tell the difference.
package error

import (
	"errors"

	error2 "github.com/nexsoft-git/nexcommon/error"
)

// Project business error codes: E-4-<PROJECT>-SRV-NNN, the same shape nexlegal
// uses for its own (E-4-NL-SRV-NNN). E-4-CMD-* belongs to nexcommon's shared
// vocabulary, with the one documented exception below.
const (
	CodeDataUsed         = "E-4-TCH-SRV-001"
	CodeImmutableField   = "E-4-TCH-SRV-002"
	CodeInactiveTeacher  = "E-4-TCH-SRV-003"
	CodeDateInFuture     = "E-4-TCH-SRV-004"
	CodeInvalidDateRange = "E-4-TCH-SRV-005"

	// CodeDataLocked, CodeUnknownData and CodeFormatField are nexcommon's own
	// codes, and they are reproduced here rather than taken from the library's
	// error.ErrDataLocked / error.ErrUnknownData / error.ErrFormatField vars for
	// the reason in the package note above - the library's are globals mutated
	// per request. The code strings are unchanged, so the client sees exactly
	// what it would have seen from the library.
	CodeDataLocked  = "E-4-CMD-DTO-007"
	CodeUnknownData = "E-4-CMD-DTO-004"
	CodeFormatField = "E-4-CMD-DTO-002"
)

func newError(status int, code string, converter error2.Converter) *error2.UnbundledErrorMessages {
	return error2.NewUnBundledErrorMessages(status, errors.New(code), converter)
}

// ErrDataUsed - the value is already taken. Criterion 3: a duplicate email is
// a field-level error naming Email, and zero rows are written.
//
// fieldName is a constanta field-name key (constanta.FieldEmail), not display
// text: the converter marks it IsConverted, so the formator looks the label up
// in the common.constanta bundle and the message follows the request language.
func ErrDataUsed(fieldName string) *error2.UnbundledErrorMessages {
	return newError(400, CodeDataUsed, errDataNameConverter).Param(fieldName)
}

// ErrImmutableField - criterion 15. An attempt to edit Teacher_ID or Hire_Date
// is rejected outright rather than silently ignored, so a caller cannot come
// away believing a rename succeeded.
func ErrImmutableField(fieldName string) *error2.UnbundledErrorMessages {
	return newError(400, CodeImmutableField, errDataNameConverter).Param(fieldName)
}

// ErrInactiveTeacher - criterion 21. Inactive is terminal: nothing transitions
// out of it, and nothing transitions into it except deactivation. One error
// covers both halves, because from the caller's side they are the same
// rejection - the stored status did not change and will not.
//
// No parameter, so no converter: the message makes no reference to a field.
func ErrInactiveTeacher() *error2.UnbundledErrorMessages {
	return newError(400, CodeInactiveTeacher, nil)
}

// ErrDateInFuture - criterion 29, a study period cannot extend past today.
// Application-layer only: a Postgres CHECK cannot reference CURRENT_DATE
// because it is not immutable, so this rule has no database backstop.
func ErrDateInFuture(fieldName string) *error2.UnbundledErrorMessages {
	return newError(400, CodeDateInFuture, errDataNameConverter).Param(fieldName)
}

// ErrInvalidDateRange - the spec's "a row's study start must precede its study
// end". Carried as an explicit rejection rather than left to the
// ck_teachereducationhistories_daterange CHECK, which would surface as a
// generic 500 on input a client can plausibly send.
func ErrInvalidDateRange(startField, endField string) *error2.UnbundledErrorMessages {
	return newError(400, CodeInvalidDateRange, errDateRangeConverter).Param(startField, endField)
}

// ErrDataLocked - criterion 35/36. The write was built on a stale read: the
// stored updated_at no longer matches the one the client presented, so nothing
// was written and the caller must re-fetch before retrying.
//
// The current record is deliberately not attached. nexcommon's formator builds
// a fixed error envelope (status, code, message) from a typed sentinel and has
// no slot for a payload, so carrying the row would mean inventing a response
// shape outside the library. The caller re-reads instead.
func ErrDataLocked() *error2.UnbundledErrorMessages {
	return newError(400, CodeDataLocked, nil)
}

// ErrUnknownData - there is no row with the given value. Used for a teacher id
// that resolves to nothing, and for an institution_level code outside the
// seeded set.
func ErrUnknownData(fieldName string) *error2.UnbundledErrorMessages {
	return newError(400, CodeUnknownData, errFieldNameConverter).Param(fieldName)
}

// ErrFormatField - a request parameter is malformed. Same code and message
// shape nexcommon's own ErrFormatField produces, so a caller cannot tell this
// rejection from one the framework made.
//
// fieldName is a constanta bundle key, not display text - see
// errFieldNameConverter.
func ErrFormatField(fieldName string) *error2.UnbundledErrorMessages {
	return newError(400, CodeFormatField, errFieldNameConverter).Param(fieldName)
}

// errDataNameConverter maps the first argument to the {{.DataName}} placeholder,
// converted through the common.constanta bundle so a key becomes a label.
var errDataNameConverter = func(value ...interface{}) map[string]error2.ErrorParam {
	return map[string]error2.ErrorParam{
		"DataName": {Param: value[0], IsConverted: true},
	}
}

// errFieldNameConverter is the {{.FieldName}} equivalent. The placeholder name
// is not a style choice: it is what nexcommon's own E-4-CMD-DTO-004 and -007
// messages interpolate, and those are the codes reproduced above.
var errFieldNameConverter = func(value ...interface{}) map[string]error2.ErrorParam {
	return map[string]error2.ErrorParam{
		"FieldName": {Param: value[0], IsConverted: true},
	}
}

var errDateRangeConverter = func(value ...interface{}) map[string]error2.ErrorParam {
	return map[string]error2.ErrorParam{
		"StartDate": {Param: value[0], IsConverted: true},
		"EndDate":   {Param: value[1], IsConverted: true},
	}
}
