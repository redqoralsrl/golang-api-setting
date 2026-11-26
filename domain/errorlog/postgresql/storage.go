package postgresql

import (
	"context"

	"go-template/domain/errorlog"
	"go-template/internal/database/postgresql"
	"go-template/internal/database/postgresql/gen"
)

type ErrorLogStorage struct {
	db *postgresql.Database
}

var _ errorlog.Repository = (*ErrorLogStorage)(nil)

func NewErrorLog(db *postgresql.Database) *ErrorLogStorage {
	return &ErrorLogStorage{db: db}
}

func (s *ErrorLogStorage) Create(ctx context.Context, e *errorlog.ErrorLog) error {
	err := s.db.Queries(ctx).CreateErrorLog(ctx, gen.CreateErrorLogParams{
		OccurredAt:   e.OccurredAt,
		IpAddress:    e.IPAddress,
		UserAgent:    e.UserAgent,
		Path:         e.Path,
		HttpMethod:   e.HTTPMethod,
		RequestedUrl: e.RequestedURL,
		ErrorCode:    int32(e.ErrorCode),
		ErrorMessage: e.ErrorMessage,
	})
	return postgresql.MapError(err)
}
