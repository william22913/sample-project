package in

import "time"

// TeacherIn is the request body for every teacher route.
//
// One struct rather than one per route, which is what the controller requires:
// WrapService reads the body DTO from a single service.GetDTO() with no
// per-endpoint override, so a service has exactly one body shape. nexlegal's
// DataGroupIn and BusinessGroupIn are built the same way. The per-route
// difference is carried by the `required:"<menu>"` tags - a field is examined
// only on the menus it names, and on any other route it is neither validated
// nor read.
//
// The three menus: insert (POST /teachers), update (PUT /teachers/{id}),
// deactivate (POST /teachers/{id}/deactivate).
type TeacherIn struct {
	FirstName string `json:"first_name" required:"insert,update" min:"2" max:"50" regex:"ascii"`
	LastName  string `json:"last_name" required:"insert,update" min:"2" max:"50" regex:"ascii"`

	// max:"50" with no min, so an empty email is caught by the validator's own
	// zero check as ErrEmptyField rather than as a length violation.
	//
	// auto_fix:"lowercase" is criterion 38: the value is normalized before it
	// reaches the service, so what is stored, what is compared against the
	// case-insensitive unique index, and what is returned all agree. The index
	// would keep the data consistent on its own, but the caller would still see
	// the mixed-case value they sent.
	//
	// `regex:"email"` also carries the ASCII rule for this field: nexcommon's
	// EMAIL_REGEX is built from \w and friends, which in Go match ASCII only.
	Email string `json:"email" required:"insert,update" max:"50" regex:"email" auto_fix:"lowercase"`

	// Optional, so `empty:"allowed"` short-circuits every check below it when
	// the value is absent - which is what makes "no phone number" valid rather
	// than a regex failure.
	//
	// `required:"insert,update"` is not in conflict with that, and is not
	// redundant with it: the tag is what makes the validator examine the field
	// at all, and an optional field without it is skipped entirely - max and
	// regex included, so any string whatsoever would be accepted.
	Phone string `json:"phone" required:"insert,update" empty:"allowed" max:"50" regex:"phone"`

	// Update only. On create this is not examined, and the service does not
	// read it either - criterion 6's "status defaults Active" is the column
	// DEFAULT in the migration, so there is one place that decides it.
	//
	// The enum is constanta.StatusActive/StatusOnLeave only. INACTIVE is
	// deliberately not an accepted value: only deactivation produces it, so a
	// caller cannot reach Inactive through an update (criterion 21, second
	// half).
	Status string `json:"status" required:"update" enum:"teacher_status"`

	// Insert only - see the presence check in Update. No json tag on the
	// time.Time itself: the wire carries hire_date as a string, and the
	// validator parses it into the time.Time from the Str companion. See
	// EducationIn for the full note on the arrangement.
	HireDate    time.Time `required:"insert" dateFormat:"date_only"`
	HireDateStr string    `json:"hire_date"`

	// Insert only. `required:"insert"` is not a demand that the array be
	// non-empty - the tag is the validator's switch for "walk this field's
	// elements on the insert route", and without it the elements are never
	// examined. An absent or empty array is valid (criterion 13's deliberate
	// empty state).
	//
	// Education changes after creation go through the education sub-resource
	// routes, and this field is simply not read on update. It is not rejected
	// the way teacher_code is: a PUT that echoes the resource back is a normal
	// client behaviour, and no criterion asks for it to fail.
	Educations []EducationIn `json:"educations" required:"insert"`

	// Present to be rejected on update, never written (criterion 15). A pointer
	// so "absent" and "present but empty" stay distinguishable - a caller
	// sending teacher_code:"" is still attempting an edit of an immutable
	// field.
	TeacherCode *string `json:"teacher_code"`

	// The optimistic lock's expected value (architecture A5), required on both
	// write routes rather than optional: criterion 36 says the check cannot be
	// bypassed by leaving the token out, and an optional field would make
	// omission the bypass.
	//
	// `dateFormat:"timestamp"` is this project's layout, not nexcommon's
	// "default", and it carries microseconds deliberately - see
	// validator.TimestampLayout for why second precision would let two updates
	// in the same second both succeed.
	UpdatedAt    time.Time `required:"update,deactivate" dateFormat:"timestamp"`
	UpdatedAtStr string    `json:"updated_at"`
}

// HireDatePresent reports whether the caller sent hire_date at all.
//
// It reads the Str companion, not the parsed time, because on the update menu
// the validator never parses it - `required:"insert"` keeps the field out of
// the update validation block entirely, so the time.Time stays zero and only
// the raw string survives. A non-empty string is therefore the whole test.
//
// The gap this leaves, stated so it is not mistaken for an oversight: a body
// carrying `"hire_date": null` sets nothing, so it reads as absent rather than
// as a rejected edit. Making it distinguishable would mean giving the Str
// companion a pointer type, which the validator cannot parse through - it
// looks the field up by name and reads it as a string.
func (t TeacherIn) HireDatePresent() bool {
	return t.HireDateStr != ""
}
