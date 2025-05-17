package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type RedisConfig struct {
	Host            string
	Port            string
	Password        string
	DB              int
	QueueKey        string
	PoolSize        int
	MinIdleConns    int
	MaxRetries      int
	RetryBackoff    time.Duration
	ConnMaxLifetime time.Duration
	// Persistence settings
	SaveInterval   time.Duration
	AppendOnly     bool
	AppendFilename string
	RDBFilename    string
}

type TypesenseConfig struct {
	APIKey            string
	Host              string
	Port              string
	Protocol          string
	CollectionName    string
	ConnectionTimeout time.Duration
	KeepAliveTimeout  time.Duration
	MaxRetries        int
	RetryBackoff      time.Duration
}

type Config struct {
	// Primary MongoDB (for watching changes)
	PrimaryMongoURI        string
	PrimaryMongoDatabase   string
	PrimaryMongoCollection string

	// Fallback MongoDB (for backup storage)
	FallbackMongoURI        string
	FallbackMongoDatabase   string
	FallbackMongoCollection string

	Typesense       TypesenseConfig
	MaxBufferSize   int
	BatchSize       int
	ProcessInterval int
	FlushInterval   int
	PrimaryRedis    RedisConfig
	SecondaryRedis  RedisConfig

	// Rate limiting configuration
	RateLimiting struct {
		GlobalRatePerSecond      float64
		InsertRatePerSecond      float64
		UpdateRatePerSecond      float64
		DeleteRatePerSecond      float64
		BurstMultiplier          float64
		OperationBurstMultiplier float64
	}
}

