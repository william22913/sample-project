package teacher

import (
	"database/sql"

	"github.com/google/uuid"
	"github.com/nexsoft-git/nexcommon/context"
	"github.com/nexsoft-git/nexcommon/model"
	"github.com/nexsoft-git/nexcommon/services"
	"github.com/nexsoft-git/nexcommon/services/audit_helper"

	"sample-project/dao"
	in "sample-project/dto/in"
	errors "sample-project/error"
)

// Service owns the teacher routes.
type Service struct {
	teacherDAO   dao.TeacherDAO
	educationDAO dao.EducationDAO
	levelDAO     dao.InstitutionLevelDAO
	auditHelper  audit_helper.AuditHelper

	// operator and validSearchBy are the list endpoint's filter vocabulary,
	// built once here. GetListValidSearch is derived from operator rather than
	// declared alongside it - see get_list_teacher.go.
	operator      services.DefaultOperators
	validSearchBy []string
}

func NewService(
	teacherDAO dao.TeacherDAO,
	educationDAO dao.EducationDAO,
	levelDAO dao.InstitutionLevelDAO,
	auditHelper audit_helper.AuditHelper,
) *Service {
	operator := listOperators()

	return &Service{
		teacherDAO:    teacherDAO,
		educationDAO:  educationDAO,
		levelDAO:      levelDAO,
		auditHelper:   auditHelper,
		operator:      operator,
		validSearchBy: operator.GetListValidSearch(),
	}
}

func (s *Service) GetListScope() []string { return []string{} }

func (s *Service) GetDTO() interface{} { return &in.TeacherIn{} }

func (s *Service) GetMultipartDTO() interface{} { return nil }

// EducationService owns the education sub-resource routes.
//
// A second value rather than more methods on Service, and the reason is a
// library constraint, not a preference: nexcommon's controller reads the
// request body's type from one service.GetDTO() per registered service, with no
// per-endpoint override (controller/http/util.go, wrapper.go). A teacher update
// and an education edit genuinely have different bodies, so they have to be
// different service values. The routes nest under /teachers/{id}/educations
// because they are addressed through a teacher; the DTOs cannot nest with them.
//
// It shares the dao values and the level lookup with Service. It deliberately
// does NOT hold the audit helper: architecture A1 registers only `teachers` for
// auditing, so an education write produces no audit entry and needs no
// transaction worth borrowing audit_helper's for. Each of its three routes is a
// single row-scoped statement, atomic on its own.
type EducationService struct {
	teacherDAO   dao.TeacherDAO
	educationDAO dao.EducationDAO
	levelDAO     dao.InstitutionLevelDAO
	db           *sql.DB
}

func NewEducationService(
	teacherDAO dao.TeacherDAO,
	educationDAO dao.EducationDAO,
	levelDAO dao.InstitutionLevelDAO,
	db *sql.DB,
) *EducationService {
	return &EducationService{
		teacherDAO:   teacherDAO,
		educationDAO: educationDAO,
		levelDAO:     levelDAO,
		db:           db,
	}
}

func (s *EducationService) GetListScope() []string { return []string{} }

func (s *EducationService) GetDTO() interface{} { return &in.EducationIn{} }

func (s *EducationService) GetMultipartDTO() interface{} { return nil }

// inTransaction wraps each education write. The three routes are single
// statements, so nothing here depends on it for correctness - it is used
// because it is the one place a transaction is opened in this package that is
// not audit_helper's, and having it in one function keeps that exception
// visible. It also means a future write that genuinely spans statements has
// somewhere to go without changing any signature.
//
// Rollback is deferred rather than conditional: after a successful Commit it
// returns ErrTxDone, which is discarded, and that is cheaper to read than a
// flag tracking whether the commit happened.
func (s *EducationService) inTransaction(
	ctx *context.ContextModel,
	fn func(tx *sql.Tx) error,
) error {
	tx, err := s.db.BeginTx(ctx.ToContext(), nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if err = fn(tx); err != nil {
		return err
	}

	return tx.Commit()
}

// uuidPath reads a uuid-valued path parameter and rejects a value the database
// cannot accept.
//
// Without this the raw string reaches Postgres, which answers `invalid input
// syntax for type uuid` (SQLSTATE 22P02); the DAO passes that up as an ordinary
// error and nexcommon's formator turns anything untyped into a 500. A client
// typo - GET /teachers/not-a-uuid - is not a server fault, and the framework
// gives path parameters no validation of its own: PathParams(...) records them
// for the swagger document and does nothing else with them.
//
// The check is uuid.Parse and not a length-and-dashes pattern because that is
// the same question Postgres is about to ask. A hand-written pattern that
// disagrees with it reintroduces the 500 on whichever accepted forms it misses,
// and Postgres accepts more than the canonical hyphenated shape.
//
// pathKey is a constanta.Path* key, fieldKey the constanta.Field* key the
// rejection names. The returned error is a business error and is deliberately
// not logged here - nexcommon's controller logs it once at the top of the
// request, and nothing else does.
func uuidPath(
	ctx *context.ContextModel,
	param model.URLParam,
	pathKey string,
	fieldKey string,
) (string, error) {
	value := param.Path[pathKey]
	if _, err := uuid.Parse(value); err != nil {
		return "", errors.ErrFormatField(fieldKey).ContextModel(ctx)
	}

	return value, nil
}

// actorID is the created_by/updated_by value for the current request.
//
// It returns an invalid NullInt64 rather than 0 when there is no authenticated
// user, so the column is written as NULL instead of as a user id of zero
// (architecture A2: there is no authentication in this phase, so this is every
// request). Shared with Service via the package.
func actorID(ctx *context.ContextModel) sql.NullInt64 {
	if ctx.Limitation.UserID <= 0 {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: ctx.Limitation.UserID, Valid: true}
}
