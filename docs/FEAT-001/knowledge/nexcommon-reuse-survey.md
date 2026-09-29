# nexcommon reuse survey — FEAT-001

**Version:** 1.3
**Date:** 2026-09-28, amended 2026-09-29.
**Surveyed against:** local clone at `C:\Users\Pongo\Documents\github\nexcommon`, `main` branch.

> **Changelog — 1.3.** Adds §8 — the i18n bundle loader (`bundles`). Needed
> because the `institution_levels` labels now have to resolve to English and
> Indonesian, and the library already has a translation store; the alternative
> was inventing a translations table beside it. Also reorders the file: the 1.2
> changelog inserted §7 *before* §6, so the sections read 5, 7, 6. They now run
> 1–8 in order. No content changed in that move — cross-references by number
> were already correct and stay correct.

> **Changelog — 1.2.** Adds §7 — `error.ErrDataLocked`. This error was present in
> the library clone all along but was missing from the survey, so the original
> architecture pass (v3) designed a `version` column instead of using the
> library's concurrency error. The design has since been corrected to the
> `updated_at`-compare convention; this section records the error so the next
> pass does not re-miss it.

> **Changelog — 1.1.** §1 previously stated the `uuid_key` *and* `deleted`
> requirements were "both on `teachers` and `teacher_education_histories`". That
> was wrong: only `uuid_key` is on both. `deleted` is on `teachers` alone, and
> `teacher_education_histories` has none — it hard-deletes. The library
> requirement itself is unchanged; the sentence about where it was applied was
> wrong, and `database.md` had it right. Corrected in §1 and flagged there.

## Why this document exists

FEAT-001 is the first feature in an empty `sample-project` repo. The
architecture pass applies the reuse-first ladder (does it exist here → shared
lib → stdlib → new code) one level up, at the design-decision level — so this
is the record of *what already existed in `nexcommon`* before anything new was
proposed.

Every reuse decision in `architecture.md` cites a line from this survey. It is
kept as a source document so `/go-dev` and `/qc` can verify a shared-lib claim
against the library itself, rather than taking the architecture doc's word for
it.

The library, not this survey, is authoritative. If they ever disagree, re-read
the clone.

---

## 1. Audit logging — reused, not built

**Location:** `services/audit_helper/` (`type.go`, `audit_helper.go`,
`audit_builder.go`, `default.go`)

FEAT-001's success criterion 16 requires exactly one audit entry per successful
update, recording before and after values. The spec's Dependencies section notes
"no log schema is specified". A bespoke `teacher_audit_logs` table was the
obvious build — it is not needed.

What the library provides:

```go
func NewAuditHelper(
    enable bool,
    db *sql.DB,
    nats nats.JetStreamContext,
    subject string,
    defaultSchema string,
) AuditHelper

func (a *auditHelper) InitAuditService(
    ctx *context.ContextModel,
    inputStruct interface{},
    serveFunction ServiceFunction,
) (*auditBuilder, error)
```

Mechanics, read from `audit_builder.go`:

- `InitAuditService` opens a `sql.Tx` and hands it to your service function
  (`ServiceFunction func(ctx, tx, input, time) (interface{}, []AuditSystemModel, error)`).
- The before-state snapshot is taken by `GetDataForAuditByIDTx` →
  `GetDataForAuditTx`, which runs:
  `SELECT a.id, uuid_key, row_to_json(a) FROM (SELECT * FROM <schema>.<table> WHERE <filters> FOR UPDATE) a`
- Audit rows are **published to a NATS subject**, not inserted into a table —
  `pushAuditToMessageBroker` → `ab.publisher.nats.Publish(subject, jsonData)`.
- `AuditSystemModel` carries `TableName`, `PrimaryKey`, `UUIDKey`, `Data`
  (the `row_to_json` snapshot), `Action` (`AuditServiceActionDelete=0`,
  `Insert=1`, `Update=2`), `SchemaName`, `CreatedBy`, `CreatedClient`, `CreatedAt`.

### Two consequences that shape our schema

1. **Every audited table needs a `uuid_key` column.** The before-snapshot query
   selects it by literal name. A table without it fails at runtime.
2. **Every audited table needs a `deleted` boolean column.**
   `GetDataForAuditByIDTx` filters `deleted = false` for non-delete actions.

