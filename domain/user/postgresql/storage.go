package postgresql

import (
	"context"
	"errors"
	"go-template/domain/user"
	"go-template/internal/database/postgresql"
	"go-template/internal/database/postgresql/gen"

	"github.com/lib/pq"
)

type UserStorage struct {
	db *postgresql.Database
}

var _ user.Repository = (*UserStorage)(nil)

func NewUser(db *postgresql.Database) *UserStorage {
	return &UserStorage{
		db: db,
	}
}

func (s *UserStorage) Create(ctx context.Context, email, passwordHash string) (*user.User, error) {
	queryRower := s.db.GetQueryRowerFromContext(ctx)

	row, err := queryRower.Queries.CreateUser(ctx, gen.CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) {
			switch pqErr.Code.Name() {
			case user.ErrDBSyntaxError:
				return nil, user.ErrUserSyntaxError
			case user.ErrDBUniqueViolation:
				return nil, user.ErrUserUniqueViolation
			case user.ErrDBNotNullViolation:
				return nil, user.ErrUserNotNullViolation
			case user.ErrDBStringDataRightTruncation:
				return nil, user.ErrUserStringDataRightTruncation
			case user.ErrDBInvalidTextRepresentation:
				return nil, user.ErrUserInvalidTextRepresentation
			}
		}

		return nil, err
	}

	return &user.User{
		ID:        int(row.ID),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		Email:     row.Email,
	}, nil
}

func (s *UserStorage) CreateLoginLog(ctx context.Context, userID int) error {
	queryRower := s.db.GetQueryRowerFromContext(ctx)

	return queryRower.Queries.CreateUserLoginLog(ctx, gen.CreateUserLoginLogParams{
		UserID: int32(userID),
	})
}
