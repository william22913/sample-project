---
feature_code: FEAT-001
service: sample-project
version: 5
status: complete
spec_version: 10
sources:
  - path: knowledge/system_requirement_document.md
    source_version: "1.0"
    note: The SRD for the Teacher Management Module (Teacher CRUD). Source of the CRUD flows and the hard-delete prohibition. Superseded by spec.md in several places — the status enum, Primary_Subject, class reassignment, and all Subject/Class/Student links.
  - path: knowledge/nexcommon-reuse-survey.md
    source_version: "1.3"
    note: Survey of the shared nexcommon library. Source of every reuse decision below — audit_helper (§1), the no-op WhitelistValidator (§2), the regex constants (§3), the HTTP/DTO scaffolding (§4), the house migration style (§5), the optimistic-lock error ErrDataLocked (§7), and the i18n bundle loader (§8).
non_functional_concerns:
  - audit_durability
  - search_index_ceiling
  - no_authentication
pr_url: https://github.com/william22913/test-docs/pull/4
last_updated: 2026-09-29
---

# FEAT-001 — Architecture

A greenfield Go service on `nexcommon`, Postgres-backed. Four endpoints for
teacher CRUD plus three for education rows. Almost nothing is built from
scratch — the shared library already covers auth, audit, validation and HTTP
wiring, and the reuse-vs-build call for each is recorded below.

## Scope

Decides the stack, the components, the reuse-vs-build calls, and the HTTP
surface. Data shape lives in `database.md`. Implementation is `/go-dev`'s.

This is the **first** feature in `sample-project` — the repo was empty at the
time of writing, so there was no existing `CLAUDE.md`/`PROJECT.md` to inherit
conventions from and the stack choice was genuinely open.

## Stack

**Go + `github.com/nexsoft-git/nexcommon` + PostgreSQL.** (Decision B1, user,
this session.)

- **Go** — the pipeline below this design is `/go-dev`; the environment's
  generators emit Go and Postgres DDL. Nothing in the feature argues for
  anything else.
- **`nexcommon`** — the shared library the team's services are built on. It
  supplies the HTTP controller wrapping, the validator strategies, the DAO
  get-list flow, audit logging, and the regex constants. Reusing it is not a
  preference here; it is what "the project's phone-number regex" in the spec
  turned out to *mean* — the pattern was in the library, not in any service.
- **PostgreSQL** — the feature is plain transactional relational CRUD, and
  `nexcommon`'s DAO layer targets it directly. The concurrency requirement
  (criteria 35–36) is satisfied by a row-level conditional `UPDATE`, which is
  what Postgres does well.

## Components

**New, built by this feature:**

| Component | Role |
| :--- | :--- |
| `teachers` table + migration | The teacher record. `database.md`. |
| `teacher_education_histories` table | Owned education rows. `database.md`. |
| `institution_levels` table + seed | Lookup for the per-row institution level. Seven seeded codes. `database.md`. |
| `i18n/institution_level/{en-US,id-ID}.json` | English and Indonesian labels for those seven codes. Content, not schema — see A6. |
| Teacher DAO / service / DTOs / endpoints | Create, list, detail, update, deactivate. |
| Education DAO / service / DTOs / endpoints | Add, edit, delete a row. |
| Audit wiring | Instantiating `audit_helper` and registering `teachers`. |

**Reused from `nexcommon`** — nothing here is reimplemented:

| Concern | Reused | Where it came from |
| :--- | :--- | :--- |
| Route wrapping, DTO decode, tag validation, response/error formatting | `http.NewHTTPController` + `WrapService` / `WrapServiceListData` | Survey §4 |
| Auth | `WhitelistValidator` | Survey §2 |
| Audit trail | `services/audit_helper` | Survey §1 |
| Phone + email patterns | `regex.PHONE_NUMBER_WITH_COUNTRY_CODE`, `regex.EMAIL_REGEX` | Survey §3 |
| List filtering, search, ordering, pagination | `dto/in.GetListRequest` + the get-list validator and DAO | Survey §4 |
| Request context | `context.ContextModel` | Survey §4 |
| Stale-write rejection | `error.ErrDataLocked` (`E-4-CMD-DTO-007`) | Survey §7 |
| English / Indonesian label dictionaries | `bundles.NewBundles` + `ReadMessageBundle`, over `nicksnyder/go-i18n/v2` | Survey §8 |

