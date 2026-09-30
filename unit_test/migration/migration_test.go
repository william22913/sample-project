// Package migration_test guards the two failure modes in the initial schema
// that do NOT surface at runtime:
//
//   - Gate 1.13: a seeded institution level with no i18n label (or a label with
//     no seeded row). ReadMessageBundle recover()s to the message ID on a
//     missing key, so a content gap degrades silently to returning the raw code
//     instead of erroring - this test is the only place it surfaces.
//   - Gate 0.4: an unbalanced StatementBegin/StatementEnd around the plpgsql
//     trigger body. sql-migrate splits statements on semicolons, so a missing
//     StatementEnd breaks the migration on its first real run. database.md
//     records that a previous revision of the design shipped exactly this bug.
package migration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

const (
	migrationFile = "sql_migrations/01_init_schema.sql"
	bundleDir     = "i18n/institution_level"
)

// repoRoot walks up from the test's working directory until it finds the
// sql_migrations directory, so the test is independent of how deep the
// unit_test/ tree nests it.
func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "sql_migrations")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not locate repo root (no sql_migrations dir found above cwd)")
		}
		dir = parent
	}
}

func readFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// seededCodes extracts the institution_levels seed values from the INSERT
// statement, without executing SQL and without a database.
func seededCodes(t *testing.T, sql string) map[string]bool {
	t.Helper()
	insert := regexp.MustCompile(`(?s)INSERT INTO institution_levels \(name\) VALUES(.*?)ON CONFLICT`)
	m := insert.FindStringSubmatch(sql)
	if m == nil {
		t.Fatal("could not find the institution_levels seed INSERT in the migration")
	}
	values := regexp.MustCompile(`'([A-Z_]+)'`).FindAllStringSubmatch(m[1], -1)
	if len(values) == 0 {
		t.Fatal("seed INSERT found but contained no quoted values")
	}
	codes := make(map[string]bool, len(values))
	for _, v := range values {
		codes[v[1]] = true
	}
	return codes
}

func dictKeys(t *testing.T, root, locale string) map[string]bool {
	t.Helper()
	raw := readFile(t, root, filepath.Join(bundleDir, locale+".json"))
	var parsed map[string]string
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("parse %s.json: %v", locale, err)
	}
	keys := make(map[string]bool, len(parsed))
	for k, v := range parsed {
		if v == "" {
			t.Errorf("%s.json: key %q has an empty label", locale, k)
		}
		keys[k] = true
	}
	return keys
}

// TestSeedAndDictionaryAgree is Gate 1.13: both directions must match - every
// seeded code has a label in both locale files, and no file carries a key with
// no seeded row.
func TestSeedAndDictionaryAgree(t *testing.T) {
	root := repoRoot(t)
	seeded := seededCodes(t, readFile(t, root, migrationFile))

	if len(seeded) != 7 {
		t.Errorf("expected the seven authoritative codes seeded, got %d: %v", len(seeded), seeded)
	}

	for _, locale := range []string{"en-US", "id-ID"} {
		keys := dictKeys(t, root, locale)

		for code := range seeded {
			if !keys[code] {
				t.Errorf("%s.json is missing a label for seeded code %q - at runtime "+
					"ReadMessageBundle would silently return the raw code %q", locale, code, code)
			}
		}
		for key := range keys {
			if !seeded[key] {
				t.Errorf("%s.json carries key %q which is not a seeded institution_levels.name", locale, key)
			}
		}
	}
}

// TestStatementMarkersBalanced is Gate 0.4: sql-migrate splits on semicolons,
// so the $$-quoted plpgsql body must be wrapped by a matching
// StatementBegin/StatementEnd pair. A lone StatementBegin breaks the migration.
func TestStatementMarkersBalanced(t *testing.T) {
	root := repoRoot(t)
	sql := readFile(t, root, migrationFile)

	begins := len(regexp.MustCompile(`-- \+migrate StatementBegin`).FindAllString(sql, -1))
	ends := len(regexp.MustCompile(`-- \+migrate StatementEnd`).FindAllString(sql, -1))

	if begins != ends {
		t.Errorf("unbalanced migrate statement markers: %d StatementBegin vs %d StatementEnd "+
			"- the migration will fail when sql-migrate splits the trigger body", begins, ends)
	}
	if begins == 0 {
		t.Error("no StatementBegin marker found, but the migration defines a $$-quoted function body")
	}

	up := len(regexp.MustCompile(`-- \+migrate Up`).FindAllString(sql, -1))
	if up != 1 {
		t.Errorf("expected exactly one '-- +migrate Up' header, found %d", up)
	}
}

// TestAsciiOnlyInDictionaries pins the spec's field-constraint rule that text
// is ASCII-only in this phase (criterion 37) - a non-ASCII label would be a
// deliberate exception, not an accident.
func TestAsciiOnlyInDictionaries(t *testing.T) {
	root := repoRoot(t)
	for _, locale := range []string{"en-US", "id-ID"} {
		raw := readFile(t, root, filepath.Join(bundleDir, locale+".json"))
		for i := 0; i < len(raw); i++ {
			if raw[i] > 127 {
				t.Errorf("%s.json contains a non-ASCII byte at offset %d; level labels are ASCII-only this phase", locale, i)
				break
			}
		}
	}
}
