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
	PasswordHash string    `json:"password_hash,omitempty"`
	AccessToken  string    `json:"access_token,omitempty"`
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
	Create(ctx context.Context, email, password string) (*User, error)
}

type JwtAdapter interface {
	GenerateToken(userID, role string, expired time.Duration) (string, error)
}