**Untouched:** everything Subject / Class / Student, student enrollment, the
grading module, payroll, and any authentication or role model.

## Reuse-vs-build decisions

Recorded ADR-style. Each was checked against the survey before anything new was
proposed.

### A1 — Audit logging: reuse `audit_helper`, do not build an audit table

- **Options.** (a) A `teacher_audit_logs` table written inside the same
  transaction; (b) reuse nexcommon's `audit_helper`, which snapshots the
  before-state and publishes the diff to NATS.
- **Chosen: (b).** Decision D1 (user, this session). The helper already
  implements before/after capture via `row_to_json` and `FOR UPDATE`, so a table
  would duplicate working code.
- **Consequence, and it is not free.** The helper publishes in a goroutine with
  a plain `nats.Publish`; a failure is logged, not retried and not rolled back.
  Criterion 16's "exactly one audit entry" is therefore **at-most-once**, and
  the entries are not durable until a NATS consumer persists the subject. Two
  hard requirements fall out: **this service needs a JetStream connection, and
  something must consume the audit subject.** If there is no NATS, this decision
  flips to (a) and exactly-once is restored. Confirm before implementation.
- **Also settled by this choice:** only `teachers` is registered for auditing.
  Auditing education rows as well would emit one entry per changed row and break
  criterion 16's "exactly one" on any save that touched education. The cost —
  education changes have no independent audit trail — is flagged in
  `database.md`.

### A2 — Auth: reuse `WhitelistValidator`, which authenticates nothing

- **Options.** (a) Build a whitelist/API-key check; (b) use nexcommon's
  `WhitelistValidator`.
- **Chosen: (b).** Decision B3 (user, this session).
- **What this actually is, stated plainly:** `WhitelistValidator` is
  `return nil`. It is nexcommon's *declaration that a route has no
  authentication* — not a weak auth mechanism, and not a stopgap. This is the
  correct match for the spec, which records that no accounts, no administrator
  distinction and no role model exist. Every endpoint uses it.
- **Consequence.** Nothing populates `ctx.AuthAccessTokenModel`, so audit entries
  carry no actor and `created_by`/`updated_by` stay `NULL`. The spec's risk note
  stands unchanged: **this feature must not be deployed anywhere real before the
  authentication feature lands.**

### A3 — Phone validation: reuse the library constant

- **Chosen:** `regex.PHONE_NUMBER_WITH_COUNTRY_CODE` (Decision B2/D4, user, this
  session).
- **Why this resolves a spec open question:** the spec's Open question 2 says the
  pattern should be sourced from the implementation rather than reinvented, and
  the user believed "the development side already holds it". It was not in the
  service repo — the repo was empty — it is in the shared library. The spec's
  premise holds at the library level, so **no spec correction is needed.**
- **Consequence the user accepted explicitly:** the pattern demands a leading
  `+` and a literal hyphen. `+62-81234567890` passes; `081234567890` is
  **rejected**.
- **Rejected alternative:** `regex.NAME_STANDARD` for `first_name`/`last_name` —
  it permits only `^[A-Z][a-z]+...`, so `O'Brien` and `MARY-JANE` fail. Spec
  criteria 4/5 ask for length rules instead, which is what is implemented.

### A4 — `institution_levels`: a seeded lookup, not a free-text column

- **Options.** (a) `institution_level VARCHAR` on the education row; (b) a
  lookup table.
- **Chosen: (b), seeded only, no CRUD surface.** Decision D3 (user, this
  session).
- **Not a spec conflict.** The spec says the level is "an explicit field on the
  row, not inferred from the institution name" — a foreign key to a seeded
  lookup is still explicit, and satisfies that sentence. The spec's "`TEACHER`
  links to no other data except its own education history" also stays literally
  true: this FK hangs off `teacher_education_histories`, not `teachers`. So no
  `architect_feedback` was filed.
- **Consequence.** The table stores a stable **code**, not a display string: the
  seeded set is fixed at seven values (`PRESCHOOL`, `PRIMARY_SCHOOL`,
  `MIDDLE_SCHOOL`, `HIGH_SCHOOL`, `BACHELOR`, `MASTER`, `DOCTOR`) and is no
  longer provisional — the user fixed the list this session. The labels a human
  reads are **not** in the database; they are i18n bundle content (A6). Making
  levels admin-editable would be a **separate feature** with its own endpoints —
  explicitly out of scope here.

