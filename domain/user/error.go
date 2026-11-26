package user

import (
	"errors"
	"fmt"
)

// 도메인 에러. repository 가 DB 에러를 이것으로 바꿔 돌려주고, 핸들러는 이것으로 status 를 정한다.
var (
	ErrUserAlreadyExists = errors.New("user already exists")
	ErrUserNotFound      = errors.New("user not found")
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}

func NewAuthError(operation string, cause error) error {
	return fmt.Errorf("auth operation %q failed: %w", operation, cause)
}
