package log

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type key struct{}

var loggingKey key

func NewLoggingContext(ctx context.Context, logger logr.Logger) context.Context {
	return context.WithValue(ctx, loggingKey, logger)
}
func AppLogger(ctx context.Context) logr.Logger {
	logger, ok := ctx.Value(loggingKey).(logr.Logger)
	if !ok {
		panic("logging not set up in context")
	}
	return logger
}

func NewLogger(name, version string, level int, format string) (logr.Logger, error) {
	return newZapLogger(name, version, level, strings.EqualFold("JSON", format))
}

func newZapLogger(name, version string, verbosityLevel int, useProductionConfig bool) (logr.Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.ConsoleSeparator = " | "
	if useProductionConfig {
		cfg = zap.NewProductionConfig()
	}
	if verbosityLevel > 0 {
		// Zap's levels get more verbose as the number gets smaller,
		// but logr's level increases with greater numbers.
		cfg.Level = zap.NewAtomicLevelAt(zapcore.Level(verbosityLevel * -1))
	} else {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	z, err := cfg.Build()
	zap.ReplaceGlobals(z)
	if err != nil {
		return logr.Logger{}, fmt.Errorf("log config: %w", err)
	}
	logger := zapr.NewLogger(z).WithName(name)
	if useProductionConfig {
		// Append the version to each log so that logging stacks like EFK/Loki
		// can correlate errors with specific versions.
		return logger.WithValues("version", version), nil
	}
	return logger, nil
}
