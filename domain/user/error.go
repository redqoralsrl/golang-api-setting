package user

import (
	"errors"
	"fmt"
)

// postgres error code name
const (
	ErrDBSyntaxError               = "syntax_error"
	ErrDBUniqueViolation           = "unique_violation"
	ErrDBNotNullViolation          = "not_null_violation"
	ErrDBStringDataRightTruncation = "string_data_right_truncation"
	ErrDBInvalidTextRepresentation = "invalid_text_representation"
)

var (
	ErrUserSyntaxError               = errors.New("input type error")
	ErrUserUniqueViolation           = errors.New("user already exists")
	ErrUserNotNullViolation          = errors.New("null value error")
	ErrUserStringDataRightTruncation = errors.New("data truncation error")
	ErrUserInvalidTextRepresentation = errors.New("data value error")
	ErrUserNotFound                  = errors.New("user not found")
)

func NewDatabaseError(operation string, cause error) error {
	return fmt.Errorf("database operation %q failed: %w", operation, cause)
}

func NewAuthError(operation string, cause error) error {
	return fmt.Errorf("auth operation %q failed: %w", operation, cause)
}

func NewJwtError(email string, cause error) error {
	return fmt.Errorf("jwt error user %q failed: %w", email, cause)
}
