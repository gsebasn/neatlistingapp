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
	MongoURI        string
	MongoDatabase   string
	MongoCollection string
	Typesense       TypesenseConfig
	MaxBufferSize   int
	BatchSize       int
	FlushInterval   int
	PrimaryRedis    RedisConfig
	BackupRedis     RedisConfig
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

	flushInterval, err := strconv.Atoi(os.Getenv("FLUSH_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid FLUSH_INTERVAL: %v", err)
	}

	primaryRedisDB, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_DB: %v", err)
	}

	backupRedisDB, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_DB"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_DB: %v", err)
	}

	primaryPoolSize, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_POOL_SIZE: %v", err)
	}

	backupPoolSize, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_POOL_SIZE"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_POOL_SIZE: %v", err)
	}

	primaryMinIdleConns, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	backupMinIdleConns, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_MIN_IDLE_CONNS"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_MIN_IDLE_CONNS: %v", err)
	}

	primaryMaxRetries, err := strconv.Atoi(os.Getenv("PRIMARY_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_MAX_RETRIES: %v", err)
	}

	backupMaxRetries, err := strconv.Atoi(os.Getenv("BACKUP_REDIS_MAX_RETRIES"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_MAX_RETRIES: %v", err)
	}

	primaryRetryBackoff, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_RETRY_BACKOFF: %v", err)
	}

	backupRetryBackoff, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_RETRY_BACKOFF"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_RETRY_BACKOFF: %v", err)
	}

	primaryConnMaxLifetime, err := time.ParseDuration(os.Getenv("PRIMARY_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_CONN_MAX_LIFETIME: %v", err)
	}

	backupConnMaxLifetime, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_CONN_MAX_LIFETIME"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_CONN_MAX_LIFETIME: %v", err)
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

	backupSaveInterval, err := time.ParseDuration(os.Getenv("BACKUP_REDIS_SAVE_INTERVAL"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_SAVE_INTERVAL: %v", err)
	}

	primaryAppendOnly, err := strconv.ParseBool(os.Getenv("PRIMARY_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid PRIMARY_REDIS_APPEND_ONLY: %v", err)
	}

	backupAppendOnly, err := strconv.ParseBool(os.Getenv("BACKUP_REDIS_APPEND_ONLY"))
	if err != nil {
		return nil, fmt.Errorf("invalid BACKUP_REDIS_APPEND_ONLY: %v", err)
	}

	return &Config{
		MongoURI:        os.Getenv("MONGODB_URI"),
		MongoDatabase:   os.Getenv("MONGODB_DATABASE"),
		MongoCollection: os.Getenv("MONGODB_COLLECTION"),
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
		MaxBufferSize: maxBufferSize,
		BatchSize:     batchSize,
		FlushInterval: flushInterval,
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
		BackupRedis: RedisConfig{
			Host:            os.Getenv("BACKUP_REDIS_HOST"),
			Port:            os.Getenv("BACKUP_REDIS_PORT"),
			Password:        os.Getenv("BACKUP_REDIS_PASSWORD"),
			DB:              backupRedisDB,
			QueueKey:        os.Getenv("BACKUP_REDIS_QUEUE_KEY"),
			PoolSize:        backupPoolSize,
			MinIdleConns:    backupMinIdleConns,
			MaxRetries:      backupMaxRetries,
			RetryBackoff:    backupRetryBackoff,
			ConnMaxLifetime: backupConnMaxLifetime,
			SaveInterval:    backupSaveInterval,
			AppendOnly:      backupAppendOnly,
			AppendFilename:  os.Getenv("BACKUP_REDIS_APPEND_FILENAME"),
			RDBFilename:     os.Getenv("BACKUP_REDIS_RDB_FILENAME"),
		},
	}, nil
}
