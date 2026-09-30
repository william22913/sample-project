-- +migrate Up
-- FEAT-001 initial schema. One bundled migration: teacher_education_histories
-- foreign-keys both teachers and institution_levels, so all three must land
-- together (database.md "Migration plan"). Statement order below is forced by
-- the dependency graph.

-- Step 1: CREATE EXTENSION pgcrypto is NOT needed - gate 0.1 (database.md).
-- gen_random_uuid() is built into PostgreSQL 13+, which is the floor this
-- service targets. Deliberately omitted rather than left in as a no-op.

-- Step 2: institution_levels - seeded lookup, no CRUD surface (architecture.md A4).
CREATE SEQUENCE IF NOT EXISTS institution_levels_pkey_seq;
CREATE TABLE IF NOT EXISTS "institution_levels" (
    id                      BIGINT NOT NULL     DEFAULT nextval('institution_levels_pkey_seq'::regclass),
    uuid_key                UUID NOT NULL       DEFAULT gen_random_uuid(),
    name                    VARCHAR(50) NOT NULL,
    created_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_institutionlevels_id PRIMARY KEY (id),
    CONSTRAINT uq_institutionlevels_uuidkey UNIQUE (uuid_key),
    CONSTRAINT uq_institutionlevels_name UNIQUE (name)
);

