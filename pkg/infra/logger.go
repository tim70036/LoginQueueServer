package infra

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Allow changing log level at run time.
	LoggerLevel = zap.NewAtomicLevelAt(zapcore.InfoLevel)
)

type LoggerFactory struct {
	baseLogger *zap.Logger
}

func (f *LoggerFactory) Create(name string) *zap.Logger {
	return f.baseLogger.Named(name)
}

func ProvideLoggerFactory() *LoggerFactory {
	// See the documentation for Config and zapcore.EncoderConfig for all the
	// available options.
	var cfg = zap.Config{
		Level:            LoggerLevel,
		Development:      false,
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
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
			EncodeDuration: zapcore.MillisDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
	}
	logger := zap.Must(cfg.Build())
	logger.Info("logger created")

	return &LoggerFactory{
		baseLogger: logger,
	}
}
