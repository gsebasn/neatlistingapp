package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment represents the application environment
type Environment string

const (
	Development Environment = "dev"
	Test        Environment = "test"
	Production  Environment = "prod"
)

// LogLevel represents the logging level
type LogLevel string

const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Warning LogLevel = "warning"
	Error   LogLevel = "error"
)

// MaskPattern represents how sensitive data should be masked
type MaskPattern string

const (
	FullMask   MaskPattern = "full"      // Replace entire value with mask
	FirstLast  MaskPattern = "firstlast" // Show first and last characters
	LastFour   MaskPattern = "last4"     // Show only last 4 characters
	FirstFour  MaskPattern = "first4"    // Show only first 4 characters
	MiddleMask MaskPattern = "middle"    // Show first and last, mask middle
)

// EndpointConfig holds configuration for specific endpoints
type EndpointConfig struct {
	Path          string
	BodySizeLimit int64
	MaskedFields  []string
	MaskPattern   MaskPattern
}

// Config holds all configuration for the application
type Config struct {
	Env      Environment
	Server   ServerConfig
	MongoDB  MongoDBConfig
	Security SecurityConfig
	Logging  LoggingConfig
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// MongoDBConfig holds MongoDB-related configuration
type MongoDBConfig struct {
	URI      string
	Database string
	Timeout  time.Duration
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	AllowedOrigins []string
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	Level           LogLevel
	Format          string           // json or text
	OutputPath      string           // file path or stdout
	BodySizeLimit   int64            // maximum size of request/response body to log (in bytes)
	MaskedFields    []string         // fields to mask in request/response bodies
	MaskPattern     MaskPattern      // how to mask sensitive data
	MaskReplacement string           // string to use for masking
	EndpointConfigs []EndpointConfig // per-endpoint configurations
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	env := getEnvironment()
	config := &Config{
		Env: env,
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", getDefaultPort(env)),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 5*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 120*time.Second),
		},
		MongoDB: MongoDBConfig{
			URI:      getEnv("MONGODB_URI", getDefaultMongoURI(env)),
			Database: getEnv("MONGODB_DB", getDefaultMongoDB(env)),
			Timeout:  getDurationEnv("MONGODB_TIMEOUT", 10*time.Second),
		},
		Security: SecurityConfig{
			AllowedOrigins: getStringSliceEnv("ALLOWED_ORIGINS", getDefaultOrigins(env)),
		},
		Logging: LoggingConfig{
			Level:           getLogLevelEnv("LOG_LEVEL", getDefaultLogLevel(env)),
			Format:          getEnv("LOG_FORMAT", getDefaultLogFormat(env)),
			OutputPath:      getEnv("LOG_OUTPUT", getDefaultLogOutput(env)),
			BodySizeLimit:   getInt64Env("LOG_BODY_SIZE_LIMIT", getDefaultBodySizeLimit(env)),
			MaskedFields:    getStringSliceEnv("LOG_MASKED_FIELDS", getDefaultMaskedFields(env)),
			MaskPattern:     getMaskPatternEnv("LOG_MASK_PATTERN", getDefaultMaskPattern(env)),
			MaskReplacement: getEnv("LOG_MASK_REPLACEMENT", "***"),
			EndpointConfigs: getEndpointConfigs(env),
		},
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate performs validation of the configuration
func (c *Config) Validate() error {
	// Validate Server configuration
	if c.Server.Port == "" {
		return fmt.Errorf("server port cannot be empty")
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("server read timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("server write timeout must be positive")
	}
	if c.Server.IdleTimeout <= 0 {
		return fmt.Errorf("server idle timeout must be positive")
	}

	// Validate MongoDB configuration
	if c.MongoDB.URI == "" {
		return fmt.Errorf("MongoDB URI cannot be empty")
	}
	if c.MongoDB.Database == "" {
		return fmt.Errorf("MongoDB database name cannot be empty")
	}
	if c.MongoDB.Timeout <= 0 {
		return fmt.Errorf("MongoDB timeout must be positive")
	}

	// Validate Security configuration
	if len(c.Security.AllowedOrigins) == 0 {
		return fmt.Errorf("at least one allowed origin must be specified")
	}

	// Validate Logging configuration
	if c.Logging.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}
	if c.Logging.Format != "json" && c.Logging.Format != "text" {
		return fmt.Errorf("log format must be either 'json' or 'text'")
	}
	if c.Logging.OutputPath == "" {
		return fmt.Errorf("log output path cannot be empty")
	}

	return nil
}

// getLogLevelEnv gets a log level from an environment variable or returns a default value
func getLogLevelEnv(key string, defaultValue LogLevel) LogLevel {
	value := strings.ToLower(os.Getenv(key))
	switch LogLevel(value) {
	case Debug, Info, Warning, Error:
		return LogLevel(value)
	default:
		return defaultValue
	}
}

// getDefaultLogLevel returns the default log level for the environment
func getDefaultLogLevel(env Environment) LogLevel {
	switch env {
	case Production:
		return Info
	case Test:
		return Debug
	default:
		return Debug
	}
}

// getDefaultLogFormat returns the default log format for the environment
func getDefaultLogFormat(env Environment) string {
	switch env {
	case Production:
		return "json"
	default:
		return "text"
	}
}

// getDefaultLogOutput returns the default log output for the environment
func getDefaultLogOutput(env Environment) string {
	switch env {
	case Production:
		return "/var/log/app.log"
	default:
		return "stdout"
	}
}

// getEnvironment determines the current environment
func getEnvironment() Environment {
	env := strings.ToLower(os.Getenv("APP_ENV"))
	switch env {
	case "prod", "production":
		return Production
	case "test":
		return Test
	default:
		return Development
	}
}

// getDefaultPort returns the default port for the environment
func getDefaultPort(env Environment) string {
	switch env {
	case Production:
		return "80"
	case Test:
		return "8081"
	default:
		return "8080"
	}
}

// getDefaultMongoURI returns the default MongoDB URI for the environment
func getDefaultMongoURI(env Environment) string {
	switch env {
	case Production:
		return "mongodb://mongodb:27017"
	case Test:
		return "mongodb://localhost:27017/test"
	default:
		return "mongodb://localhost:27017"
	}
}

// getDefaultMongoDB returns the default MongoDB database name for the environment
func getDefaultMongoDB(env Environment) string {
	switch env {
	case Production:
		return "realestate_prod"
	case Test:
		return "realestate_test"
	default:
		return "realestate"
	}
}

// getDefaultOrigins returns the default allowed origins for the environment
func getDefaultOrigins(env Environment) []string {
	switch env {
	case Production:
		return []string{"https://yourdomain.com"}
	case Test:
		return []string{"http://localhost:3000"}
	default:
		return []string{"http://localhost:3000", "http://localhost:8080"}
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// getDurationEnv gets a duration from an environment variable or returns a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		fmt.Printf("Warning: Invalid duration format for %s, using default: %v\n", key, defaultValue)
		return defaultValue
	}
	return duration
}

// getInt64Env gets an int64 from an environment variable or returns a default value
func getInt64Env(key string, defaultValue int64) int64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	intValue, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		fmt.Printf("Warning: Invalid int64 format for %s, using default: %d\n", key, defaultValue)
		return defaultValue
	}
	return intValue
}

