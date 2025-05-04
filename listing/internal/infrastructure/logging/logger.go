package logging

import (
	"os"
	"path/filepath"

	"listing/internal/infrastructure/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumber "gopkg.in/natefinch/lumberjack.v2"
)

// Logger wraps zap.Logger to provide application-specific logging
type Logger struct {
	*zap.Logger
}

// New creates a new logger based on the provided configuration
func New(cfg *config.Config) (*Logger, error) {
	// Create encoder config based on environment
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set up output
	var output zapcore.WriteSyncer
	if cfg.Logging.OutputPath == "stdout" {
		output = zapcore.AddSync(os.Stdout)
	} else {
		// Ensure log directory exists
		logDir := filepath.Dir(cfg.Logging.OutputPath)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, err
		}

		// Configure log rotation
		rotator := &lumber.Logger{
			Filename:   cfg.Logging.OutputPath,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   // days
			Compress:   true, // compress rotated files
		}

		// Use both stdout and file output in development
		if cfg.Env == config.Development {
			output = zapcore.NewMultiWriteSyncer(
				zapcore.AddSync(os.Stdout),
				zapcore.AddSync(rotator),
			)
		} else {
			output = zapcore.AddSync(rotator)
		}
	}

	// Create core
	var core zapcore.Core
	if cfg.Logging.Format == "json" {
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			output,
			getZapLevel(cfg.Logging.Level),
		)
	} else {
		core = zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			output,
			getZapLevel(cfg.Logging.Level),
		)
	}

	// Create logger
	logger := zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return &Logger{logger}, nil
}

// getZapLevel converts our LogLevel to zapcore.Level
func getZapLevel(level config.LogLevel) zapcore.Level {
	switch level {
	case config.Debug:
		return zapcore.DebugLevel
	case config.Info:
		return zapcore.InfoLevel
	case config.Warning:
		return zapcore.WarnLevel
	case config.Error:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// With creates a child logger with the given fields
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{l.Logger.With(fields...)}
}

// WithError creates a child logger with an error field
func (l *Logger) WithError(err error) *Logger {
	return l.With(zap.Error(err))
}

// WithString creates a child logger with a string field
func (l *Logger) WithString(key, value string) *Logger {
	return l.With(zap.String(key, value))
}

// WithInt creates a child logger with an integer field
func (l *Logger) WithInt(key string, value int) *Logger {
	return l.With(zap.Int(key, value))
}

// WithFloat creates a child logger with a float field
func (l *Logger) WithFloat(key string, value float64) *Logger {
	return l.With(zap.Float64(key, value))
}

// WithBool creates a child logger with a boolean field
func (l *Logger) WithBool(key string, value bool) *Logger {
	return l.With(zap.Bool(key, value))
}

// WithAny creates a child logger with any field
func (l *Logger) WithAny(key string, value interface{}) *Logger {
	return l.With(zap.Any(key, value))
}