Both requirements were applied to `teachers`, because `teachers` is the only
table FEAT-001 registers for auditing. It is the only table carrying `deleted`;
`uuid_key` is on all three tables, but on `teacher_education_histories` and
`institution_levels` it is house convention rather than a library requirement —
nothing reads it there.

**`teacher_education_histories` deliberately has no `deleted` column** and
**must not be registered with `audit_helper` as the schema stands.** The
before-snapshot query filters `deleted = false` by literal name against whatever
table it is given, so registering an education row would fail at runtime. If
education rows ever need their own audit trail, the column has to be added
first. (Survey v1.0 said the column was on both tables; that was the error this
changelog fixes.)

### Durability caveat — recorded, accepted

`serviceWithAudit` calls `go ab.pushAuditToMessageBroker(...)` — a **goroutine,
fire-and-forget**. `nats.Publish` (not a JetStream ack-required publish) failing
only logs:

```go
_, err = ab.publisher.nats.Publish(ab.publisher.subject, jsonData)
if err != nil {
    log.Error()...Msg("Error Found when publishing data audit")
}
```

So criterion 16's "exactly one audit entry" is **at-most-once**, and the entry
does not exist anywhere until a separate NATS consumer persists it. This was
raised with the user and reuse was chosen anyway — see `architecture.md`. If
audit must become durable, this is the decision to revisit, and the fallback is
a local audit table written inside the same transaction.

---

## 2. Auth — reused, and it is a no-op by design

**Location:** `controller/whitelist.go`

```go
func (cv ControllerValidator) WhitelistValidator(ctx context.Context, header map[string]string) error {
    return nil
}
```

`WhitelistValidator` performs **no validation at all**. It is nexcommon's
declared idiom for "this route has no authentication" — `controller/type.go`
defines `WHITELIST_TOKEN_VALIDATOR = "${WHITELIST}"`.

This matches FEAT-001's spec exactly: no teacher accounts, no administrator
distinction, no role model. It is not a stopgap auth mechanism — it is the
explicit statement that auth is absent.

Related validators found alongside it, for contrast: `FixedTokenValidator`
(compares a static token), `FixedTokenValidatorWithKong`, `InternalAccessValidator`,
`UserTokenValidator`, `UserAccessValidatorWithKong`.

**Consequence for audit:** with no auth, nothing populates
`ctx.AuthAccessTokenModel.ResourceUserID`, so `AuditSystemModel.CreatedBy` and
`CreatedClient` are empty. Audit entries for FEAT-001 record *what changed*, not
*who changed it*. That is an accepted consequence of the deferral, not a defect.

---

## 3. Regex — reused, and it resolves the spec's open question 2

**Location:** `regex/regex.go`

The spec's Contact fields section and Open question 2 say the phone pattern
should be sourced from the implementation rather than reinvented, and the user
noted "the development side already holds it". The pattern is not in the
(empty) service repo — it is in the shared library:

```go
const PHONE_NUMBER_WITH_COUNTRY_CODE = "[+][0-9]+[-][1-9][0-9]{8,12}$"
const FORMAT_PHONE_NUMBER_ND         = "^(08|\\+628|\\+62-8|628)(\\d{10,11})$"
const EMAIL_REGEX                    = "^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"
```

**Selected: `PHONE_NUMBER_WITH_COUNTRY_CODE`** (user decision, this session).

It requires a leading `+` and a literal hyphen — `+62-81234567890` passes,
`081234567890` does **not**. The user was shown this consequence explicitly and
confirmed it. Criterion 31 says phone "matches the project's phone-number
regex"; this satisfies it.

Also available and unused by this feature, listed so they are not rediscovered:
`COUNTRY_CODE`, `IPV4_REGEX`, `NIK`, `NPWP`, `USERNAME`, `NAME_STANDARD`,
`TEXT_ONLY`. Note `NAME_STANDARD = "^[A-Z][a-z]+(([ ][A-Z][a-z])?[a-z]*)*$"` is
**too strict** for `first_name`/`last_name` here — it permits only a
capitalised-lowercase shape, so `MARY-JANE` or `O'Brien` would fail. FEAT-001
uses plain length validation (2–50) per spec criteria 4, not this constant.

---

## 4. HTTP / validation / DTO scaffolding — reused

**Location:** `http/`, `controller/`, `dto/`

The standard endpoint shape, confirmed in `nexcommon-go-standards` and the
library's own `nexcommon.md`:

