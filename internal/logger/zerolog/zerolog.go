package zerolog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"go-template/internal/logger"

	"github.com/rs/zerolog"
)

// ZeroLogger Zerolog 구현체. dev 는 컬러 콘솔(Debug), 그 외는 JSON(Info)으로 출력한다.
type ZeroLogger struct {
	log zerolog.Logger
}

var _ logger.Logger = (*ZeroLogger)(nil)

func NewLogger(stage string) logger.Logger {
	zerolog.TimeFieldFormat = time.RFC3339

	if stage == "dev" {
		out := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
		return &ZeroLogger{log: zerolog.New(out).Level(zerolog.DebugLevel).With().Timestamp().Logger()}
	}
	return &ZeroLogger{log: zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()}
}

func (z *ZeroLogger) Debug(msg string, fields ...logger.Field) { z.write(z.log.Debug(), msg, fields) }
func (z *ZeroLogger) Info(msg string, fields ...logger.Field)  { z.write(z.log.Info(), msg, fields) }
func (z *ZeroLogger) Warn(msg string, fields ...logger.Field)  { z.write(z.log.Warn(), msg, fields) }
func (z *ZeroLogger) Fatal(msg string, fields ...logger.Field) { z.write(z.log.Fatal(), msg, fields) }

func (z *ZeroLogger) Error(msg string, fields ...logger.Field) {
	if logger.HasCanceled(fields) {
		z.write(z.log.Warn(), msg, fields)
		return
	}
	z.write(z.log.Error(), msg, fields)
}

func (z *ZeroLogger) With(fields ...logger.Field) logger.Logger {
	ctx := z.log.With()
	for _, f := range fields {
		ctx = ctx.Interface(f.Key, f.Value)
	}
	return &ZeroLogger{log: ctx.Logger()}
}

func (z *ZeroLogger) Close() error { return nil }

// write 호출 위치(파일:줄)와 필드를 붙여 출력한다. Caller(2): write ← Info 등 ← 실제 호출자
func (z *ZeroLogger) write(event *zerolog.Event, msg string, fields []logger.Field) {
	if _, file, line, ok := runtime.Caller(2); ok {
		event = event.Str("caller", fmt.Sprintf("%s/%s:%d", filepath.Base(filepath.Dir(file)), filepath.Base(file), line))
	}
	for _, f := range fields {
		event = event.Interface(f.Key, f.Value)
	}
	event.Msg(msg)
}
