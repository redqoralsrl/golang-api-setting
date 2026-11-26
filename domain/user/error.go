package user

import (
	"errors"
	"fmt"
)

// 도메인 에러. repository 가 DB 에러를 이것으로 바꿔 돌려주고, 핸들러는 이것으로 status 를 정한다.
var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
	// ErrInvalidCredentials 이메일이 없는 경우와 비밀번호가 틀린 경우를 구분하지 않는다 (계정 존재 여부 노출 방지)
	ErrInvalidCredentials = errors.New("invalid email or password")
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}

func NewAuthError(operation string, cause error) error {
	return fmt.Errorf("auth operation %q failed: %w", operation, cause)
}