### A5 — Concurrency: optimistic lock via `updated_at`, not a `version` column

- **Options.** (a) `version` integer; (b) `ETag`/`If-Match`; (c) compare
  `updated_at`; (d) a conditional write on field values.
- **Chosen: (c) — compare `updated_at`.** The spec's Open question 5 leaves the
  mechanism to `/architect`; criteria 35–36 only state the requirement. This is
  the nexcommon convention: the client re-presents the `updated_at` it read, and
  a conditional `UPDATE ... WHERE id = $1 AND updated_at = $client_updated_at`
  either changes one row (current) or zero rows (stale).
- **Why the `version` column was dropped:** it is not how this team's code does
  optimistic locking. The shared library exposes `error.ErrDataLocked`
  (`E-4-CMD-DTO-007`, HTTP 400, names the field via `errFieldNameConverter`) for
  exactly the zero-rows-affected case — see the reuse survey §7. A separate
  `version` column would be a second concurrency mechanism alongside the one the
  library already supports, and the spec asks for one mechanism, not two. A
  client cannot bypass the check by omitting the value: the expected `updated_at`
  is matched against the row the `UPDATE` targets, so an omitted value matches
  nothing (criterion 36). Full mechanism in `database.md`.
- **Rejected:** (a) the `version` column — not the team convention, and
  redundant once `updated_at` already carries monotonic change. (d) conditional
  write on field values is the implicit form of (c) and adds nothing.

### A6 — Institution-level labels: reuse the i18n bundles, do not build a translations table

- **Options.** (a) A `institution_level_translations` child table
  (`institution_level_id`, `locale`, `label`); (b) two label columns on the
  lookup itself (`label_en`, `label_id`); (c) reuse nexcommon's i18n bundle
  loader over JSON dictionaries.
- **Chosen: (c).** The user's requirement is that each seeded level carries an
  English and an Indonesian label. The library already has a translation store
  and it is **file-based, not row-based**: `bundles.NewBundles(rootDir,
  defaultLanguage)` walks `i18n/<bundle>/<locale>.json`, where the bundle name is
  the sub-path joined with `.`, and `ReadMessageBundle(bundleName, messageID,
  language, param)` resolves a key. Survey §8.
- **Shape.** Bundle `institution_level` (directory `i18n/institution_level/`),
  one file per locale — `en-US.json`, `id-ID.json`, matching the two locales the
  library already ships. The seven seeded codes are the message IDs, so the
  dictionary keys and the seeded column values are the same strings by
  construction and cannot drift into separate vocabularies.
- **Why not a table.** (a) would be a **second** i18n mechanism beside the one
  the library and its error formatter already use — `error/formator.go` resolves
  `common.constanta` and `common.error` from these same bundles. And the labels
  are not relational data: nothing joins on them, filters by them, or enforces
  uniqueness over them, so a table would buy a join and a migration for content
  that is versioned better in git. (b) hard-codes the locale set into the schema,
  so a third language becomes a migration rather than a new JSON file.
- **Consequence, and it is the real limitation.** `ReadMessageBundle` takes a
  `language` argument, and nexcommon sources that from the **auth token**
  (`ctx.AuthAccessTokenModel.Locale = tokenModel.Locale`). FEAT-001 has no auth
  (A2), so `Locale` is never populated and **there is no per-request language
  selection in this phase** — the default language handed to `NewBundles`
  (`id-ID`, matching the library's own `language.Indonesian` bundle default and
  its tests) decides the response language for every caller. The spec is silent
  on language: it mandates no locale field, header or query parameter, and pins
  no column for one. Adding a client-selectable mechanism would be inventing API
  surface the spec never authorised, so **no `architect_feedback` is filed** and
  the gap is carried as an open item in `database.md` for the language-selection
  feature to close.
- **Failure mode, which is benign.** `ReadMessageBundle` `recover()`s to the
  `messageID` on any panic — missing bundle, missing file, missing key — so an
  unrecognised code degrades to returning `HIGH_SCHOOL` verbatim rather than
  producing a 500. The API therefore stays up on a content gap. The cost is that
  a missing translation is silent; Gate 1.13 in `database.md` checks the
  dictionary covers every seeded code.

## HTTP surface

Indicative paths; `/go-dev` follows the project's `WrapService` conventions.
Every route uses `WhitelistValidator` (A2).

