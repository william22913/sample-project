package dao_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot walks up from the test's working directory until it finds
// sql_migrations, so the test is independent of how deep unit_test/ nests it.
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

// sourceOf reads a repo-relative file so a test can assert on a SQL clause
// directly.
//
// This is deliberately not done through sqlmock. sqlmock can only be made to
// assert a query's shape by matching it against a regex, and the only honest
// regex is a re-typed copy of the statement - which then asserts that the test
// and the code agree, not that the clause is present. Reading the source
// asserts the clause itself, and it is what makes the negative checks (a
// column must NOT be in a SET list) expressible at all.
func sourceOf(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// queryLiteralAfter returns the body of the raw string literal assigned to
// `query` that contains marker - i.e. the SQL the driver actually receives,
// with the prose around it excluded.
//
// A file-wide regex is not good enough for the assertions below, in both
// directions. Comments explaining a predicate quote it in prose, so a
// file-wide search still finds a clause that has been deleted from the SQL;
// and the same word ("deleted", "$1") recurs in several statements, so a
// file-wide search cannot tell one statement from another. Scoping to the
// literal keeps the assertion about the statement.
//
// The opener is anchored on `:= ` rather than on any backtick because comments
// use backticks for inline code (`status <> 'INACTIVE'`), and an unanchored
// search would happily treat one of those as the start of a string literal.
// The marker itself must therefore be text that appears only in the SQL.
func queryLiteralAfter(t *testing.T, src, marker string) string {
	t.Helper()
	at := strings.Index(src, marker)
	if at < 0 {
		t.Fatalf("marker %q not found in source", marker)
	}
	open := strings.LastIndex(src[:at], ":= `")
	if open < 0 {
		t.Fatalf("no `query := ...` raw string literal opens before %q", marker)
	}
	open += len(":= `")
	rel := strings.Index(src[at:], "`")
	if rel < 0 {
		t.Fatalf("no raw string literal closes after %q", marker)
	}
	return src[open : at+rel]
}

// statementLiteralAfter is queryLiteralAfter's counterpart for a statement
// passed to educationWrite(...) rather than assigned to `query`.
//
// The anchor has to differ. These literals are function arguments, so there is
// no `:= ` in front of them, and the enclosing call is what distinguishes one
// statement from another - a file-wide search for "DELETE FROM" would find
// whichever of the comment prose or the CTE wrapper mentioned it first.
func statementLiteralAfter(t *testing.T, src, marker string) string {
	t.Helper()
	at := strings.Index(src, marker)
	if at < 0 {
		t.Fatalf("marker %q not found in source", marker)
	}
	const anchor = "educationWrite(`"
	open := strings.LastIndex(src[:at], anchor)
	if open < 0 {
		t.Fatalf("no educationWrite(`...`) literal opens before %q", marker)
	}
	open += len(anchor)
	rel := strings.Index(src[at:], "`")
	if rel < 0 {
		t.Fatalf("no raw string literal closes after %q", marker)
	}
	return src[open : at+rel]
}
