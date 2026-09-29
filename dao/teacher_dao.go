package dao

import (
	"context"
	"database/sql"
	"time"

	nexctx "github.com/nexsoft-git/nexcommon/context"
	nexdao "github.com/nexsoft-git/nexcommon/dao"
	nexin "github.com/nexsoft-git/nexcommon/dto/in"
	"github.com/nexsoft-git/nexcommon/model"

	"sample-project/repository"
)

// TeacherDAO is the only writer of the teachers table.
//
// Note what is absent: there is no Delete method, and there never will be.
// Spec criterion 19 requires hard delete to be unreachable, "not merely
// discouraged" - so deactivation is a status write and no code path can remove
// a teacher row. Do not add one.
type TeacherDAO struct {
	Db *sql.DB
}

func NewTeacherDAO(db *sql.DB) TeacherDAO {
	return TeacherDAO{Db: db}
}

// teacherColumns is spelled once so the SELECT list and the Scan order cannot
// drift apart - they are matched positionally and a mismatch fails at runtime,
// not at compile time.
const teacherColumns = `id, uuid_key, teacher_code, first_name, last_name, email, phone,
	hire_date, status, created_by, updated_by, created_at, updated_at`

// rowScanner is the one method *sql.Row shares with nothing else we need, so a
// single-method interface is enough to scan through either a plain read or a
// read inside a transaction.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

