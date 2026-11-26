package jwt

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const minSecretKeyLen = 32

type JWTAdapter struct {
	secretKey []byte
	parser    *jwt.Parser
}

func NewJWTAdapter(secretKey string) (*JWTAdapter, error) {
	if len(secretKey) < minSecretKeyLen {
		return nil, fmt.Errorf("jwt secret key must be at least %d characters", minSecretKeyLen)
	}

	return &JWTAdapter{
		secretKey: []byte(secretKey),
		// HS256 외 알고리즘(none 포함)과 exp 없는 토큰은 거부한다
		parser: jwt.NewParser(jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired()),
	}, nil
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (a *JWTAdapter) GenerateToken(userID, role string, expired time.Duration) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expired)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(a.secretKey)
}

func (a *JWTAdapter) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := a.parser.ParseWithClaims(tokenString, claims, func(*jwt.Token) (interface{}, error) {
		return a.secretKey, nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
