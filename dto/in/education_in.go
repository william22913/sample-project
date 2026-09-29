// Package in holds request bodies. Every field rule lives in these struct tags
// and nowhere else - nexcommon's tag validator reads them by reflection, so a
// tag is the single place a limit is stated.
package in

import (
	"encoding/json"
	"time"
)

// EducationIn is one education-history row, used both standalone (the education
// sub-resource routes) and embedded in a teacher create.
//
// Every field is required, which is why `required` lists both menus rather than
// only insert: the same struct is validated on the create route (menu
// "insert"), on add-education (also "insert") and on edit ("update").
type EducationIn struct {
	// InstitutionLevel carries the level *code* (`BACHELOR`), not an id - the
	// service resolves it to institution_level_id. The set of valid codes is
	// deliberately not duplicated here as an enum: the seeded table is the one
	// source of truth, and a third copy in Go would be one more thing to keep
	// in step with the migration.
	InstitutionLevel string `json:"institution_level" required:"insert,update" max:"50" regex:"ascii"`

	Institution string `json:"institution" required:"insert,update" max:"100" regex:"ascii"`

	// The paired *Str fields are how nexcommon parses a date: the validator
	// looks up the field named <Field>Str, reads that string, and assigns the
	// parsed value to the time.Time beside it.
	//
	// The time.Time fields deliberately carry NO json tag. The wire carries one
	// date string, decoded into the Str field; the time.Time is derived from it
	// by the validator. Giving both the same tag would be redundant at best and
	// is rejected by `go vet`'s structtag check, which is reason enough to
	// follow the arrangement real nexcommon services use.
	StudyStartDate    time.Time `required:"insert,update" dateFormat:"date_only"`
	StudyStartDateStr string    `json:"study_start_date"`

	StudyEndDate    time.Time `required:"insert,update" dateFormat:"date_only"`
	StudyEndDateStr string    `json:"study_end_date"`

	// Score is a plain string, holding the number exactly as the client wrote
	// it. The spec types it `numeric` on the wire, so it arrives as a JSON
	// number (`"score": 87.5`); UnmarshalJSON below does the conversion.
	//
	// It is a string rather than a float64 because `87.125` must be rejected,
	// not rounded (criterion 40), and a float64 has already lost the
	// information needed to tell it from 87.125000000000001 by the time
	// anything can look at it (architecture C25). Keeping the literal text
	// lets the `score` regex decide range and precision on the digits, and the
	// same text goes to the NUMERIC(5,2) column unchanged.
	//
	// It is a *plain* string, not json.Number, because nexcommon's validator
	// dispatches on the exact reflected type name: its string branch tests
	// `reflect.String.String() == field.Type.String()`, which is false for
	// json.Number ("json.Number"), and no other branch matches it either - so
	// a json.Number field is silently never validated, and this tag would do
	// nothing. TestScoreTagIsActuallyReachable pins that down.
	Score string `json:"score" required:"insert,update" regex:"score"`

	FocusedSubject string `json:"focused_subject" required:"insert,update" max:"100" regex:"ascii"`
}

// scoreAlias exists only to give UnmarshalJSON a version of EducationIn with
// no methods of its own - decoding through it cannot re-enter UnmarshalJSON.
type scoreAlias EducationIn

// UnmarshalJSON decodes EducationIn with score read as a JSON number.
//
// The default decoder cannot do it: the wire type is a number but the field is
// a string, and json.Unmarshal refuses to put a number into a string field.
// Reading it as json.Number first - which accepts a bare number *or* a quoted
// one, and keeps the digits verbatim either way - and then assigning the text
// is what makes the field land as a string for the validator to inspect.
//
// The embedded scoreAlias carries every other field, tags included; the outer
// Score shadows the embedded one because a shallower field wins.
func (e *EducationIn) UnmarshalJSON(data []byte) error {
	var aux struct {
		*scoreAlias
		Score json.Number `json:"score"`
	}
	aux.scoreAlias = (*scoreAlias)(e)

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// An absent or null score leaves this empty, which the validator reports
	// as a missing field - the same outcome as an empty string.
	e.Score = aux.Score.String()

	return nil
}
