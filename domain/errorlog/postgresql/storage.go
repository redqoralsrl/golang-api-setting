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
	return &ErrorLogStorage{
		db: db,
	}
}

func (s *ErrorLogStorage) Create(ctx context.Context, errorLog *errorlog.ErrorLog) (*errorlog.ErrorLog, error) {
	queryRower := s.db.GetQueryRowerFromContext(ctx)

	row, err := queryRower.Queries.CreateErrorLog(ctx, gen.CreateErrorLogParams{
		Timestamp:    errorLog.Timestamp,
		IpAddress:    errorLog.IPAddress,
		UserAgent:    errorLog.UserAgent,
		Path:         errorLog.Path,
		HttpMethod:   errorLog.HTTPMethod,
		RequestedUrl: errorLog.RequestedURL,
		ErrorCode:    int32(errorLog.ErrorCode),
		ErrorMessage: errorLog.ErrorMessage,
	})
	if err != nil {
		return nil, err
	}

	return &errorlog.ErrorLog{
		ID:           int(row.ID),
		Timestamp:    row.Timestamp,
		IPAddress:    row.IpAddress,
		UserAgent:    row.UserAgent,
		Path:         row.Path,
		HTTPMethod:   row.HttpMethod,
		RequestedURL: row.RequestedUrl,
		ErrorCode:    int(row.ErrorCode),
		ErrorMessage: row.ErrorMessage,
	}, nil
}
