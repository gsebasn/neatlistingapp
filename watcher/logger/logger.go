package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// LogLevel represents the logging level
type LogLevel string

const (
	// LogLevelDebug represents debug level logging
	LogLevelDebug LogLevel = "debug"
	// LogLevelInfo represents info level logging
	LogLevelInfo LogLevel = "info"
	// LogLevelWarn represents warn level logging
	LogLevelWarn LogLevel = "warn"
	// LogLevelError represents error level logging
	LogLevelError LogLevel = "error"
)

// Config holds the logging configuration
type Config struct {
	Level      LogLevel
	TimeFormat string
	Pretty     bool
}

// DefaultConfig returns the default logging configuration
func DefaultConfig() *Config {
	return &Config{
		Level:      LogLevelInfo,
		TimeFormat: time.RFC3339,
		Pretty:     false,
	}
}

// Init initializes the global logger with the given configuration
func Init(config *Config) {
	// Set the global log level
	level, err := zerolog.ParseLevel(string(config.Level))
	if err != nil {
		level = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(level)

	// Configure time format
	zerolog.TimeFieldFormat = config.TimeFormat

	// Configure output
	if config.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: config.TimeFormat,
		})
	} else {
		log.Logger = zerolog.New(os.Stdout).With().Timestamp().Logger()
	}
}

// WithContext returns a logger with the given context fields
func WithContext(fields map[string]interface{}) *zerolog.Logger {
	logger := log.With()
	for k, v := range fields {
		logger = logger.Interface(k, v)
	}
	l := logger.Logger()
	return &l
}

// Debug logs a debug message
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info logs an info message
func Info() *zerolog.Event {
	return log.Info()
}

// Warn logs a warning message
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error logs an error message
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal logs a fatal message and exits
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// Panic logs a panic message and panics
func Panic() *zerolog.Event {
	return log.Panic()
}
