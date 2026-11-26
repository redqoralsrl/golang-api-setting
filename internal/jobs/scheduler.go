// Package jobs API 서버 안에서 주기 작업(cron)을 돌린다.
package jobs

import (
	"context"
	"fmt"
	"time"

	"go-template/internal/database"
	"go-template/internal/logger"

	"github.com/robfig/cron/v3"
)

type Job struct {
	Name    string        // advisory lock 키와 로그에 쓴다. 작업마다 달라야 한다
	Spec    string        // cron 표현식 (UTC). 예: "5 * * * *" 매시 5분, "@every 10m"
	Timeout time.Duration // 한 번 실행의 최대 시간
	Run     func(ctx context.Context) error
}

// Scheduler 서버가 여러 대면 cron 이 모두 깨어나지만 작업별 advisory lock 을 잡은 한 대만 실행한다.
type Scheduler struct {
	cron   *cron.Cron
	locker database.SessionLocker
	logger logger.Logger
	ctx    context.Context
	cancel context.CancelFunc
}

func NewScheduler(locker database.SessionLocker, l logger.Logger) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		cron:   cron.New(cron.WithLocation(time.UTC)),
		locker: locker,
		logger: l.With(logger.NewField("component", "jobs")),
		ctx:    ctx,
		cancel: cancel,
	}
}

// Add 작업 등록. Start 전에 부른다.
func (s *Scheduler) Add(job Job) error {
	_, err := s.cron.AddFunc(job.Spec, func() { s.run(job) })
	if err != nil {
		return fmt.Errorf("register job %q: %w", job.Name, err)
	}
	return nil
}

func (s *Scheduler) run(job Job) {
	defer func() {
		if p := recover(); p != nil {
			s.logger.Error("job panicked", logger.NewField("job", job.Name), logger.NewField("panic", fmt.Sprint(p)))
		}
	}()

	release, acquired, err := s.locker.TryLock(s.ctx, database.NSJob, job.Name)
	if err != nil {
		s.logger.Error("failed to acquire job lock", logger.NewField("job", job.Name), logger.NewError(err))
		return
	}
	if !acquired {
		s.logger.Debug("job is running on another instance", logger.NewField("job", job.Name))
		return
	}
	defer release()

	ctx, cancel := context.WithTimeout(s.ctx, job.Timeout)
	defer cancel()

	startedAt := time.Now()
	if err := job.Run(ctx); err != nil {
		s.logger.Error("job failed", logger.NewField("job", job.Name), logger.NewField("elapsed", time.Since(startedAt).String()), logger.NewError(err))
		return
	}
	s.logger.Info("job finished", logger.NewField("job", job.Name), logger.NewField("elapsed", time.Since(startedAt).String()))
}

func (s *Scheduler) Start() { s.cron.Start() }

// Stop 실행 중인 작업에 취소를 알리고 끝날 때까지 기다린다.
func (s *Scheduler) Stop() {
	s.cancel()
	<-s.cron.Stop().Done()
}
