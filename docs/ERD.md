# sample-project — Entity Relationship Diagram

3 of 3 tables registered — full detail, single ungrouped diagram (below the split threshold).

This is the cumulative entity-relationship picture across every feature built for this service. It is maintained by `/architect` as part of each feature's own PR — new or changed tables are merged in as features land. It is not a substitute for a feature's own `database.md`: that document holds the design rationale, the normalization findings and the migration plan; this is the combined picture of every table the service has.

```mermaid
erDiagram
    teachers ||--o{ teacher_education_histories : "owns"
    institution_levels ||--o{ teacher_education_histories : "classifies"

    teachers {
        bigint    id           PK "seq teachers_pkey_seq"
        uuid      uuid_key     UK "NOT NULL DEFAULT gen_random_uuid()"
        varchar   teacher_code UK "VARCHAR(20) NOT NULL; TCH-<hire_year>-<id>; trigger-assigned; immutable"
        varchar   first_name      "VARCHAR(50) NOT NULL; 2-50; lower bound is app-only"
        varchar   last_name       "VARCHAR(50) NOT NULL; 2-50; lower bound is app-only"
        varchar   email           "VARCHAR(50) NOT NULL; unique on lower(email); stored lowercased"
        varchar   phone           "VARCHAR(50) NULL; +CC-... form only"
        date      hire_date       "NOT NULL; immutable per criterion 15"
        varchar   status          "VARCHAR(10) NOT NULL DEFAULT ACTIVE; CHECK ACTIVE|ON_LEAVE|INACTIVE"
        boolean   deleted         "NOT NULL DEFAULT FALSE; audit_helper filter; never set true"
        bigint    created_by      "NULL; always NULL this phase (no auth)"
        bigint    updated_by      "NULL; always NULL this phase (no auth)"
        timestamp created_at      "NOT NULL DEFAULT CURRENT_TIMESTAMP"
        timestamp updated_at      "NOT NULL DEFAULT CURRENT_TIMESTAMP; optimistic-concurrency token"
    }

    institution_levels {
        bigint    id          PK "seq institution_levels_pkey_seq"
        uuid      uuid_key    UK "NOT NULL DEFAULT gen_random_uuid(); convention; nothing reads it"
        varchar   name        UK "VARCHAR(50) NOT NULL UNIQUE; stable code, not a label - PRESCHOOL|PRIMARY_SCHOOL|MIDDLE_SCHOOL|HIGH_SCHOOL|BACHELOR|MASTER|DOCTOR; label via i18n bundle, never stored"
        timestamp created_at     "NOT NULL DEFAULT CURRENT_TIMESTAMP"
        timestamp updated_at     "NOT NULL DEFAULT CURRENT_TIMESTAMP"
    }

    teacher_education_histories {
        bigint    id                   PK "seq teacher_education_histories_pkey_seq"
        uuid      uuid_key             UK "NOT NULL DEFAULT gen_random_uuid(); convention; nothing reads it"
        bigint    teacher_id           FK "NOT NULL; -> teachers.id"
        bigint    institution_level_id FK "NOT NULL; -> institution_levels.id"
        varchar   institution             "VARCHAR(100) NOT NULL; free text by spec"
        date      study_start_date        "NOT NULL"
        date      study_end_date          "NOT NULL"
        numeric   score                   "NUMERIC(5,2) NOT NULL; CHECK 0-100"
        varchar   focused_subject         "VARCHAR(100) NOT NULL; free text by spec"
        timestamp created_at              "NOT NULL DEFAULT CURRENT_TIMESTAMP"
        timestamp updated_at              "NOT NULL DEFAULT CURRENT_TIMESTAMP"
    }
```

**Deliberately absent: the level-label translations.** `institution_levels.name`
stores a code; its English and Indonesian labels live in i18n bundle JSON files
(`i18n/institution_level/{en-US,id-ID}.json`), not in a table. Nothing joins on
or filters by a label, so there is no entity to draw here — its absence is the
design, not an omission. See FEAT-001's `database.md` for the dictionary itself.

## Tables by feature

| Feature | Tables added/changed |
| :--- | :--- |
| [FEAT-001](feature/FEAT-001/database.md) | `institution_levels`, `teachers`, `teacher_education_histories` |

## Cardinality

| Relationship | Cardinality | Note |
| :--- | :--- | :--- |
| `teachers` → `teacher_education_histories` | 1 → 0..n | Zero rows is a deliberate valid state (criterion 13). Each row belongs to exactly one teacher. |
| `institution_levels` → `teacher_education_histories` | 1 → 0..n | Each row names exactly one level (`NOT NULL`). A level may classify zero rows. |

No other relationship exists, and that is deliberate — the spec's non-goals drop
every Subject / Class / Student link. **`teachers` does not carry an
`institution_level_id`**: the level is per-education-row, not per-teacher. A
teacher with three schools has three levels.

Both FKs default to `NO ACTION` — no `ON DELETE` clause is written. That is
implicit rather than chosen: hard-deleting a teacher is unreachable (criterion
19), so no cascade is ever exercised, and an operational `DELETE FROM teachers`
failing loudly is the preferable behaviour.
