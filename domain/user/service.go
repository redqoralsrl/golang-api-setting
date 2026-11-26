package user

import (
	"context"
	"database/sql"
	"strconv"
	"strings"

	"go-template/internal/auth"
	"go-template/internal/database/postgresql"
	"go-template/internal/logger"

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
	return &Service{
		repo:               r,
		jwtAdapter:         j,
		transactionManager: tm,
		logger:             logger,
	}
}

func (s *Service) Create(ctx context.Context, email, password string) (*AuthResult, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, NewAuthError("hash user password", err)
	}

	var user *User
	// 가입과 첫 로그인 기록은 함께 성공하거나 함께 실패한다.
	// (Postgres 는 트랜잭션 안 쿼리가 하나라도 실패하면 이후 Commit 도 실패하므로 에러를 무시할 수 없다)
	err = s.transactionManager.WithTransaction(ctx, sql.LevelReadCommitted, false, func(txCtx context.Context) error {
		user, err = s.repo.Create(txCtx, strings.ToLower(strings.TrimSpace(email)), string(passwordHash))
		if err != nil {
			return err
		}
		if err := s.repo.CreateLoginLog(txCtx, user.ID); err != nil {
			return NewDatabaseError("create user login log", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	accessToken, err := s.jwtAdapter.GenerateToken(strconv.Itoa(user.ID), string(auth.RoleUser), auth.UserTokenTTL)
	if err != nil {
		return nil, NewJwtError(user.Email, err)
	}

	return &AuthResult{User: user, AccessToken: accessToken}, nil
}
