package errorlog

import (
	"context"
	"time"
)

type ErrorLog struct {
	ID           int64     `json:"id"`
	OccurredAt   time.Time `json:"occurred_at"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	Path         string    `json:"path"`
	HTTPMethod   string    `json:"http_method"`
	RequestedURL string    `json:"requested_url"`
	ErrorCode    int       `json:"error_code"`
	ErrorMessage string    `json:"error_message"`
}

type Reader interface{}

type Writer interface {
	Create(ctx context.Context, errorLog *ErrorLog) error
}

type Repository interface {
	Reader
	Writer
}

type UseCase interface {
	Create(ctx context.Context, errorLog *ErrorLog) error
}
