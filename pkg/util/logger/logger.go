package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger generates new logger instance
func NewLogger(level string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	var lvl zapcore.Level
	if err := lvl.Set(level); err != nil {
		return nil, err
	}

	cfg.Level = zap.NewAtomicLevelAt(lvl)
	cfg.EncoderConfig.TimeKey = "ts"
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	return cfg.Build()
}
