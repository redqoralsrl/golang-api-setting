package user

import (
	"context"
	"time"
)

type User struct {
	ID           int       `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // 응답에 절대 싣지 않는다
}

// AuthResult 가입·로그인 결과. AccessToken 은 dev 에서만 바디로 내려간다(그 외는 쿠키).
type AuthResult struct {
	User        *User  `json:"user"`
	AccessToken string `json:"access_token,omitempty"`
}

type Reader interface {
}

type Writer interface {
	Create(ctx context.Context, email, passwordHash string) (*User, error)
	CreateLoginLog(ctx context.Context, userID int) error
}

type Repository interface {
	Reader
	Writer
}

type UseCase interface {
	Create(ctx context.Context, email, password string) (*AuthResult, error)
}

type JwtAdapter interface {
	GenerateToken(userID, role string, expired time.Duration) (string, error)
}
