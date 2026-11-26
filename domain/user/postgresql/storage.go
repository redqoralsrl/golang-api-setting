package postgresql

import (
	"context"
	"errors"

	"go-template/domain/user"
	"go-template/internal/database"
	"go-template/internal/database/postgresql"
	"go-template/internal/database/postgresql/gen"
)

type UserStorage struct {
	db *postgresql.Database
}

var _ user.Repository = (*UserStorage)(nil)

func NewUser(db *postgresql.Database) *UserStorage {
	return &UserStorage{db: db}
}

func (s *UserStorage) Create(ctx context.Context, email, passwordHash string) (*user.User, error) {
	row, err := s.db.Queries(ctx).CreateUser(ctx, gen.CreateUserParams{Email: email, PasswordHash: passwordHash})
	if err != nil {
		return nil, mapError(err)
	}
	return toUser(row), nil
}

func (s *UserStorage) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	row, err := s.db.Queries(ctx).GetUserByEmail(ctx, gen.GetUserByEmailParams{Email: email})
	if err != nil {
		return nil, mapError(err)
	}
	return toUser(row), nil
}

func (s *UserStorage) GetByID(ctx context.Context, id int64) (*user.User, error) {
	row, err := s.db.Queries(ctx).GetUserByID(ctx, gen.GetUserByIDParams{ID: id})
	if err != nil {
		return nil, mapError(err)
	}
	return toUser(row), nil
}

func (s *UserStorage) CreateLoginLog(ctx context.Context, userID int64) error {
	return mapError(s.db.Queries(ctx).CreateUserLoginLog(ctx, gen.CreateUserLoginLogParams{UserID: userID}))
}

// mapError DB 에러를 도메인 에러로 바꾼다. 매핑하지 않는 에러는 원인을 보존해 돌려준다.
func mapError(err error) error {
	err = postgresql.MapError(err)
	switch {
	case err == nil:
		return nil
	case errors.Is(err, database.ErrNotFound):
		return user.ErrUserNotFound
	case errors.Is(err, database.ErrUniqueViolation):
		return user.ErrUserAlreadyExists
	default:
		return user.NewDatabaseError("user", err)
	}
}

func toUser(row gen.V1User) *user.User {
	return &user.User{
		ID:           row.ID,
		CreatedAt:    row.CreatedAt,
		UpdatedAt:    row.UpdatedAt,
		Email:        row.Email,
		PasswordHash: row.PasswordHash,
	}
}
