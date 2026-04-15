package zap

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func config(stage string) (*zap.Logger, error) {
	var config zap.Config

	if stage == "dev" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 컬러로 레벨 표시
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder        // 시간 포맷 ISO8601
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder      // 파일명만 표시
		config.DisableStacktrace = false                                    // 스택트레이스 활성화
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)                 // 로그 레벨 설정
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 시간 포맷 ISO8601
		config.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder // 파일명만 표시
		config.DisableStacktrace = true                                // 스택트레이스 비활성화
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)             // 로그 레벨 설정
	}

	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func NewLogger(stage string) (*zap.Logger, error) {
	logger, err := config(stage)
	if err != nil {
		return nil, err
	}
	return logger, nil
}
