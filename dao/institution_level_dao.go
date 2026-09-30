package dao

import (
	"context"
	"database/sql"

	"sample-project/repository"
)

// InstitutionLevelDAO reads the seeded institution_levels lookup. Read-only by
// design - architecture A4 makes this a fixed vocabulary, so there is no
// insert/update/delete here to accidentally expose.
type InstitutionLevelDAO struct {
	Db *sql.DB
}

func NewInstitutionLevelDAO(db *sql.DB) InstitutionLevelDAO {
	return InstitutionLevelDAO{Db: db}
}

const institutionLevelColumns = `id, uuid_key, name, created_at, updated_at`

// GetByName resolves the level code a request carries into the id the
// education table's foreign key needs. An unknown code comes back as a
// found-but-empty model, the same convention as the teacher reads, so the
// caller decides whether an unknown level is a 400 or a 404.
func (d InstitutionLevelDAO) GetByName(ctx context.Context, name string) (result repository.InstitutionLevelModel, err error) {
	ctx, cancel := withQueryTimeout(ctx)
	defer cancel()

	query := `SELECT ` + institutionLevelColumns + ` FROM institution_levels WHERE name = $1`

	err = d.Db.QueryRowContext(ctx, query, name).Scan(
		&result.ID, &result.UUIDKey, &result.Name, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		err = nil
	}

	return
}