// getStringSliceEnv gets a string slice from an environment variable or returns a default value
func getStringSliceEnv(key string, defaultValue []string) []string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return strings.Split(value, ",")
}

// getDefaultBodySizeLimit returns the default body size limit for the environment
func getDefaultBodySizeLimit(env Environment) int64 {
	switch env {
	case Production:
		return 1024 * 1024 // 1MB
	default:
		return 10 * 1024 * 1024 // 10MB
	}
}

// getDefaultMaskedFields returns the default masked fields for the environment
func getDefaultMaskedFields(env Environment) []string {
	switch env {
	case Production:
		return []string{"password", "token", "secret", "key", "authorization"}
	case Test:
		return []string{"password", "token"}
	default:
		return []string{"password"}
	}
}

// getDefaultMaskPattern returns the default mask pattern for the environment
func getDefaultMaskPattern(env Environment) MaskPattern {
	switch env {
	case Production:
		return FullMask
	case Test:
		return LastFour
	default:
		return FirstLast
	}
}

// getMaskPatternEnv gets a mask pattern from an environment variable or returns a default value
func getMaskPatternEnv(key string, defaultValue MaskPattern) MaskPattern {
	value := strings.ToLower(os.Getenv(key))
	switch MaskPattern(value) {
	case FullMask, FirstLast, LastFour, FirstFour, MiddleMask:
		return MaskPattern(value)
	default:
		return defaultValue
	}
}

// getEndpointConfigs returns endpoint-specific configurations
func getEndpointConfigs(env Environment) []EndpointConfig {
	switch env {
	case Production:
		return []EndpointConfig{
			{
				Path:          "/api/v1/users",
				BodySizeLimit: 512 * 1024, // 512KB
				MaskedFields:  []string{"password", "token", "secret", "ssn"},
				MaskPattern:   FullMask,
			},
			{
				Path:          "/api/v1/payments",
				BodySizeLimit: 256 * 1024, // 256KB
				MaskedFields:  []string{"credit_card", "cvv", "password"},
				MaskPattern:   LastFour,
			},
		}
	case Test:
		return []EndpointConfig{
			{
				Path:          "/api/v1/users",
				BodySizeLimit: 1024 * 1024, // 1MB
				MaskedFields:  []string{"password", "token"},
				MaskPattern:   LastFour,
			},
		}
	default:
		return []EndpointConfig{
			{
				Path:          "/api/v1/users",
				BodySizeLimit: 5 * 1024 * 1024, // 5MB
				MaskedFields:  []string{"password"},
				MaskPattern:   FirstLast,
			},
		}
	}
}
