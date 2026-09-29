---
feature_code: FEAT-001
service: sample-project
version: 10
status: complete
pr_url: https://github.com/william22913/test-docs/pull/6
completeness:
  problem: covered
  target_user: covered
  success_criteria: covered
  non_goals: covered
  primary_flow: covered
  edge_cases: covered
  dependencies: covered
  priority: covered
sources:
  - path: knowledge/system_requirement_document.md
    source_version: "1.0"
    note: The SRD for the Teacher Management Module (Teacher CRUD) — business objectives, user roles, functional CRUD rules, the TEACHER entity, and the use-case flow. Primary source for this feature; every claim in this spec traces to it unless marked otherwise.
architect_feedback: []
last_updated: 2026-09-29
---

# FEAT-001 — Teacher CRUD

Service: `sample-project` · Part of the School Database Management System (SDMS).

Claims are marked `(source: SRD §X)` when they come from
`knowledge/system_requirement_document.md`, and `(source: user, this session)` when
they come from the user in conversation. Anything marked **(unresolved)** is an open
decision, not a settled requirement.

## Problem

SDMS covers academic administration broadly — teacher records, student enrollment,
class scheduling, and grade entry — but there is currently no system at all: nothing
centralizes educator data. Teacher profiles must be accurate because they are a
prerequisite for the downstream workflows that depend on them: assigning instructors
to subjects, scheduling classes, and allowing grade submission.
(source: SRD §1)

Three concrete objectives the teacher module must serve (source: SRD §2.1):

- **Centralize educator data** — one source of truth for all active *and historical*
  teacher profiles.
- **Enable downstream workflows** — produce clean data inputs for class allocation
  and subject grade entry.
- **Auditability & security** — only authorized school administrators may modify
  personnel data, and changes must be auditable.

This is the first feature built in `sample-project`. Nothing it depends on currently
exists — see Dependencies.

## Target user

| Role | Allowed operations | Notes |
| :--- | :--- | :--- |
| **School Administrator** | Create, Read, Update, Deactivate | Full management access to all teacher records and their subject/class assignments. |
| **Teacher** | Read (self-profile only), Update (limited) | May view their own profile; may update contact details (phone, email) only. |
| **System / Database** | Soft delete, audit logging | Non-human actor — performs the compliance and integrity operations. |

(source: SRD §3)

Target user for FEAT-001 is the **School Administrator**, acting on teacher records.
A `Teacher` is a *subject* of this feature, not a user of it.

Two things the user confirmed this session qualify that (source: user, this session):

- **No teacher accounts exist.** Teachers are records only; nothing logs in. So
  SRD §3's Teacher self-service role is deferred to whichever feature first gives a
  teacher a login — it is not a live unknown.
- **No administrator distinction exists either.** There is no role model, no
  `is_admin` flag — no authentication or authorization of any kind in this phase.
  "School Administrator" therefore describes **the human doing the job**, not an
  enforced system role.

Read together: FEAT-001's user is real, but nothing in the software identifies or
restricts them. Every authorization requirement in the SRD is deferred — see
Non-goals and the risk note there.

## Success criteria

Each line is an action with an observable result, checkable by someone who was not
involved in this conversation. `[SRD]` = required by the source document; `[new]` =
proposed by the analyst and approved by the user this session. Numbers in `[new]`
lines are tunable defaults, not requirements.

### Create

1. **[SRD]** Admin creates a teacher with all required fields → the record persists
   and is retrievable by the returned `Teacher_ID`.
2. **[new]** `Teacher_ID` matches the agreed format and is unique — creating N
   teachers yields N distinct IDs, with no reuse.
3. **[SRD]** Creating a teacher whose email already exists → rejected with a
   field-level error on `Email`, and **zero rows written**.
4. **[SRD]** `First_Name` / `Last_Name` outside 2–50 characters → rejected, and the
   error names the offending field.
5. **[SRD]** Omitting any required field → rejected, and the error names the field.
6. **[SRD]** `Employment_Status` defaults to `Active` when not supplied.
7. **[new]** Create and its education rows are **atomic** — if the third education
   row fails, no teacher and no orphaned education rows exist.
