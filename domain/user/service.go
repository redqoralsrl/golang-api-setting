package user

import (
	"context"
	"database/sql"
	"go-template/internal/database/postgresql"
	"go-template/internal/http/chi/middleware"
	"go-template/internal/logger"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo               Repository
	jwtAdapter         JwtAdapter
	transactionManager postgresql.DBTransactionManager
	logger             logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, j JwtAdapter, tm postgresql.DBTransactionManager, logger logger.Logger) *Service {
	service := &Service{
		repo:               r,
		jwtAdapter:         j,
		transactionManager: tm,
		logger:             logger,
	}

	return service
}

func (s *Service) Create(ctx context.Context, email, password string) (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, NewAuthError("hash admin password", err)
	}

	var user *User
	if err := s.transactionManager.WithTransaction(ctx, sql.LevelReadCommitted, false, func(txCtx context.Context) error {
		user, err = s.repo.Create(ctx, email, string(passwordHash))
		if err != nil {
			s.logger.Error("failed to create user", logger.Field{Key: "error", Value: err.Error()})
			return NewDatabaseError("create user", err)
		}

		accessToken, err := s.jwtAdapter.GenerateToken(strconv.Itoa(user.ID), string(middleware.UserRole), middleware.UserExpired)
		if err != nil {
			return NewJwtError(user.Email, err)
		}

		user.AccessToken = accessToken

		err = s.repo.CreateLoginLog(ctx, user.ID)
		if err != nil {
			s.logger.Error("failed to create user login log", logger.Field{Key: "error", Value: err.Error()})
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return user, nil
}
