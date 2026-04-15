package errorlog

import (
	"context"
	"time"
)

type ErrorLog struct {
	ID           int       `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
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
	Create(ctx context.Context, errorLog *ErrorLog) (*ErrorLog, error)
}

type Repository interface {
	Reader
	Writer
}

type UseCase interface {
	Create(ctx context.Context, errorLog *ErrorLog) (*ErrorLog, error)
}
