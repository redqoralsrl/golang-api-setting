package zerolog

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"go-template/internal/logger"

	"github.com/rs/zerolog"
)

func config(stage string) zerolog.Logger {
	var logger zerolog.Logger

	// 시간 포맷터 설정
	zerolog.TimeFieldFormat = time.RFC3339

	if stage == "dev" {
		// 개발 환경 설정
		output := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
			NoColor:    false, // 컬러 출력 활성화
		}

		logger = zerolog.New(output).
			Level(zerolog.DebugLevel). // 디버그 레벨 설정
			With().
			Timestamp(). // 타임스탬프 추가
			Logger()

	} else {
		// 운영 환경 설정
		logger = zerolog.New(os.Stdout).
			Level(zerolog.InfoLevel). // 인포 레벨 설정
			With().
			Timestamp(). // 타임스탬프 추가
			Logger()
	}
	logger.Warn().Msg("Logger initialized")
	logger.Error().Fields(map[string]interface{}{"key": "value"}).Msg("Logger initialized")
	logger.Debug().Fields(map[string]interface{}{"key": "value"}).Msg("Logger initialized")

	return logger
}

func NewLogger(stage string) logger.Logger {
	logger := config(stage)
	return &ZeroLogger{log: logger}
}

// ZeroLogger Zerolog 구현체
type ZeroLogger struct {
	log zerolog.Logger
}

func (z *ZeroLogger) Debug(msg string, fields ...logger.Field) {
	_, file, line, ok := runtime.Caller(1)
	event := z.log.Debug()
	if ok {
		parts := strings.Split(file, "/")
		shortPath := filepath.Base(file)
		if len(parts) >= 2 {
			shortPath = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		event = event.Str("caller", fmt.Sprintf("%s:%d", shortPath, line))
	}
	z.appendFields(event, fields...)
	event.Msg(msg)
}

func (z *ZeroLogger) Info(msg string, fields ...logger.Field) {
	_, file, line, ok := runtime.Caller(1)
	event := z.log.Info()
	if ok {
		parts := strings.Split(file, "/")
		shortPath := filepath.Base(file)
		if len(parts) >= 2 {
			shortPath = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		event = event.Str("caller", fmt.Sprintf("%s:%d", shortPath, line))
	}
	z.appendFields(event, fields...)
	event.Msg(msg)
}

func (z *ZeroLogger) Warn(msg string, fields ...logger.Field) {
	_, file, line, ok := runtime.Caller(1)
	event := z.log.Warn()
	if ok {
		parts := strings.Split(file, "/")
		shortPath := filepath.Base(file)
		if len(parts) >= 2 {
			shortPath = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		event = event.Str("caller", fmt.Sprintf("%s:%d", shortPath, line))
	}
	z.appendFields(event, fields...)
	event.Msg(msg)
}

func (z *ZeroLogger) Error(msg string, fields ...logger.Field) {
	_, file, line, ok := runtime.Caller(1)
	event := z.log.Error()
	if ok {
		parts := strings.Split(file, "/")
		shortPath := filepath.Base(file)
		if len(parts) >= 2 {
			shortPath = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		event = event.Str("caller", fmt.Sprintf("%s:%d", shortPath, line))
	}
	z.appendFields(event, fields...)
	event.Msg(msg)
}

func (z *ZeroLogger) Fatal(msg string, fields ...logger.Field) {
	_, file, line, ok := runtime.Caller(1)
	event := z.log.Fatal()
	if ok {
		parts := strings.Split(file, "/")
		shortPath := filepath.Base(file)
		if len(parts) >= 2 {
			shortPath = parts[len(parts)-2] + "/" + parts[len(parts)-1]
		}
		event = event.Str("caller", fmt.Sprintf("%s:%d", shortPath, line))
	}
	z.appendFields(event, fields...)
	event.Msg(msg)
}

func (z *ZeroLogger) With(fields ...logger.Field) logger.Logger {
	contextLogger := z.log.With()
	for _, f := range fields {
		contextLogger = contextLogger.Interface(f.Key, f.Value)
	}
	return &ZeroLogger{log: contextLogger.Logger()}
}

func (z *ZeroLogger) appendFields(event *zerolog.Event, fields ...logger.Field) {
	for _, f := range fields {
		event.Interface(f.Key, f.Value)
	}
}

func (z *ZeroLogger) Close() error {
	return nil
}