func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	godotenv.Load() // Ignore error if file doesn't exist

	maxBufferSize, err := strconv.Atoi(os.Getenv("MAX_BUFFER_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_BUFFER_SIZE: %v", err)
	}

	batchSize, err := strconv.Atoi(os.Getenv("BATCH_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid BATCH_SIZE: %v", err)
	}

	processInterval, err := strconv.Atoi(os.Getenv("PROCESS_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid PROCESS_INTERVAL: %v", err)
	}

	flushInterval, err := strconv.Atoi(os.Getenv("FLUSH_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid FLUSH_INTERVAL: %v", err)
	}

	primaryRedisDB, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_DB: %v", err)
	}

	secondaryRedisDB, err := strconv.Atoi(os.Getenv("SECONDARY_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_DB: %v", err)
	}

	primaryPoolSize, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_POOL_SIZE: %v", err)
	}

	secondaryPoolSize, err := strconv.Atoi(os.Getenv("SECONDARY_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_POOL_SIZE: %v", err)
	}

	primaryMinIdleConns, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	secondaryMinIdleConns, err := strconv.Atoi(os.Getenv("SECONDARY_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	primaryMaxRetries, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MAX_RETRIES: %v", err)
	}

	secondaryMaxRetries, err := strconv.Atoi(os.Getenv("SECONDARY_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_MAX_RETRIES: %v", err)
	}

	primaryRetryBackoff, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_RETRY_BACKOFF: %v", err)
	}

	secondaryRetryBackoff, err := time.ParseDuration(os.Getenv("SECONDARY_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_RETRY_BACKOFF: %v", err)
	}

	primaryConnMaxLifetime, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_CONN_MAX_LIFETIME: %v", err)
	}

	secondaryConnMaxLifetime, err := time.ParseDuration(os.Getenv("SECONDARY_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_CONN_MAX_LIFETIME: %v", err)
	}

	typesenseConnectionTimeout, err := time.ParseDuration(os.Getenv("TYPESENSE_CONNECTION_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_CONNECTION_TIMEOUT: %v", err)
	}

	typesenseKeepAliveTimeout, err := time.ParseDuration(os.Getenv("TYPESENSE_KEEPALIVE_TIMEOUT"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_KEEPALIVE_TIMEOUT: %v", err)
	}

	typesenseMaxRetries, err := strconv.Atoi(os.Getenv("TYPESENSE_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_MAX_RETRIES: %v", err)
	}

	typesenseRetryBackoff, err := time.ParseDuration(os.Getenv("TYPESENSE_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid TYPESENSE_RETRY_BACKOFF: %v", err)
	}

	// Load Redis persistence settings
	primarySaveInterval, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_SAVE_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_SAVE_INTERVAL: %v", err)
	}

	secondarySaveInterval, err := time.ParseDuration(os.Getenv("SECONDARY_REDIS_SAVE_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_SAVE_INTERVAL: %v", err)
	}

	primaryAppendOnly, err := strconv.ParseBool(os.Getenv("PRIMARY_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_APPEND_ONLY: %v", err)
	}

	secondaryAppendOnly, err := strconv.ParseBool(os.Getenv("SECONDARY_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid SECONDARY_REDIS_APPEND_ONLY: %v", err)
	}

	// Load rate limiting settings
	globalRatePerSecond, err := strconv.ParseFloat(os.Getenv("GLOBAL_RATE_PER_SECOND"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid GLOBAL_RATE_PER_SECOND: %v", err)
	}

	insertRatePerSecond, err := strconv.ParseFloat(os.Getenv("INSERT_RATE_PER_SECOND"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid INSERT_RATE_PER_SECOND: %v", err)
	}

	updateRatePerSecond, err := strconv.ParseFloat(os.Getenv("UPDATE_RATE_PER_SECOND"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid UPDATE_RATE_PER_SECOND: %v", err)
	}

	deleteRatePerSecond, err := strconv.ParseFloat(os.Getenv("DELETE_RATE_PER_SECOND"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid DELETE_RATE_PER_SECOND: %v", err)
	}

	burstMultiplier, err := strconv.ParseFloat(os.Getenv("RATE_LIMIT_BURST_MULTIPLIER"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_BURST_MULTIPLIER: %v", err)
	}

	operationBurstMultiplier, err := strconv.ParseFloat(os.Getenv("OPERATION_RATE_LIMIT_BURST_MULTIPLIER"), 64)
	if err != nil {
		return nil, fmt.Errorf("invalid OPERATION_RATE_LIMIT_BURST_MULTIPLIER: %v", err)
	}

	return &Config{
		// Primary MongoDB config
		PrimaryMongoURI:        os.Getenv("PRIMARY_MONGODB_URI"),
		PrimaryMongoDatabase:   os.Getenv("PRIMARY_MONGODB_DATABASE"),
		PrimaryMongoCollection: os.Getenv("PRIMARY_MONGODB_COLLECTION"),

		// Fallback MongoDB config
		FallbackMongoURI:        os.Getenv("FALLBACK_MONGODB_URI"),
		FallbackMongoDatabase:   os.Getenv("FALLBACK_MONGODB_DATABASE"),
		FallbackMongoCollection: os.Getenv("FALLBACK_MONGODB_COLLECTION"),

		Typesense: TypesenseConfig{
			APIKey:            os.Getenv("TYPESENSE_API_KEY"),
			Host:              os.Getenv("TYPESENSE_HOST"),
			Port:              os.Getenv("TYPESENSE_PORT"),
			Protocol:          os.Getenv("TYPESENSE_PROTOCOL"),
			CollectionName:    os.Getenv("TYPESENSE_COLLECTION_NAME"),
			ConnectionTimeout: typesenseConnectionTimeout,
			KeepAliveTimeout:  typesenseKeepAliveTimeout,
			MaxRetries:        typesenseMaxRetries,
			RetryBackoff:      typesenseRetryBackoff,
		},
		MaxBufferSize:   maxBufferSize,
		BatchSize:       batchSize,
		ProcessInterval: processInterval,
		FlushInterval:   flushInterval,
		PrimaryRedis: RedisConfig{
			Host:            os.Getenv("PRIMARY_REDIS_HOST"),
			Port:            os.Getenv("PRIMARY_REDIS_PORT"),
			Password:        os.Getenv("PRIMARY_REDIS_PASSWORD"),
			DB:              primaryRedisDB,
			QueueKey:        os.Getenv("PRIMARY_REDIS_QUEUE_KEY"),
			PoolSize:        primaryPoolSize,
			MinIdleConns:    primaryMinIdleConns,
			MaxRetries:      primaryMaxRetries,
			RetryBackoff:    primaryRetryBackoff,
			ConnMaxLifetime: primaryConnMaxLifetime,
			SaveInterval:    primarySaveInterval,
			AppendOnly:      primaryAppendOnly,
			AppendFilename:  os.Getenv("PRIMARY_REDIS_APPEND_FILENAME"),
			RDBFilename:     os.Getenv("PRIMARY_REDIS_RDB_FILENAME"),
		},
		SecondaryRedis: RedisConfig{
			Host:            os.Getenv("SECONDARY_REDIS_HOST"),
			Port:            os.Getenv("SECONDARY_REDIS_PORT"),
			Password:        os.Getenv("SECONDARY_REDIS_PASSWORD"),
			DB:              secondaryRedisDB,
			QueueKey:        os.Getenv("SECONDARY_REDIS_QUEUE_KEY"),
			PoolSize:        secondaryPoolSize,
			MinIdleConns:    secondaryMinIdleConns,
			MaxRetries:      secondaryMaxRetries,
			RetryBackoff:    secondaryRetryBackoff,
			ConnMaxLifetime: secondaryConnMaxLifetime,
			SaveInterval:    secondarySaveInterval,
			AppendOnly:      secondaryAppendOnly,
			AppendFilename:  os.Getenv("SECONDARY_REDIS_APPEND_FILENAME"),
			RDBFilename:     os.Getenv("SECONDARY_REDIS_RDB_FILENAME"),
		},
		RateLimiting: struct {
			GlobalRatePerSecond      float64
			InsertRatePerSecond      float64
			UpdateRatePerSecond      float64
			DeleteRatePerSecond      float64
			BurstMultiplier          float64
			OperationBurstMultiplier float64
		}{
			GlobalRatePerSecond:      globalRatePerSecond,
			InsertRatePerSecond:      insertRatePerSecond,
			UpdateRatePerSecond:      updateRatePerSecond,
			DeleteRatePerSecond:      deleteRatePerSecond,
			BurstMultiplier:          burstMultiplier,
			OperationBurstMultiplier: operationBurstMultiplier,
		},
	}, nil
}