```go
httpController.WrapService(
    http.NewWarpServiceParam(service, serviceMethod, httpController.WhitelistValidator()).
        APIScope("write").
        PathParams("id").
        DTOOut(http.NewDTOParam(outputDTO)),
)
```

List/count endpoints use `WrapServiceListData` with
`NewWarpGetListServiceParam`, `DefaultDataOrdering("id")`, `DTOOut(...)`.
Get-list filtering/ordering is handled by `dto/in.GetListRequest` + the
validator — filter operators `in`, `lk`, `rng`, `eq`, `be`, `lq`; connector
syntax `name||description&&title`. **Do not hand-parse filters.**

Request flow included free by the controller (nothing to reimplement in a
service): ContextModel → validator → permission/API-scope checks → body decode +
tag validation → path params/headers → idempotency start → call service →
idempotency finish → response formatting → logging.

There is a **third** wrapper, `WrapServiceCountData`, for count-only endpoints.
Not used by FEAT-001 — pagination needs the list + count pair, which
`WrapServiceListData` covers.

---

## 5. DDL / migration style — matched, not invented

**Location:** `limited_scheduler/queue/db_queue_migration.sql`,
`db_queue_v2_migration.sql`

The house migration format is `sql-migrate`: a `-- +migrate Up` header, a
`-- +migrate StatementBegin` block, and this table idiom —

```sql
CREATE SEQUENCE IF NOT EXISTS <table>_pkey_seq;
CREATE TABLE IF NOT EXISTS "<table>" (
    id          BIGINT NOT NULL DEFAULT nextval('<table>_pkey_seq'::regclass),
    ...
    created_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP WITHOUT TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_<compact>_id PRIMARY KEY (id),
    CONSTRAINT uq_<compact>_<cols> UNIQUE (...)
);
CREATE INDEX IF NOT EXISTS idx_<compact>_<col> ON <table> (<col>);
```

Sequence-backed `BIGINT id`, compact constraint names (`pk_dbqueuev2_id`),
`idx_<compact>_<col>` index names. `database.md`'s DDL follows this shape.

Note the library's own migrations are **simpler** than the project convention —
they carry no `uuid_key` and no audit columns, because `db_queue` is not itself
audited. The `uuid_key`/`deleted` requirement comes from `audit_helper`
(§1), not from this migration style.

---

## 6. Things checked and deliberately *not* used

| Found | Left alone because |
| :--- | :--- |
| `services/housekeeping_view` | No housekeeping/orphan cleanup in scope — no hard delete exists (criterion 19). |
| `dto/in.GetMultipartDTO`, file-upload path | FEAT-001 has no file fields. |
| `services/scheduler`, `limited_scheduler` | No scheduled work in this feature. |
| `dao.GetListDataDAO` multi-database / sharding / Mongo | Single Postgres, single schema. The composition root should still be *checked* for a second (`PostgresqlView`) section before wiring, per `nexcommon-go-data-standards`. |
| `error2.NewUnBundledErrorMessages` converter pattern | Now used — `ErrDataLocked` (§7) and the field-level validation errors (criteria 3, 4, 5, 28) all flow through it. Flag was resolved by the v10 design pass. |
| `regex.NAME_STANDARD` | Too strict for the spec's name rules — see §3. |

---

## 7. Optimistic-lock error — reused, not built

**Location:** `error/message.go:18`, `error/type.go`

```go
var ErrDataLocked = NewUnBundledErrorMessages(400, errors.New("E-4-CMD-DTO-007"), errFieldNameConverter)
```

FEAT-001's success criteria 35–36 require a stale write to be rejected
server-side. The zero-rows-affected case of a conditional `UPDATE` needs to
surface as a real, actionable error the client understands — not a generic 500.
nexcommon already defines that error: `ErrDataLocked`, HTTP 400, code
`E-4-CMD-DTO-007`, using `errFieldNameConverter` (the same converter the other
DTO field errors use — it takes the field name as its param, so the error names
the offending field).

**This is the team's optimistic-concurrency error.** The convention it implies:
lock by comparing the `updated_at` the client read against the row's current
`updated_at` in a conditional `UPDATE`; zero rows affected → return
`ErrDataLocked`. **No `version` column** — the library's error exists for this
pattern, and adding a `version` column would be a second mechanism alongside it.

