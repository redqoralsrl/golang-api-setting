package user

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"go-template/internal/auth"
	"go-template/internal/database"
	"go-template/internal/logger"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      Repository
	tokens    TokenIssuer
	txManager database.TxManager
	logger    logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, tokens TokenIssuer, tm database.TxManager, l logger.Logger) *Service {
	return &Service{repo: r, tokens: tokens, txManager: tm, logger: l}
}

func (s *Service) Create(ctx context.Context, email, password string) (*AuthResult, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, NewAuthError("hash user password", err)
	}

	var u *User
	// 가입과 첫 로그인 기록은 함께 성공하거나 함께 실패한다.
	err = s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if u, err = s.repo.Create(ctx, normalizeEmail(email), string(passwordHash)); err != nil {
			return err
		}
		return s.repo.CreateLoginLog(ctx, u.ID)
	})
	if err != nil {
		return nil, err
	}

	return s.issue(u)
}

// dummyHash 없는 이메일로 로그인해도 bcrypt 비교를 한 번 해서 응답 시간으로 계정 존재 여부가 드러나지 않게 한다.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

func (s *Service) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	u, err := s.repo.GetByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}

	// 로그인 기록 실패로 로그인을 막지는 않는다
	if err := s.repo.CreateLoginLog(ctx, u.ID); err != nil {
		s.logger.Warn("failed to create user login log", logger.NewField("user_id", u.ID), logger.NewError(err))
	}
	return s.issue(u)
}

func (s *Service) Get(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) issue(u *User) (*AuthResult, error) {
	token, err := s.tokens.GenerateToken(strconv.FormatInt(u.ID, 10), string(auth.RoleUser), auth.UserTokenTTL)
	if err != nil {
		return nil, NewAuthError("generate token", err)
	}
	return &AuthResult{User: u, AccessToken: token}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