8. **[SRD — deferred]** On success exactly one onboarding setup link is dispatched to
   the new email. *Deferred: it presumes a teacher account to set up, and no teacher
   accounts exist this phase.*

### Read

9. **[SRD]** Search by `Name` and by `Teacher_ID` each return the expected teacher.
10. **[new]** Search is case-insensitive and matches substrings, not only exact
    values.
11. **[SRD]** The `Active` / `Inactive` filter partitions the set — the two counts
    sum to the total, with no teacher in both. *`On Leave` teachers fall on the
    Active side — see Employment status semantics.*
12. **[SRD]** Pagination defaults to 20 records per page; page 2 returns the next 20
    with no duplicates and no gaps; the final page may be partial.
13. **[new]** The detail view renders with **zero** education rows as a deliberate
    empty state — not an error, not a blank screen.

### Update

14. **[SRD]** Contact details and employment status persist after save.
15. **[SRD]** Attempting to edit `Teacher_ID` or `Hire_Date` → rejected, and the
    stored value is unchanged.
16. **[new]** Each successful update writes **exactly one** audit entry recording
    the before and after values.
17. **[new]** `updated_at` changes only when a field actually changed — a no-op save
    leaves it untouched.

### Deactivate

18. **[SRD]** Deactivate sets `Status = Inactive`; the row still exists and remains
    readable.
19. **[SRD]** No hard-delete path is reachable — the operation is unreachable, not
    merely discouraged.
20. **[SRD — deferred]** A deactivated teacher immediately loses system access.
    *Deferred: there is no access system to revoke from.*
21. **[new]** Reactivation is not available. Any request that would move a teacher
    from `Inactive` back to `Active` or `On Leave` is rejected, and the stored status
    is unchanged. *`Inactive` is terminal in this phase — enforced in the Update path,
    not merely absent from the UI.*

### Authorization

> All three criteria in this group are **deferred** — the user confirmed this session
> that no role model exists of any kind. (source: user, this session)

22. **[SRD — deferred]** A non-administrator cannot reach create, update, or
    deactivate. *No admin distinction exists to test against.*
23. **[SRD — deferred]** A teacher can read their own profile, and cannot read
    another's. *No teacher accounts exist.*
24. **[SRD — deferred]** A teacher can update only phone and email; any other field is
    rejected. *No teacher accounts exist.*

### Education history

25. **[new]** A row's score round-trips exactly — 3.75 stored reads back as 3.75,
    with no floating-point drift.
26. **[new]** Education rows display ordered by study end date, descending.
27. **[new]** Deleting a row removes that row only — the teacher and its sibling rows
    are untouched.
28. **[new]** A score outside 0–100 is rejected with a field-level error.
29. **[new]** An education row whose end date is in the future is rejected.

### Cross-cutting

30. **[new]** Email uniqueness holds under concurrency: two simultaneous creates with
    the same email produce exactly one row. *Enforced in the database, not only in
    application code.*
31. **[new]** `Phone_Number`, when present, matches the project's phone-number
    regex; a non-matching value is rejected. *(The pattern itself is owned by the
    implementation — see Contact fields.)*
32. **[new]** The list view responds within p95 < 500ms at 20 records per page.
    *Invented number — replace with the real target or drop the line.*

### Employment status