// scanTeacher treats "no row" as a found-but-empty result rather than an
// error: callers check ID.Valid to distinguish absent from present, which keeps
// ErrNoRows from leaking out of the data layer as something a service must
// remember to translate.
func scanTeacher(row rowScanner) (result repository.TeacherModel, err error) {
	err = row.Scan(
		&result.ID, &result.UUIDKey, &result.TeacherCode, &result.FirstName, &result.LastName,
		&result.Email, &result.Phone, &result.HireDate, &result.Status,
		&result.CreatedBy, &result.UpdatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// Insert must run inside the caller's transaction so the teacher row and its
// education rows commit or roll back together (spec criterion 7).
//
// teacher_code is deliberately absent from the column list: the
// trg_teachers_setcode BEFORE INSERT trigger derives it from hire_date and the
// sequence-assigned id. The DEFAULT on id is applied before a BEFORE trigger
// runs, so NEW.id is populated by the time the trigger reads it.
func (d TeacherDAO) Insert(ctx context.Context, tx *sql.Tx, m repository.TeacherModel) (result repository.TeacherModel, err error) {
	query := `INSERT INTO teachers
		(first_name, last_name, email, phone, hire_date, created_by, updated_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING ` + teacherColumns

	result, err = scanTeacher(tx.QueryRowContext(ctx, query,
		m.FirstName, m.LastName, m.Email, m.Phone, m.HireDate, m.CreatedBy, m.UpdatedBy,
	))

	return
}

func (d TeacherDAO) GetByID(ctx context.Context, id int64) (result repository.TeacherModel, err error) {
	ctx, cancel := withQueryTimeout(ctx)
	defer cancel()

	query := `SELECT ` + teacherColumns + ` FROM teachers WHERE id = $1`
	return scanTeacher(d.Db.QueryRowContext(ctx, query, id))
}

func (d TeacherDAO) GetByUUIDKey(ctx context.Context, uuidKey string) (result repository.TeacherModel, err error) {
	ctx, cancel := withQueryTimeout(ctx)
	defer cancel()

	query := `SELECT ` + teacherColumns + ` FROM teachers WHERE uuid_key = $1`
	return scanTeacher(d.Db.QueryRowContext(ctx, query, uuidKey))
}

// teacherListQuery is the list's base SELECT. No table alias: nexcommon's
// get-list builder renders a filter key as a bare column expression (its
// ParamPrefix is empty unless a caller adds to it), and the query joins
// nothing, so bare names resolve.
//
// The trailing space is not cosmetic. The builder appends its WHERE clause to
// this string with no separator of its own, so without it the statement reads
// `FROM teachersWHERE ...` and Postgres rejects it as one long identifier.
// Unlike the other DAOs here, this statement is not fully spelled out at the
// call site and cannot be - the filters arrive as data.
const teacherListQuery = `SELECT ` + teacherColumns + ` FROM teachers `

// GetListTeacher runs the get-list flow against nexcommon's shared builder
// rather than hand-rolling a WHERE clause.
//
// That is not just convention: the builder is what turns a model.SearchParam
// into parameterized SQL, including the `lk` operator's LOWER(column) LIKE $n
// (criteria 9 and 10) and the offset/limit pair whose default the validator has
// already clamped to 20 by the time this runs (criterion 12). Rebuilding any of
// it here would be a second, unparameterized copy of the same logic.
//
// searchBy arrives already rewritten by the service - see
// expandActiveFilter - so this layer only has to not interfere with it.
//
// One thing this read does NOT get, unlike every other read in this package:
// withQueryTimeout. nexcommon's builder runs the statement through
// param.db.Query with no context parameter at all, so there is nothing to
// attach a deadline to. Bounding it would mean not using the builder, which is
// what supplies the filters. Recorded here rather than silently dropped.
func (d TeacherDAO) GetListTeacher(ctx *nexctx.ContextModel, dto nexin.GetListRequest, searchBy []model.SearchParam) (result []interface{}, err error) {
	getListDAO := nexdao.GetListDataDAO{DB: d.Db}

	param, err := getListDAO.NewGetListDataParam(ctx, teacherListQuery, dto, searchBy, parseTeacherRow)
	if err != nil {
		return
	}

	return getListDAO.GetListDataWithDefaultMustCheck(param)
}

// parseTeacherRow adapts scanTeacher to the RowsParser signature the get-list
// builder expects - the same column list, so a list row and a detail row cannot
// disagree about how a NULL phone is read.
func parseTeacherRow(rows *sql.Rows) (interface{}, error) {
	return scanTeacher(rows)
}

// GetIDByUUIDKey resolves the external uuid_key to the internal id every FK
// and conditional write needs - the one extra lookup every entity wants beyond
// plain CRUD.
func (d TeacherDAO) GetIDByUUIDKey(ctx context.Context, uuidKey string) (id sql.NullInt64, err error) {
	ctx, cancel := withQueryTimeout(ctx)
	defer cancel()

	query := `SELECT id FROM teachers WHERE uuid_key = $1`
	err = d.Db.QueryRowContext(ctx, query, uuidKey).Scan(&id)
	if err == sql.ErrNoRows {
		err = nil
	}

	return
}

// Update is the optimistic lock (architecture A5). A matched row comes back
// through RETURNING; a WHERE that matched nothing comes back as an invalid ID
// rather than an error, the same found-but-empty convention the reads use - and
// that is the signal the service turns into ErrDataLocked. An omitted
// expected_updated_at matches nothing either, so the check cannot be bypassed
// by leaving the field out (criterion 36).
//
// RETURNING rather than RowsAffected plus a second read, because the caller
// needs the *new* updated_at to hand back as the next lock value (criterion 25
// treats these values as exact). The value is CURRENT_TIMESTAMP, which only the
// database knows; re-reading it on the pool would race the uncommitted
// transaction and see the old row.
//
// teacher_code and hire_date are not in the SET list. Criterion 15 requires an
// attempt to edit them to be an explicit rejection rather than a silent no-op,
// which the service enforces - see update_teacher.go.
func (d TeacherDAO) Update(ctx context.Context, tx *sql.Tx, id int64, m repository.TeacherModel, expectedUpdatedAt time.Time) (result repository.TeacherModel, err error) {
	query := `UPDATE teachers SET
			first_name = $1,
			last_name  = $2,
			email      = $3,
			phone      = $4,
			status     = $5,
			updated_by = $6,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $7 AND updated_at = $8
		RETURNING ` + teacherColumns

	return scanTeacher(tx.QueryRowContext(ctx, query,
		m.FirstName, m.LastName, m.Email, m.Phone, m.Status, m.UpdatedBy,
		id, expectedUpdatedAt,
	))
}

// Deactivate is a status write, not a delete - the row and every column but
// status/updated_* survive (criterion 18).
//
// The `status <> 'INACTIVE'` predicate makes Inactive terminal on this path too
// (criterion 21): a second deactivate matches nothing, exactly as a transition
// out of Inactive does in Update.
func (d TeacherDAO) Deactivate(ctx context.Context, tx *sql.Tx, id int64, updatedBy sql.NullInt64, expectedUpdatedAt time.Time) (result repository.TeacherModel, err error) {
	query := `UPDATE teachers SET
			status     = 'INACTIVE',
			updated_by = $1,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND updated_at = $3 AND status <> 'INACTIVE'
		RETURNING ` + teacherColumns

	return scanTeacher(tx.QueryRowContext(ctx, query, updatedBy, id, expectedUpdatedAt))
}

// CheckDuplicateEmail answers "is this email already taken" as a count, for
// the duplicate-field error criterion 3 asks for. It runs inside the caller's
// transaction so the check and the write it guards see one snapshot.
//
// It is a pre-check, not the enforcement. uq_teachers_email_lower on
// lower(email) is what actually makes email unique (criterion 30) - two
// concurrent inserts can both pass this query, and the loser then fails the
// index. That path returns a generic 500, which is honest: it is a genuine
// conflict the caller did not cause and cannot fix, not the ordinary
// "you picked a taken email" case this query exists to report.
//
// excludeID skips the row being updated, so re-saving a teacher's own email is
// not a duplicate. Pass 0 when inserting.
func (d TeacherDAO) CheckDuplicateEmail(ctx context.Context, tx *sql.Tx, email string, excludeID int64) (count sql.NullInt64, err error) {
	query := `SELECT count(id) FROM teachers WHERE lower(email) = lower($1) AND id <> $2`
	err = tx.QueryRowContext(ctx, query, email, excludeID).Scan(&count)

	return
}
