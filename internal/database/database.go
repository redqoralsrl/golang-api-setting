// Package database 는 DB 드라이버와 무관한 트랜잭션·에러·락 인터페이스를 둔다.
// 도메인은 이 패키지만 보고, 구현은 internal/database/postgresql 에 있다.
package database

import (
	"context"
	"errors"
)

type IsolationLevel int

const (
	ReadCommitted IsolationLevel = iota // 기본값
	RepeatableRead
	Serializable
)

type TxOptions struct {
	Isolation IsolationLevel
	ReadOnly  bool
}

type TxOption func(*TxOptions)

// ReadOnly 읽기 전용 트랜잭션
func ReadOnly() TxOption { return func(o *TxOptions) { o.ReadOnly = true } }

// WithIsolation 격리 수준 지정. 기본은 ReadCommitted
func WithIsolation(level IsolationLevel) TxOption {
	return func(o *TxOptions) { o.Isolation = level }
}

// TxManager fn 안에서 받은 ctx 를 repository 에 넘기면 같은 트랜잭션으로 실행된다.
// fn 이 에러를 돌려주거나 panic 하면 롤백한다. 이미 트랜잭션 안이면 바깥 트랜잭션을 그대로 쓴다.
type TxManager interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error, opts ...TxOption) error
}

// 드라이버 에러를 바꾼 공통 에러. repository 는 이것을 도메인 에러로 바꿔 돌려준다.
//
//	if errors.Is(err, database.ErrUniqueViolation) { return user.ErrUserAlreadyExists }
var (
	ErrNotFound            = errors.New("record not found")
	ErrUniqueViolation     = errors.New("unique violation")
	ErrForeignKeyViolation = errors.New("foreign key violation")
	ErrNotNullViolation    = errors.New("not null violation")
	ErrCheckViolation      = errors.New("check violation")
	ErrValueTooLong        = errors.New("value too long")
	ErrInvalidValue        = errors.New("invalid value")
)
