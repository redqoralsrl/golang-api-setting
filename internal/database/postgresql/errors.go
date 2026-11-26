package postgresql

import (
	"errors"
	"fmt"

	"go-template/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL 에러 코드 → 공통 에러. https://www.postgresql.org/docs/current/errcodes-appendix.html
var pgCodes = map[string]error{
	"23505": database.ErrUniqueViolation,
	"23503": database.ErrForeignKeyViolation,
	"23502": database.ErrNotNullViolation,
	"23514": database.ErrCheckViolation,
	"22001": database.ErrValueTooLong,
	"22P02": database.ErrInvalidValue,
}

// MapError pgx 에러를 database.Err* 로 감싼다. 원래 에러도 errors.Is/As 로 꺼낼 수 있다.
// 매핑할 게 없으면 그대로 돌려준다.
func MapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: %w", database.ErrNotFound, err)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if mapped, ok := pgCodes[pgErr.Code]; ok {
			return fmt.Errorf("%w: %w", mapped, err)
		}
	}
	return err
}
