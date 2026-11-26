package jwt

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testKey = "0123456789abcdef0123456789abcdef"

func TestNewJWTAdapterRejectsShortKey(t *testing.T) {
	if _, err := NewJWTAdapter("short"); err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestTokenRoundTrip(t *testing.T) {
	a, _ := NewJWTAdapter(testKey)
	token, err := a.GenerateToken("42", "user", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := a.ValidateToken(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != "42" || claims.Role != "user" {
		t.Errorf("unexpected claims %+v", claims)
	}
}

func TestValidateTokenRejects(t *testing.T) {
	a, _ := NewJWTAdapter(testKey)

	expired, _ := a.GenerateToken("1", "user", -time.Minute)
	other, _ := NewJWTAdapter(strings.Repeat("x", 32))
	foreign, _ := other.GenerateToken("1", "user", time.Minute)
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{UserID: "1"}).SignedString([]byte(testKey))
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute))},
	}).SignedString([]byte(testKey))

	for name, token := range map[string]string{
		"expired": expired, "other key": foreign, "no exp": noExp, "other alg": hs512, "garbage": "a.b.c",
	} {
		if _, err := a.ValidateToken(token); err == nil {
			t.Errorf("%s: expected error", name)
		}
	}
}