33. **[new]** A teacher whose status is `On Leave` is returned by the `Active` filter
    — the status has no behavioural effect on filtering. *(The "can still
    authenticate" half is deferred with the login surface.)*
34. **[SRD]** A teacher whose status is `Inactive` is not returned by the `Active`
    filter. *(The "cannot authenticate" half is deferred.)*

### Concurrency

35. **[new]** A write based on a stale read is rejected: after administrator A saves,
    administrator B's update — built on the pre-save state — fails and returns the
    current record, so B must re-fetch before retrying.
36. **[new]** The check is enforced server-side. A client cannot bypass it by
    omitting a token or version value.

### Field constraints

37. **[new]** A text field receiving a non-ASCII value → rejected, and the error
    names the offending field. Applies to names as well as `Email`.
38. **[new]** `Email` is persisted lowercased: creating with `Teacher@School.EDU`
    stores and returns `teacher@school.edu`, and a later create with
    `TEACHER@SCHOOL.EDU` is rejected as a duplicate.
39. **[new]** A value exceeding its field's max length → rejected, and the error
    names the offending field. A value exactly at the max is accepted.
40. **[new]** An education `score` with more than 2 decimal places → rejected, not
    rounded; the stored value is unchanged.

### Deferred — no authentication or roles this phase

The user confirmed this session that neither teacher accounts nor an administrator
distinction exist (source: user, this session). The criteria below are stated SRD
requirements that **cannot be implemented or verified yet**. They are kept visible
rather than deleted, so the feature that adds authentication knows what it inherits:

| # | Criterion | Deferred because |
| :--- | :--- | :--- |
| 8 | Onboarding setup link dispatched on create | presumes a teacher account to set up |
| 20 | Deactivated teacher loses system access | no access system to revoke from |
| 22 | Non-administrator cannot reach write operations | no admin distinction exists |
| 23 | Teacher reads own profile only | no teacher accounts |
| 24 | Teacher updates phone and email only | no teacher accounts |
| 33 (part) | `On Leave` teacher can still authenticate | no login surface |
| 34 (part) | `Inactive` teacher cannot authenticate | no login surface |

Until then, FEAT-001 is a data-management surface with no access control — see the
risk note under Non-goals.

## Non-goals

Explicitly out of scope for this phase (source: SRD §2.2):

- Student registration and enrollment workflows.
- The interface for teachers entering student scores (the Grading Module).
- Payroll and salary management.

Also out of scope for FEAT-001:

- **Hard deletion** of teacher records — the SRD calls it a strict violation (§4.4),
  so there is no delete path to build.
- **Every Subject, Class, and Student dependency** in the SRD — confirmed deferred by
  the user this session. See the scope-reduction table below.
- **Any link from `TEACHER` to another entity**, except its own education history.
  The user confirmed this explicitly: *"no link to any data, except the education
  history."* (source: user, this session)
- **Authentication and authorization.** No teacher accounts, no administrator
  distinction, no role model of any kind — confirmed by the user this session.
  Seven success criteria are deferred as a result (see the table under Success
  criteria). (source: user, this session)

> **Risk — FEAT-001 has no access control.** With no authentication in place, every
> teacher record is readable and writable by anyone who can reach the endpoints —
> including the personal contact details the SRD wants restricted to administrators
> (§2.1, §3). That is acceptable for a first, internal pass, but **this feature must
> not be deployed anywhere real before the authentication feature lands.** Recorded
> here as a known consequence of the deferral, not as a defect to fix in this phase.

### Confirmed scope reduction vs the SRD

The user confirmed this session that all four class/subject-dependent requirements
below are **not implemented in this phase** — teacher CRUD first, no links to other
data. Recorded, not silently resolved (source: SRD §§4.1–4.4; source: user, this
session):

| SRD requires | Status in FEAT-001 |
| :--- | :--- |
| `Primary_Subject` is a required create field | **Dropped** from Create |
| Update flow reassigns classes when status changes | **Dropped** from Update |
| Deactivate validates no active classes depend on the teacher | **Dropped** from Deactivate |
| Update edits a teacher's assigned subjects | **Dropped** from Update |
| Detail view shows Assigned Classes, Taught Subjects, Active Student Lists | **Dropped** from the detail view |

FEAT-001 therefore ships without these SRD requirements, deliberately. The feature
that introduces `SUBJECT` / `CLASS` / `STUDENT` will need to add them back.

## Primary flow

Four operations, all driven by the School Administrator (source: SRD §4, §6).
Steps that depend on Subject/Class/Student are **deferred** — see the table above.

### Create

1. Administrator opens Teacher Management and selects Create.
2. Enters personal, professional, and contact fields: `First_Name`, `Last_Name`,
   `Email`, `Phone_Number`, `Hire_Date`, `Employment_Status` (default `Active`).
   `Primary_Subject` is dropped. (source: SRD §4.1; source: user, this session)
3. Optionally adds one or more education-history records — see Education history.
   (source: user, this session)
4. Validate email uniqueness and field rules.
5. On success: system generates `Teacher_ID` (format e.g. `TCH-2026-001`), and saves
   the record and its education-history rows. Sending the onboarding account-setup
   link is **deferred** — it presumes a teacher account, and none exist this phase.

### Read / View

1. Administrator searches by `Name` or `Teacher_ID`, or opens the list.
   Search-by-`Subject` is dropped. (source: SRD §4.2; source: user, this session)
2. Optionally filters by `Employment_Status`.
3. List displays paginated, 20 records per page.
4. Detail view shows Personal Information and Education History. Assigned Classes,
   Taught Subjects, and Active Student Lists are dropped. (source: SRD §4.2;
   source: user, this session)

### Update

1. Administrator opens a teacher record and edits contact details or employment
   status. Editing assigned subjects is dropped. (source: SRD §4.3; source: user,
   this session)
2. `Teacher_ID` and historical `Hire_Date` are rejected as edits.
3. May add, edit, or remove education-history rows — see Education history.
   (source: user, this session)
4. Save, and write an audit-log entry.

### Deactivate (no delete)

1. Administrator selects Deactivate.
2. `Status` is toggled to `Inactive` (soft delete — the record is retained).
3. The teacher drops out of the `Active` filter.
4. Save log.

Revoking system access is **deferred** — there is no access system to revoke from.
The observable effect lands entirely in step 2, since `On Leave` and `Active` both
remain in the `Active` filter and only `Inactive` leaves it.

## Education history

Added this session by the user — not present in the SRD. A teacher's education is
stored as an owned history table (`teacher_education_history`), not a column on
`TEACHER`, because a teacher accumulates several records over time and per
institution. (source: user, this session)

Each row records, for one institution:

- **Institution** — where the study took place.
- **Institution level** — high school, university, etc. An explicit field on the
  row, not inferred from the institution name.
- **Study start and end date** — the range the teacher studied there.
- **Score** — a numeric score on a **0–100** scale, carrying **at most 2 decimal
  places**. A score outside the range, or one with finer precision, is rejected.
  (source: user, this session; lengths and character rules in Field constraints)
- **Focused subject** — a free-text marker of what the teacher specialized in *at
  that institution*. The user's example: *Informatics Engineering* at university,
  *Science* at high school. This is deliberately **not** a reference to the school's
  subject data and **not** linked to any subject the teacher teaches here — it labels
  what was studied. (source: user, this session)

**No "last education" concept.** The user removed it this session: with a history
table present, the UI shows the history; there is no derived "most recent
qualification" field or resolution rule to maintain.

**Date rules** (source: user, this session):

- A row's study start must precede its study end.
- **Future dates are rejected** — a study period cannot extend past today.
- **Overlapping ranges are allowed**, including across different institutions (a
  double degree) and including two rows at the *same* institution. The user confirmed
  same-institution overlap is acceptable. (source: user, this session)

**Row rules:** rows are **optional** (a teacher may exist with none) and are **freely
editable and deletable** after creation (source: user, this session).

This adds one dependency entity. The table's columns, keys, and relationship to
`TEACHER` are `/architect`'s to design in `database.md` — recorded here only as a
requirement.

## Employment status semantics

Three statuses exist (source: SRD §4.1, §5). Their behaviour was confirmed by the
user this session — **only `Inactive` is a functional state; `On Leave` is a label
with no behavioural effect** (source: user, this session):

| Status | Keeps system access | In the `Active` filter | Set by |
| :--- | :--- | :--- | :--- |
| `Active` | Yes | Yes | Default on Create |
| `On Leave` | **Yes** — behaves as active | Yes | Update |
| `Inactive` | **No** — revoked immediately | No | Deactivate (soft delete) |

The user's words: *"On leave only shown status, it keep act as a active."* The user
added the rationale that the label exists because a leave is short-lived — it marks
a temporarily-absent teacher without treating them as gone. (source: user, this
session)

**`Inactive` is terminal (one-way).** The user confirmed this session that
reactivation is explicitly **not** in this phase: once deactivated, a teacher stays
that way. Enforcement is required in the Update path — Update freely edits employment
status otherwise, so without an explicit rejection the one-way rule would be
unenforceable in practice. (source: user, this session; see success criterion 21)

Consequences worth noting for whoever implements this:

- `On Leave` is display/reporting metadata only. It does not gate access, does not
  change the filter grouping, and — with the class-reassignment prompt dropped this
  phase — gates nothing at all yet.
- The SRD's §4.3 treats `On Leave` and `Inactive` alike for the class-reassignment
  prompt. That prompt is dropped this phase, so the SRD's only stated behaviour for
  `On Leave` is currently unimplemented. If a later feature reintroduces it, it will
  need to reconcile "`On Leave` should trigger reassignment" with "`On Leave` acts as
  active" — those pull in opposite directions.
- Only Deactivate produces `Inactive`. Nothing else in FEAT-001 writes that value.

## Contact fields

- **`Email`** — required, unique across the entire database, max 50 characters,
  ASCII only, stored lowercased. (source: SRD §4.1; source: user, this session)
- **`Phone_Number`** — **nullable**, max 50 characters, and validated against the
  project's phone-number regex when a value is present. (source: user, this session)
  **(unresolved)** The regex pattern itself is not restated in this spec — the user
  noted the development side already holds it. `/architect` should source the pattern
  from the implementation rather than inventing a second one here.

Lengths and character rules for every field are collected once, in Field constraints.

## Field constraints

Length and character rules for every field, stated once here — the narrative
sections elsewhere describe behaviour, not limits. Confirmed by the user this
session (source: user, this session) unless another source is named.

| Field | Max | Charset | Enforced at |
| :--- | :--- | :--- | :--- |
| `First_Name` | 50 (min 2) | ASCII | validator + DB |
| `Last_Name` | 50 (min 2) | ASCII | validator + DB |
| `Email` | 50 | ASCII | validator + DB |
| `Phone_Number` | 50 | ASCII | validator + DB |
| `Teacher_ID` (`teacher_code`) | 20 | ASCII | system-generated |
| `Institution` | 100 | ASCII | validator + DB |
| `Focused_Subject` | 100 | ASCII | validator + DB |
| `institution_levels.name` | 50 | ASCII | seeded, never user input |
| Education `score` | 0–100, max 2 decimals | numeric | validator + DB `CHECK` |

Notes:

- **Max lengths are inclusive** — a 50-character `Email` is valid, 51 is rejected.
- **ASCII only, all text fields.** A non-ASCII value is rejected with a field-level
  error. This applies to names as well as `Email` — no accented characters and no
  non-Latin scripts in this phase. (source: user, this session)
- **`Email` is stored lowercased.** The case-insensitive uniqueness index
  (`lower(email)`) already enforces this at the database; the API normalizes to
  lowercase before persisting, so the stored value and the compared value agree
  rather than relying on the index alone. (source: user, this session; see
  success criteria 3, 30, 38)
- **Education `score` carries at most 2 decimals.** A value with finer precision
  (e.g. `87.125`) is **rejected**, not rounded — matching the spec's reject-don't-
  coerce pattern elsewhere. `3.75` round-trips exactly. (source: user, this session;
  see criterion 25)
- **Enforcement is at both layers** — DTO validation for the human-readable message,
  and the database (`VARCHAR` caps, `CHECK`, `UNIQUE`) as the backstop that cannot be
  bypassed. This extends the pattern already used for email uniqueness (criterion 30)
  and field rules (criterion 4). (source: user, this session)

**Deviation from the SRD.** SRD §5 gives `email VARCHAR(100)` and `phone
VARCHAR(20)`. The user's ruling this session is **50 for both**, which supersedes the
SRD — consistent with how this spec already supersedes it on the status enum and the
Subject/Class/Student links. Consequences for `/architect`'s `database.md`:
`email` 100→50, `phone` 20→50, and `institution` / `focused_subject` 150→100 to match
the caps above. (source: SRD §5; source: user, this session)

**No `Education notes` field.** The architect's feedback referenced "education
notes". No such field exists in this spec, and the user confirmed this session it was
never requested. An education row carries Institution, Institution level, study
start/end dates, Score, and Focused subject — nothing else. Recorded so it is not
raised again.

## Edge cases

Stated in the SRD (source: SRD §4.1, §4.4, §7):

- **Duplicate email on create** — rejected; email is unique database-wide.
- **Hard delete attempted** — a strict violation; hard deletion is disabled to
  preserve student historical grade logs and attendance records.
- **Duplicate profiles under different emails** — mitigated by strict uniqueness
  checks on the identifying attributes.

Deferred with Subject/Class (confirmed this session, so not edge cases FEAT-001
handles): status change while classes are assigned, and soft-delete while active
classes depend on the teacher.

Education history — resolved this session (source: user, this session):

- Optional per teacher — a teacher with zero rows is valid, and the profile must
  render sensibly in that state.
- Rows are editable and deletable, including the most recent one; nothing else in the
  profile derives from education, so removal has no cascade.
- Score outside 0–100 → rejected.
- End date in the future → rejected.
- Overlapping ranges → permitted, at different institutions and at the same one.

Status and concurrency — resolved this session (source: user, this session):

- **Reactivation** — one-way. `Inactive` is terminal in this phase; Update must
  reject a transition out of it (success criterion 21).
- **Concurrent edits** — the lost-update case is handled: a write built on a stale
  read is rejected, and the second administrator must re-fetch before retrying
  (success criteria 35–36).

## Dependencies

- **Identity / authentication provider — not in this phase.** The SRD assumes an
  existing or to-be-integrated IdP via OAuth/SAML (§4.1, §4.4, §7), but the user
  confirmed this session that no accounts and no role model exist. FEAT-001
  therefore has **no** identity dependency to satisfy — it is not waiting on an IdP,
  it simply has no access control. Adding one is a later feature's job, and that
  feature inherits the seven deferred criteria listed under Success criteria.
  (source: SRD §4.1, §4.4, §7; source: user, this session)
- **Education history** (`teacher_education_history`) — a new entity created by this
  feature (see Education history). Institution, institution level, study date range,
  score, and focused subject per row; owned by `TEACHER` and read with it.
  (source: user, this session)
- **Audit log** — §2.1 and §3 require audit trails for personnel changes; no log
  schema is specified (source: SRD §2.1, §3, §6).
- **No other entity links.** The user confirmed this session that `TEACHER` links to
  no other data except its own education history. The SRD's Subject / Class / Student
  dependencies are dropped for this phase (see the scope-reduction table).
  (source: user, this session)

## Priority

Build order, confirmed by the user this session (source: user, this session):

1. **Create + education history** — first, because it is the only step that can
   invalidate the data model. Discovering late that `Teacher_ID` is not unique
   enough, or that the education rows need an extra key, is expensive here and cheap
   now.
2. **Read** — list, search, filter, pagination, and the detail view including its
   zero-education-rows empty state.
3. **Update** — including the two restricted fields and the audit entry.
4. **Deactivate** — last, and smallest: with Class and authentication both dropped,
   it is nothing but a status toggle.

- **This phase:** Teacher CRUD, teacher-only, with education history. (source: SRD
  §2.2; source: user, this session)
- **Deferred:** student registration/enrollment, the score-entry interface, payroll
  (source: SRD §2.2), and every Subject/Class/Student-dependent behaviour listed in
  the scope-reduction table.
- **Teacher self-service** (self-read + limited contact update): role is defined in
  §3 but not listed in-scope — see Open questions.

## Open questions

These do **not** block `/architect` — each is either an implementation-owned detail
or a question that only arises in a later feature. They are recorded so they are not
rediscovered as surprises.

1. **Audit approval** for `Teacher_ID` / `Hire_Date` edits — who approves, and is it
   an approval workflow in this system or an out-of-band process? *The restriction
   itself is confirmed (criterion 15); only the approval mechanism is open.*
2. **Phone regex ownership** — confirm the implementation's existing pattern is the
   one to use, rather than defining a second one here.
3. **Teacher self-service role and all access control** — deferred to the feature
   that first introduces authentication. SRD §3 defines the role; the seven deferred
   criteria in Success criteria are what that feature inherits. Not a live unknown —
   a recorded handoff.
4. **`On Leave` vs the SRD's class-reassignment prompt** — SRD §4.3 treats `On Leave`
   like `Inactive` for reassignment, which contradicts "`On Leave` acts as active".
   Moot this phase (the prompt is dropped); must be reconciled when `CLASS` returns.
5. **Concurrency mechanism** — criteria 35–36 state the requirement; whether it is an
   optimistic version column, an ETag, or a conditional write is `/architect`'s call.