| Method | Path | Spec operation |
| :--- | :--- | :--- |
| `POST` | `/teachers` | Create (with optional embedded education rows) |
| `GET` | `/teachers` | List — search by name / `Teacher_ID`, filter status, paginate |
| `GET` | `/teachers/{id}` | Detail, including the education array (may be empty) |
| `PUT` | `/teachers/{id}` | Update contact details / status / expected `updated_at` |
| `POST` | `/teachers/{id}/deactivate` | Deactivate — status toggle only |
| `POST` | `/teachers/{id}/educations` | Add an education row |
| `PUT` | `/teachers/{id}/educations/{eduId}` | Edit an education row |
| `DELETE` | `/teachers/{id}/educations/{eduId}` | Remove an education row |

**Education rows get their own sub-resource rather than being mutated inside a
full-teacher `PUT`.** Criterion 27 is a row-scoped observable action — "deleting
a row removes that row only, the teacher and its sibling rows are untouched".
Folding add/edit/delete into a whole-teacher replace makes that criterion hard to
verify and invites a caller to clobber siblings by omission.

**There is no `DELETE /teachers/{id}`.** Not omitted by accident — criterion 19
requires hard delete to be *unreachable*, so no route and no DAO method exists
for it. Deactivation is its own endpoint because the spec treats it as its own
operation with its own flow.

## Non-functional concerns

Three non-functional concerns came up in this design. Each is handled here
rather than in a separate document, because each is a paragraph, not a doc.

1. **Audit durability.** The NATS publish is fire-and-forget (A1). This is the
   single most consequential thing on this page. It is recorded in A1 and in
   `database.md`, and it is the reason criterion 16 is at-most-once.
2. **Search-index ceiling.** Substring name search is a sequential scan by
   design; the `pg_trgm` upgrade path is written down but not applied
   (`database.md`). It collides with criterion 32, whose latency number the spec
   itself calls invented.
3. **No auth.** The spec already carries this as a risk note. A2 restates the
   consequence and does not soften it.

No separate non-functional document is written for any of the three. Each is
recorded in the ADR or `database.md` section that owns it, and a document
holding a single paragraph would be a stub, not a reference.

---

## Spec traceability

One line per requirement in `spec.md`. Every line states a concrete
architectural answer, or an explicit "not applicable, because …". Criteria
marked **[deferred]** are deferred by the spec itself and are carried here as
"not applicable" so the list stays checkable end to end.

### Problem

- **Centralize educator data** — the `teachers` table is the single source of
  truth. Historical rows are retained (soft delete via `status`), never removed.
- **Enable downstream workflows** — satisfied by producing validated, uniquely
  identified teacher records through the API. No schema provision is made for
  class/subject allocation, because every such link is dropped this phase.
- **Auditability & security** — audit via `audit_helper` (A1); security is
  deferred with no auth this phase (A2), and the residual risk is recorded.
- *"Nothing it depends on currently exists"* — correct at the time of writing;
  this design creates the first tables in the repo.

### Target user

- **School Administrator** — the human doing the job. No role model exists, so
  no enforcement: every route uses the no-op `WhitelistValidator` (A2).
- **Teacher (self-read / limited update)** — not applicable: no teacher accounts
  exist, so nothing can identify a teacher as a caller. No account table, no
  login route.
- **System / Database (soft delete, audit)** — soft delete is `status =
  'INACTIVE'`; audit is the `audit_helper` publish (A1).

### Success criteria

**Create**

- **C1** create with required fields → persists, retrievable by returned `Teacher_ID` — `POST /teachers`; the trigger assigns `teacher_code`; `GET /teachers/{id}`.
- **C2** `Teacher_ID` matches format, unique, no reuse — PK-derived `TCH-<hire_year>-<id>`, `uq_teachers_teachercode`. Gaps are possible; reuse is not.
- **C3** duplicate email → rejected with field-level `Email` error, zero rows written — `uq_teachers_email_lower` (case-insensitive, see `database.md` N1); pre-check is for the message only; single transaction.
- **C4** `First_Name`/`Last_Name` outside 2–50 → rejected naming the field — DTO validator; `VARCHAR(50)` caps the top end.
- **C5** omitting a required field → rejected naming the field — `NOT NULL` + DTO validator.
- **C6** `Employment_Status` defaults to `Active` — column `DEFAULT 'ACTIVE'`.
- **C7** create and its education rows atomic — one `sql.Tx` spanning both inserts, rolled back together.
- **C8** *[deferred]* onboarding setup link — **not applicable**: it presumes a teacher account to set up, and no accounts exist. No mail dependency is introduced.

