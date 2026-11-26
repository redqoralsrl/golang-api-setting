package logger

import (
	"context"
	"errors"
	"strings"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
	With(fields ...Field) Logger
	Close() error
}

type Field struct {
	Key   string
	Value any
	// Canceled 클라이언트가 요청을 끊어 작업이 멈춘 오류다. 서버 장애가 아니므로 구현체는 Error 대신 Warn 으로 남긴다.
	Canceled bool
}

func NewField(key string, value any) Field {
	return Field{Key: key, Value: value}
}

func NewError(err error) Field {
	if err == nil {
		return Field{Key: "error", Value: nil}
	}
	// 실행 중인 쿼리가 취소되면 드라이버에 따라 context.Canceled 대신 PostgreSQL 서버 오류(57014)가 온다.
	canceled := errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "canceling statement due to user request")
	return Field{Key: "error", Value: err.Error(), Canceled: canceled}
}

// HasCanceled fields 중 클라이언트 취소로 생긴 오류가 있는지
func HasCanceled(fields []Field) bool {
	for _, f := range fields {
		if f.Canceled {
			return true
		}
	}
	return false
}