-- Step 3: seed. The seven codes are authoritative (database.md "Seed data").
-- They are also the i18n message IDs in i18n/institution_level/*.json, keyed
-- on the code rather than on id, so the two cannot drift.
INSERT INTO institution_levels (name) VALUES
    ('PRESCHOOL'),
    ('PRIMARY_SCHOOL'),
    ('MIDDLE_SCHOOL'),
    ('HIGH_SCHOOL'),
    ('BACHELOR'),
    ('MASTER'),
    ('DOCTOR')
ON CONFLICT (name) DO NOTHING;

-- Step 4: teachers.
CREATE SEQUENCE IF NOT EXISTS teachers_pkey_seq;
CREATE TABLE IF NOT EXISTS "teachers" (
    id                      BIGINT NOT NULL     DEFAULT nextval('teachers_pkey_seq'::regclass),
    uuid_key                UUID NOT NULL       DEFAULT gen_random_uuid(),
    teacher_code            VARCHAR(20) NOT NULL,
    first_name              VARCHAR(50) NOT NULL,
    last_name               VARCHAR(50) NOT NULL,
    email                   VARCHAR(50) NOT NULL,
    phone                   VARCHAR(50) NULL,
    hire_date               DATE NOT NULL,
    status                  VARCHAR(10) NOT NULL DEFAULT 'ACTIVE',
    deleted                 BOOLEAN NOT NULL DEFAULT FALSE,
    created_by              BIGINT NULL,
    updated_by              BIGINT NULL,
    created_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_teachers_id PRIMARY KEY (id),
    CONSTRAINT uq_teachers_uuidkey UNIQUE (uuid_key),
    CONSTRAINT uq_teachers_teachercode UNIQUE (teacher_code),
    CONSTRAINT ck_teachers_status CHECK (status IN ('ACTIVE', 'ON_LEAVE', 'INACTIVE'))
);

-- Case-insensitive uniqueness (database.md N1). This REPLACES a plain
-- UNIQUE (email): a plain constraint is case-sensitive in PostgreSQL, so
-- Alice@school.edu and alice@school.edu were two insertable rows. Lookups and
-- the duplicate pre-check must therefore be written
-- `WHERE lower(email) = lower($1)` to use this index.
CREATE UNIQUE INDEX IF NOT EXISTS uq_teachers_email_lower
    ON teachers (lower(email));

CREATE INDEX IF NOT EXISTS idx_teachers_status ON teachers (status);

COMMENT ON COLUMN teachers.deleted IS
    'Never set to true. Present only because nexcommon audit_helper filters on it. Use status = ''INACTIVE'' for deactivation.';

-- Step 5: teacher_code generation. The plpgsql body contains semicolons inside
-- $$ ... $$, so it MUST sit between StatementBegin/StatementEnd or sql-migrate
-- splits it (database.md gate 0.4). Both halves are required.
-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION set_teacher_code() RETURNS trigger AS $$
BEGIN
    IF NEW.teacher_code IS NULL OR NEW.teacher_code = '' THEN
        NEW.teacher_code := 'TCH-'
            || to_char(NEW.hire_date, 'YYYY')
            || '-'
            || lpad(NEW.id::text, 3, '0');
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +migrate StatementEnd

-- CREATE TRIGGER has no IF NOT EXISTS, so the DROP precedes it for re-run safety.
DROP TRIGGER IF EXISTS trg_teachers_setcode ON teachers;
CREATE TRIGGER trg_teachers_setcode
    BEFORE INSERT ON teachers
    FOR EACH ROW EXECUTE FUNCTION set_teacher_code();

-- Step 6: teacher_education_histories. Both FK targets now exist. Note the
-- deliberate absence of a `deleted` column: these rows hard-delete (spec
-- criterion 27), and the table must NOT be registered with audit_helper as the
-- schema stands - its before-snapshot query filters `deleted = false` by
-- literal name and would fail at runtime.
CREATE SEQUENCE IF NOT EXISTS teacher_education_histories_pkey_seq;
CREATE TABLE IF NOT EXISTS "teacher_education_histories" (
    id                      BIGINT NOT NULL DEFAULT nextval('teacher_education_histories_pkey_seq'::regclass),
    uuid_key                UUID NOT NULL DEFAULT gen_random_uuid(),
    teacher_id              BIGINT NOT NULL,
    institution_level_id    BIGINT NOT NULL,
    institution             VARCHAR(100) NOT NULL,
    study_start_date        DATE NOT NULL,
    study_end_date          DATE NOT NULL,
    score                   NUMERIC(5,2) NOT NULL,
    focused_subject         VARCHAR(100) NOT NULL,
    created_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP WITHOUT TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_teachereducationhistories_id PRIMARY KEY (id),
    CONSTRAINT uq_teachereducationhistories_uuidkey UNIQUE (uuid_key),
    CONSTRAINT fk_teachereducationhistories_teacher FOREIGN KEY (teacher_id)
        REFERENCES teachers (id),
    CONSTRAINT fk_teachereducationhistories_level FOREIGN KEY (institution_level_id)
        REFERENCES institution_levels (id),
    CONSTRAINT ck_teachereducationhistories_score CHECK (score >= 0 AND score <= 100),
    CONSTRAINT ck_teachereducationhistories_daterange CHECK (study_end_date >= study_start_date)
);

-- No unique business key on education rows: overlapping date ranges are
-- permitted, including two rows at the same institution, so
-- (teacher_id, institution, study_start_date) must NOT be declared unique.

CREATE INDEX IF NOT EXISTS idx_teachereducationhistories_teacherid
    ON teacher_education_histories (teacher_id);
CREATE INDEX IF NOT EXISTS idx_teachereducationhistories_teacherid_enddate
    ON teacher_education_histories (teacher_id, study_end_date DESC);

-- Note: criterion 29 (study_end_date not in the future) has NO constraint here
-- and cannot have one - PostgreSQL requires CHECK expressions to be immutable
-- and CURRENT_DATE/now() are only stable. It is enforced in the service layer.

-- +migrate Down
DROP TABLE IF EXISTS teacher_education_histories;
DROP SEQUENCE IF EXISTS teacher_education_histories_pkey_seq;
DROP TRIGGER IF EXISTS trg_teachers_setcode ON teachers;
DROP FUNCTION IF EXISTS set_teacher_code();
DROP TABLE IF EXISTS teachers;
DROP SEQUENCE IF EXISTS teachers_pkey_seq;
DROP TABLE IF EXISTS institution_levels;
DROP SEQUENCE IF EXISTS institution_levels_pkey_seq;