**Read**

- **C9** search by `Name` and by `Teacher_ID` — get-list validator `lk` on names, `eq` on `teacher_code`.
- **C10** case-insensitive, substring — `ILIKE` through the `lk` operator. Unindexed by design; ceiling documented in `database.md`.
- **C11** `Active`/`Inactive` filter partitions the set — `Active` is `status <> 'INACTIVE'`; the two filters are complements, so no row is in both.
- **C12** pagination, 20/page, no duplicates or gaps — get-list offset/limit, default page size 20.
- **C13** detail view with zero education rows is a deliberate empty state, not an error — the education query returns an empty array; the teacher detail response is `200`, not `404` or `500`.
- **C14** — *(criterion 14 is an Update item; listed there.)*

**Update**

- **C14** contact details and employment status persist after save — `PUT /teachers/{id}`.
- **C15** editing `Teacher_ID` or `Hire_Date` → rejected, stored value unchanged — neither is a writable field on the update DTO; an attempt is an explicit rejection, not a silent ignore.
- **C16** exactly one audit entry per successful update with before and after — `audit_helper` registered on `teachers` only (A1). At-most-once under publish failure; flagged.
- **C17** `updated_at` changes only when a field actually changed — the service diffs before writing and skips the `UPDATE` entirely on a no-op save. No trigger; the diff is needed for A1 anyway.
- **C18** deactivate sets `Status = Inactive`, row still exists and is readable — soft delete via `POST /teachers/{id}/deactivate`; the row is untouched otherwise.

**Deactivate**

- **C18** — see above (spec lists it under Deactivate too).
- **C19** no hard-delete path reachable — no `DELETE /teachers/{id}` route and no DAO delete method. Unreachable, not merely discouraged.
- **C20** *[deferred]* deactivated teacher loses system access — **not applicable**: there is no access system to revoke from.
- **C21** reactivation unavailable; any transition out of `Inactive` rejected, stored status unchanged — enforced in the Update path (a guard, not mere UI absence) and in Deactivate itself. `Inactive` is terminal.
- **C22** *[deferred]* non-administrator cannot reach writes — **not applicable**: no admin distinction exists to test against (A2).
- **C23** *[deferred]* teacher reads own profile only — **not applicable**: no teacher accounts exist.
- **C24** *[deferred]* teacher updates phone and email only — **not applicable**: no teacher accounts exist.

**Education history**

- **C25** score round-trips exactly — `NUMERIC(5,2)`, and the Go side must use a decimal representation, not `float64`.
- **C26** rows ordered by study end date descending — `ORDER BY study_end_date DESC`, backed by a `(teacher_id, study_end_date DESC)` index.
- **C27** deleting a row removes that row only — row-scoped `DELETE /teachers/{id}/educations/{eduId}`; sibling rows and the teacher are untouched.
- **C28** score outside 0–100 → rejected with a field-level error — `CHECK` constraint for the guarantee, DTO validator for the message.
- **C29** education row with a future end date rejected — **application-layer only**: a Postgres `CHECK` cannot reference `CURRENT_DATE` (not immutable). Recorded as the one education rule with no database backstop.

**Cross-cutting**

- **C30** email uniqueness holds under concurrency — the functional unique index `uq_teachers_email_lower`, deliberately the enforcement point rather than the application pre-check. Case-insensitive: `Alice@…` and `alice@…` are the same mailbox and collide.
- **C31** `Phone_Number`, when present, matches the project's phone regex — `regex.PHONE_NUMBER_WITH_COUNTRY_CODE` from nexcommon (A3); column nullable, so absent is valid.
- **C32** p95 < 500ms at 20 records/page — **no target is designed against.** The spec marks the number invented ("replace with the real target or drop the line"). No latency budget was agreed, so none is claimed; the substring-search ceiling is documented for when a real target arrives.

**Employment status**

- **C33** `On Leave` is returned by the `Active` filter — `Active` is defined as `status <> 'INACTIVE'`, so `On Leave` groups with `Active` by construction.
- **C34** `Inactive` is not returned by the `Active` filter — same definition; `Inactive` is the only excluded value.

**Concurrency**

