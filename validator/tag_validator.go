// Package validator registers the validation rules nexcommon does not ship.
//
// nexcommon's tag_validator covers required/min/max/enum/dateFormat and a fixed
// vocabulary of regexes (email, name, numeric, npwp, nik, fax, ...). Three rules
// this feature needs have no equivalent there, and all three are regexes, so
// they are added through the library's own AddRegex hook rather than as a
// parallel validation pass. That keeps every field rule declared in one place -
// the struct tag - which is what makes the DTOs readable as a specification.
package validator

import (
	"github.com/nexsoft-git/nexcommon/regex"
	"github.com/nexsoft-git/nexcommon/util/validator/tag_validator"

	"sample-project/constanta"
)

// Rule keys. A struct tag cannot reference a constant, so these strings are
// duplicated in the `regex:"..."` tags; TestRuleKeysMatchTags reads the tags
// back by reflection and fails if the two ever disagree.
//
// RulePhone is deliberately the same key the library's validator already
// registers in its own build(). AddRegex overwrites it, so `regex:"phone"`
// anywhere in this project gets the anchored pattern - which is what a reader
// would assume the tag meant, and the alternative is a tag that looks correct
// and quietly is not. The rule name is left as the library's own
// ("PHONE_NUMBER_REGEX"), so the message a caller sees is unchanged.
const (
	RuleAscii = "ascii"
	RulePhone = "phone"
	RuleScore = "score"
)

// Date format keys, same duplication caveat via the `dateFormat:"..."` tags.
const DateFormatTimestamp = "timestamp"

// Enum keys, same duplication caveat via the `enum:"..."` tags.
const EnumTeacherStatus = "teacher_status"

// TimestampLayout is the wire format for updated_at.
//
// It carries microseconds on purpose. updated_at is the optimistic lock
// (architecture A5): the client echoes back what it read, and the conditional
// UPDATE matches on exact equality. A second-precision format would round the
// value the client received away from the value stored - every write would
// look stale - and truncating the column to seconds instead would let two
// updates inside the same second both succeed, which is the one thing the lock
// exists to prevent.
const TimestampLayout = "2006-01-02T15:04:05.000000Z"

// asciiOnly rejects any byte above 0x7F. Criterion 37 applies this to every
// text field, names included, not only to email.
const asciiOnly = `^[\x00-\x7F]*$`

// scoreRange is "0-100 inclusive, at most two decimal places" in one rule.
//
// Applied to the raw JSON literal, so 87.125 is rejected on its digits rather
// than through a float comparison. Note what is not here: a `-`, so a negative
// value fails before it can reach the column's CHECK; and no exponent form, so
// `1e2` is rejected rather than silently accepted as 100.
const scoreRange = `^(100(\.0{1,2})?|\d{1,2}(\.\d{1,2})?)$`

// NewTagValidator returns nexcommon's tag validator with this project's rules
// added. The HTTP controller must be given this instance - the library's own
// NewTagValidator() knows nothing about the three rules above, and a DTO
// carrying regex:"score" validated against it would fail open with an empty
// regex.
func NewTagValidator() tag_validator.TagValidator {
	v := tag_validator.NewTagValidator()

	v.AddRegex(RuleAscii, asciiOnly, "ASCII_ONLY_REGEX")

	// The pattern is the library's own constant (architecture A3), with a
	// leading anchor added - it already ends in `$` but never begins with `^`,
	// so MatchString accepts any string *containing* a match. Without the
	// anchor, "junk+62-81234567890" and, more to the point, a non-ASCII prefix
	// in front of a well-formed number would both pass. This is the smallest
	// change that makes the constant mean what it was written to mean; the
	// pattern itself is untouched.
	v.AddRegex(RulePhone, "^"+regex.PHONE_NUMBER_WITH_COUNTRY_CODE, "PHONE_NUMBER_REGEX")

	v.AddRegex(RuleScore, scoreRange, "SCORE_RANGE_REGEX")

	// INACTIVE is deliberately absent. Only Deactivate produces it (spec,
	// "Employment status semantics"), so a status of INACTIVE on an update is
	// rejected here rather than reaching the service - criterion 21's "no
	// transition out of Inactive" has a second half, that nothing transitions
	// *into* it except deactivation.
	v.AddEnum(EnumTeacherStatus, constanta.StatusActive, constanta.StatusOnLeave)

	v.AddDateFormat(DateFormatTimestamp, TimestampLayout)

	return v
}
