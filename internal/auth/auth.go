// Package auth 는 HTTP·도메인 계층이 함께 쓰는 인증 값(역할, 토큰 만료, 요청 주체)을 둔다.
// 도메인이 HTTP 미들웨어를 import 하지 않도록 여기로 분리했다.
package auth

import (
	"context"
	"time"
)

type Role string

const (
	RoleUser Role = "user"
)

// UserTokenTTL 앱 사용자 access token 유효 시간
const UserTokenTTL = 1 * time.Hour

// Identity 인증된 요청의 주체
type Identity struct {
	UserID string
	Role   Role
}

type identityKey struct{}

func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey{}, id)
}

// FromContext 인증 미들웨어를 통과한 요청이면 ok 가 true 다.
func FromContext(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey{}).(Identity)
	return id, ok
}
