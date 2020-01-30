package notifiers

import (
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New creates a new zap logger with given log level
func New(level string) (*zap.Logger, error) {
	var l, err = logLevel(level)

	if err != nil {
		return nil, errors.Wrap(err, "failed to parse log level")
	}

	loggerConfig := zap.Config{
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "severity",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	loggerConfig.Level = zap.NewAtomicLevelAt(l)
	loggerConfig.DisableStacktrace = true

	return loggerConfig.Build()
}

// This can be used for testing.
func NewDiscard() *zap.Logger {
	return zap.NewNop()
}

func logLevel(level string) (zapcore.Level, error) {
	level = strings.ToUpper(level)

	var l zapcore.Level

	switch level {
	case "DEBUG":
		l = zapcore.DebugLevel

	case "INFO":
		l = zapcore.InfoLevel

	case "WARN":
		l = zapcore.WarnLevel

	case "ERROR":
		l = zapcore.ErrorLevel

	default:
		return l, errors.Errorf("invalid loglevel: %s", level)
	}

	return l, nil
}