- **C35** a write based on a stale read is rejected and returns the current record — conditional `UPDATE ... WHERE id = $1 AND updated_at = $client_updated_at`; zero rows affected → `error.ErrDataLocked` (`E-4-CMD-DTO-007`) carrying the current record so the loser can re-fetch. Mechanism: A5.
- **C36** enforced server-side; cannot be bypassed by omitting the token — the expected `updated_at` is matched against the stored row, so an omitted value matches nothing rather than skipping the check.

### Primary flow

**Create**

- **1** administrator opens Teacher Management, selects Create — no architectural bearing; the API is the surface, and no UI is in scope.
- **2** enters personal, professional and contact fields; `Primary_Subject` dropped — the create DTO carries exactly those fields; `Primary_Subject` does not exist in the schema or the DTO.
- **3** optionally adds education-history rows — education rows are accepted in the same create call and written in the same transaction (C7).
- **4** validate email uniqueness and field rules — `UNIQUE` constraint + DTO validator.
- **5** on success, `Teacher_ID` generated and the record saved; onboarding link deferred — trigger assigns the code; no mail path exists (C8).

**Read / View**

- **1** search by name or `Teacher_ID`; search-by-Subject dropped — get-list validator; there is no subject column to search.
- **2** optionally filter by employment status — status filter.
- **3** list is paginated at 20/page — get-list default.
- **4** detail shows Personal Information and Education History; the rest dropped — the detail DTO carries the teacher record and the education array, and nothing else.

**Update**

- **1** edit contact details or employment status; subject editing dropped — update DTO.
- **2** `Teacher_ID` and `Hire_Date` rejected as edits — C15.
- **3** add, edit or remove education rows — the education sub-resource endpoints.
- **4** save and write an audit-log entry — one transaction; audit via `audit_helper` (A1).

**Deactivate**

- **1** administrator selects Deactivate — dedicated endpoint.
- **2** `Status` toggled to `Inactive`, record retained — soft delete.
- **3** the teacher drops out of the `Active` filter — follows from the filter definition.
- **4** save log — audit on the teacher record.

### Edge cases

- **Duplicate email on create** — rejected; `uq_teachers_email_lower` (C3). The pre-check query must be written `lower(email) = lower($1)` to use the index.
- **Hard delete attempted** — no path exists (C19).
- **Duplicate profiles under different emails** — **only partially addressed, and worth stating plainly.** The SRD's mitigation is "strict uniqueness checks on the identifying attributes", but the only attribute this feature makes unique is `email`. Nothing prevents two records for the same person under different addresses — there is no government-ID or equivalent field in the spec's attribute list. This is an accepted gap, not a solved problem.
- **Education: optional per teacher** — the FK is on the child, so zero rows is a valid state and the detail view renders empty (C13).
- **Education: editable and deletable, including the most recent** — row-level endpoints (C27); nothing else derives from education, so deletion cascades nowhere.
- **Education: score outside 0–100** — rejected (C28).
- **Education: end date in the future** — rejected (C29, app layer).
- **Education: overlapping ranges permitted** — no exclusion constraint is created; same-institution overlap included.
- **Status: reactivation is one-way** — `Inactive` terminal, enforced in the Update path (C21).
- **Concurrency: concurrent edits** — stale write rejected, current record returned (C35–C36).

### Dependencies

- **Identity / authentication provider** — **not applicable**: not in this phase, and the feature does not wait on one. A later auth feature inherits the seven deferred criteria.
- **Education history** — created by this feature as `teacher_education_histories`, owned by `teachers`, read with it. Columns and keys in `database.md`.
- **Audit log** — satisfied by reusing `nexcommon`'s `audit_helper` rather than a new schema (A1). Requires NATS JetStream and a consumer.
- **No other entity links** — honored. `teachers` links to nothing but its own education rows. `institution_levels` is a lookup hanging off the education row, not a link from `teachers` (A4).
- **`institution_levels` (new, not in the spec)** — added by decision D3. A seeded lookup, no CRUD surface. Recorded here rather than as `architect_feedback`, because it contradicts no spec requirement — see A4.

### Non-goals — confirmation that nothing was designed for them

- **Hard deletion** — no route, no DAO method (C19).
- **Student enrollment, grading interface, payroll** — untouched.
- **Every Subject / Class / Student dependency** — no table, column or DTO field references them.
- **Authentication / authorization** — deliberately absent; `WhitelistValidator` is a no-op by design (A2).
