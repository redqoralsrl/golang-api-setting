package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenConfig struct {
	SecretKey string
}

type JWTAdapter struct {
	config TokenConfig
}

func NewJWTAdapter(secretKey string) *JWTAdapter {
	return &JWTAdapter{
		config: TokenConfig{
			SecretKey: secretKey,
		},
	}
}

type Claims struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

func (a *JWTAdapter) GenerateToken(userID, role string, expired time.Duration) (string, error) {
	claims := &Claims{
		UserID: userID,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expired)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.config.SecretKey))
}

func (a *JWTAdapter) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(a.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
