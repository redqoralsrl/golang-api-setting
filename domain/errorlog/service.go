package errorlog

import (
	"context"

	"go-template/internal/logger"
)

type Service struct {
	repo   Repository
	logger logger.Logger
}

var _ UseCase = (*Service)(nil)

func NewService(r Repository, logger logger.Logger) *Service {
	return &Service{
		repo:   r,
		logger: logger,
	}
}

func (s *Service) Create(ctx context.Context, errorLog *ErrorLog) (*ErrorLog, error) {
	log, err := s.repo.Create(ctx, errorLog)
	if err != nil {
		s.logger.Error("failed to create error log", logger.NewError(err))
		return nil, NewDatabaseError("create error log", err)
	}

	return log, nil
}