**Where it is used:** `database.md` A5 / the `teachers` column notes, and
`architecture.md` C35. A previous version of the design (v3) proposed a
`version INTEGER` column because this error was missed in the survey; v4 drops it
in favour of the `updated_at` compare + `ErrDataLocked`.

**Caveat — this survey only covers the local clone.** `ErrDataLocked` is present
at `C:\Users\Pongo\Documents\github\nexcommon\error\message.go` but **not** in
the public `github.com/william22913/common` mirror, which has only the six DTO
errors (`ErrUnauthorized`, `ErrReservedValueString`, `ErrEmptyField`,
`ErrUnknownData`, `ErrFormatFieldRule`, `ErrFormatField`). If
`github.com/nexsoft-git/nexcommon` (the `go.mod` dependency) ever diverges from
this local clone, re-confirm `ErrDataLocked`'s code/status there before relying
on it.

---

## 8. i18n bundles — reused, not built

**Location:** `bundles/bundles.go`, `bundles/type.go`; dictionaries under
`i18n/common/{constanta,error}/{en-US,id-ID}.json`

```go
func NewBundles(rootDir string, defaultLanguage string) (Bundles, error)

func (b bundles) ReadMessageBundle(
    bundleName string,
    messageID  string,
    language   string,
    param      map[string]interface{},
) (output string)
```

FEAT-001 needs each seeded institution level to carry an English and an
Indonesian label. The obvious build is a `institution_level_translations` table;
it is not needed. The library already has a translation store, and it is
**file-based, not row-based**.

What the loader actually does, read from `bundles.go`:

- `loadBundleI18N` walks `./<rootDir>/` and treats **every directory under it as
  one bundle**, named by its path segments joined with `.`. So
  `i18n/common/constanta/` is bundle `common.constanta`, and a directory
  `i18n/institution_level/` is bundle `institution_level`. A directory holding no
  files is skipped. The nesting is what produces the dot — there is no registry
  and no config listing bundles, so adding one is creating a directory.
- Each bundle is an `i18n.NewBundle(language.Indonesian)` with
  `json.Unmarshal` registered, and every `.json` file in the directory is loaded
  into it. **The bundle's base language is hard-coded to Indonesian** in the
  library — `defaultLanguage` does not set it, and only decides what
  `ReadMessageBundle` falls back to when its `language` argument is empty.
- The JSON files are **flat `KEY: "Label"` maps**, one file per locale, with the
  locale in the filename (`en-US.json`, `id-ID.json`). The **message IDs are the
  keys** — which is what lets a lookup table's stored code double as the message
  ID, so the two vocabularies cannot drift apart.
- `ReadMessageBundle` **`recover()`s on panic and returns the `messageID`**. A
  missing bundle, missing file, or missing key therefore degrades to the raw key
  rather than erroring — there is no error return to check, and the failure is
  silent by construction. That is benign for a label (the response falls back to
  `HIGH_SCHOOL`) but means a content gap will not surface on its own; hence the
  seed/dictionary agreement gate in `database.md`.
- A `param` map is passed through as `TemplateData` for messages with
  placeholders. FEAT-001's level labels have none.

**Who consumes it today:** only the error layer. `error/formator.go:116` and
`error/type.go:73` resolve `common.constanta` to turn a field name into a human
label, and `common.error` to turn an error code into a message. So in the library
as it stands, the bundles are the *error- and field-label* translation store.

**Where the language comes from — and why it matters here.** `ReadMessageBundle`
takes `language` as an argument, and the only thing in the library that populates
it does so from the **auth token**: `controller/user_access.go:160` sets
`_ctx.AuthAccessTokenModel.Locale = tokenModel.Locale` (and
`internal_access.go:87` likewise). There is no `Accept-Language` / `X-Locale`
header parsing anywhere in the library, and no locale on `ContextModel` outside
the auth model. FEAT-001 uses `WhitelistValidator` (§2), so no token is parsed,
`Locale` stays empty, and every request resolves against the `NewBundles`
default. That is recorded as an open item in `database.md` — the selection
mechanism is a spec-level decision, not one `/architect` should invent.

**Test-usage signal for the intended defaults:** every call site in the library's
own tests is `bundles.NewBundles("../../i18n", "id-ID")` — `id-ID` is both the
repo's bundle base language and the tests' default, which is why the FEAT-001
design uses the same default.

**Where it is used:** `architecture.md` A6, and `database.md`'s level-label
dictionary section.

