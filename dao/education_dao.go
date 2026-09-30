package dao

import (
	"context"
	"database/sql"

	"sample-project/repository"
)

// EducationDAO owns teacher_education_histories.
//
// Every read and write is scoped by teacher_id as well as the row id. That is
// not defensive padding: criterion 27 says deleting a row removes that row
// only, and criterion 28 says an edit aimed at another teacher's row must not
// reach it. Scoping in SQL makes both true by construction rather than by a
// service-layer check somebody can forget.
type EducationDAO struct {
	Db *sql.DB
}

func NewEducationDAO(db *sql.DB) EducationDAO {
	return EducationDAO{Db: db}
}

// educationColumns joins the level name in rather than storing it, so a caller
// gets the code (BACHELOR) and never the surrogate id, and the detail view
// needs no second query per education row. The join is INNER because
// institution_level_id is NOT NULL and carries a foreign key, so it can never
// silently drop a row.
//
// Spelled once because each write returns the row it wrote through the same
// list - see educationWrite below - and a second copy is how the read and the
// write responses would stop agreeing.
const educationColumns = `teh.id, teh.uuid_key, teh.teacher_id, teh.institution_level_id, il.name,
		teh.institution, teh.study_start_date, teh.study_end_date, teh.score,
		teh.focused_subject, teh.created_at, teh.updated_at`

const educationSelect = `SELECT ` + educationColumns + `
	FROM teacher_education_histories teh
	JOIN institution_levels il ON il.id = teh.institution_level_id`

// educationWrite wraps a data-modifying statement so it returns the row it
// wrote, joined to its level code exactly as a read would.
//
// This is one statement rather than a write plus a re-read, and the reason is
// the one that also makes the value correct: the row is uncommitted, so a read
// on the pool would not see it, and a read on the transaction would be a second
// query for values already in hand. It is also why every education write is a
// single-row, single-statement operation that needs no transaction of its own -
// the CTE is atomic by itself.
//
// `written` shadows the table alias so the column list above applies unchanged.
func educationWrite(statement string) string {
	return `WITH written AS (` + statement + ` RETURNING *)
	SELECT ` + educationColumns + `
	FROM written teh
	JOIN institution_levels il ON il.id = teh.institution_level_id`
}

func scanEducation(row rowScanner) (result repository.TeacherEducationHistoryModel, err error) {
	err = row.Scan(
		&result.ID, &result.UUIDKey, &result.TeacherID, &result.InstitutionLevelID,
		&result.InstitutionLevel, &result.Institution, &result.StudyStartDate,
		&result.StudyEndDate, &result.Score, &result.FocusedSubject,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		err = nil
	}
	return
}

// GetByTeacherID returns a teacher's education rows newest-completed first
// (criterion 26). id is the tiebreak so two rows ending on the same date come
// back in a stable order rather than whatever the planner picks - the index
// idx_teachereducationhistories_teacherid_enddate covers exactly this
// ordering.
func (d EducationDAO) GetByTeacherID(ctx context.Context, teacherID int64) (result []repository.TeacherEducationHistoryModel, err error) {
	ctx, cancel := withQueryTimeout(ctx)
	defer cancel()

	query := educationSelect + `
		WHERE teh.teacher_id = $1
		ORDER BY teh.study_end_date DESC, teh.id DESC`

	rows, err := d.Db.QueryContext(ctx, query, teacherID)
	if err != nil {
		return
	}
	defer rows.Close()

	result = []repository.TeacherEducationHistoryModel{}
	for rows.Next() {
		row, scanErr := scanEducation(rows)
		if scanErr != nil {
			return result, scanErr
		}
		result = append(result, row)
	}

	err = rows.Err()
	return
}

// Insert takes a transaction because the caller may be a teacher create, where
// the teacher row and its education rows are one atomic unit (criterion 7).
// The standalone add-education route passes its own.
//
// teacher_id is the teacher's internal bigint, not its uuid_key: this is the
// one place the surrogate is the right currency, because it is what the column
// holds.
func (d EducationDAO) Insert(ctx context.Context, tx *sql.Tx, m repository.TeacherEducationHistoryModel) (result repository.TeacherEducationHistoryModel, err error) {
	query := educationWrite(`INSERT INTO teacher_education_histories
		(teacher_id, institution_level_id, institution, study_start_date, study_end_date, score, focused_subject)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`)

	return scanEducation(tx.QueryRowContext(ctx, query,
		m.TeacherID, m.InstitutionLevelID, m.Institution,
		m.StudyStartDate, m.StudyEndDate, m.Score, m.FocusedSubject,
	))
}

// Update edits one row, scoped by the teacher as well as the row.
//
// A row belonging to another teacher matches nothing - the same answer as a row
// that does not exist, which is the point: the caller cannot use this to probe
// for rows it has no business seeing (criterion 28).
//
// It is addressed by uuid_key rather than by the internal id, so the bigint
// never has to leave the database to be handed back in. That also removes a
// lookup: nothing needs to resolve the key to an id before writing.
//
// RowsAffected is gone in favour of RETURNING. A zero-row result is the same
// signal it was, but a found row now arrives without a second query - and it is
// the row as Postgres wrote it, which matters for score, where NUMERIC(5,2)
// rewrites "87.5" to "87.50" and the response has to show what was stored.
func (d EducationDAO) Update(ctx context.Context, tx *sql.Tx, uuidKey string, teacherID int64, m repository.TeacherEducationHistoryModel) (result repository.TeacherEducationHistoryModel, err error) {
	query := educationWrite(`UPDATE teacher_education_histories SET
			institution_level_id = $1,
			institution          = $2,
			study_start_date     = $3,
			study_end_date       = $4,
			score                = $5,
			focused_subject      = $6,
			updated_at           = CURRENT_TIMESTAMP
		WHERE uuid_key = $7 AND teacher_id = $8`)

	return scanEducation(tx.QueryRowContext(ctx, query,
		m.InstitutionLevelID, m.Institution, m.StudyStartDate, m.StudyEndDate,
		m.Score, m.FocusedSubject, uuidKey, teacherID,
	))
}

// Delete is a real DELETE, not a soft delete: this table has no `deleted`
// column and is deliberately not registered with audit_helper, so a removed
// education row is gone. Criterion 27's observable behaviour is the row
// disappearing while its siblings and the teacher survive, which is what the
// teacher_id predicate guarantees - and it is checked by the same statement
// that deletes, so there is no window between deciding and doing.
//
// The deleted row comes back through RETURNING so the caller can report what
// was removed. A zero-row result means the key matched nothing *for this
// teacher*.
func (d EducationDAO) Delete(ctx context.Context, tx *sql.Tx, uuidKey string, teacherID int64) (result repository.TeacherEducationHistoryModel, err error) {
	query := educationWrite(`DELETE FROM teacher_education_histories
		WHERE uuid_key = $1 AND teacher_id = $2`)

	return scanEducation(tx.QueryRowContext(ctx, query, uuidKey, teacherID))
}
