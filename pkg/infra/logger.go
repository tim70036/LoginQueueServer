package infra

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	BaseLogger = NewLogger()

	// Allow changing log level at run time.
	LoggerLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
)

func NewLogger() *zap.Logger {
	// See the documentation for Config and zapcore.EncoderConfig for all the
	// available options.
	var cfg = zap.Config{
		Level:            LoggerLevel,
		Development:      false,
		Encoding:         "console",
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
		EncoderConfig: zapcore.EncoderConfig{
			// Keys can be anything except the empty string.
			TimeKey:  "time",
			LevelKey: "level",
			NameKey:  "name",
			// CallerKey:      "caller",
			// FunctionKey:    "function",
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalColorLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.StringDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}
	logger := zap.Must(cfg.Build())
	logger.Info("logger created")
	return logger
}
