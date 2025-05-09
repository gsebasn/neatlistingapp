package main

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupConfigTestEnv() {
	// MongoDB
	os.Setenv("MONGODB_URI", "mongodb://localhost:27017")
	os.Setenv("MONGODB_DATABASE", "testdb")
	os.Setenv("MONGODB_COLLECTION", "testcoll")

	// Buffer settings
	os.Setenv("MAX_BUFFER_SIZE", "1000")
	os.Setenv("BATCH_SIZE", "100")
	os.Setenv("FLUSH_INTERVAL", "5")

	// Primary Redis
	os.Setenv("PRIMARY_REDIS_HOST", "localhost")
	os.Setenv("PRIMARY_REDIS_PORT", "6379")
	os.Setenv("PRIMARY_REDIS_PASSWORD", "")
	os.Setenv("PRIMARY_REDIS_DB", "0")
	os.Setenv("PRIMARY_REDIS_QUEUE_KEY", "primary-queue")
	os.Setenv("PRIMARY_REDIS_POOL_SIZE", "10")
	os.Setenv("PRIMARY_REDIS_MIN_IDLE_CONNS", "5")
	os.Setenv("PRIMARY_REDIS_MAX_RETRIES", "3")
	os.Setenv("PRIMARY_REDIS_RETRY_BACKOFF", "100ms")
	os.Setenv("PRIMARY_REDIS_CONN_MAX_LIFETIME", "30m")
	os.Setenv("PRIMARY_REDIS_SAVE_INTERVAL", "1m")
	os.Setenv("PRIMARY_REDIS_APPEND_ONLY", "true")
	os.Setenv("PRIMARY_REDIS_APPEND_FILENAME", "appendonly.aof")
	os.Setenv("PRIMARY_REDIS_RDB_FILENAME", "dump.rdb")

	// Backup Redis
	os.Setenv("BACKUP_REDIS_HOST", "localhost")
	os.Setenv("BACKUP_REDIS_PORT", "6380")
	os.Setenv("BACKUP_REDIS_PASSWORD", "")
	os.Setenv("BACKUP_REDIS_DB", "0")
	os.Setenv("BACKUP_REDIS_QUEUE_KEY", "backup-queue")
	os.Setenv("BACKUP_REDIS_POOL_SIZE", "10")
	os.Setenv("BACKUP_REDIS_MIN_IDLE_CONNS", "5")
	os.Setenv("BACKUP_REDIS_MAX_RETRIES", "3")
	os.Setenv("BACKUP_REDIS_RETRY_BACKOFF", "100ms")
	os.Setenv("BACKUP_REDIS_CONN_MAX_LIFETIME", "30m")
	os.Setenv("BACKUP_REDIS_SAVE_INTERVAL", "1m")
	os.Setenv("BACKUP_REDIS_APPEND_ONLY", "true")
	os.Setenv("BACKUP_REDIS_APPEND_FILENAME", "appendonly.aof")
	os.Setenv("BACKUP_REDIS_RDB_FILENAME", "dump.rdb")

	// Typesense
	os.Setenv("TYPESENSE_API_KEY", "test-key")
	os.Setenv("TYPESENSE_HOST", "localhost")
	os.Setenv("TYPESENSE_PORT", "8108")
	os.Setenv("TYPESENSE_PROTOCOL", "http")
	os.Setenv("TYPESENSE_COLLECTION_NAME", "test-collection")
	os.Setenv("TYPESENSE_CONNECTION_TIMEOUT", "5s")
	os.Setenv("TYPESENSE_KEEPALIVE_TIMEOUT", "30s")
	os.Setenv("TYPESENSE_MAX_RETRIES", "3")
	os.Setenv("TYPESENSE_RETRY_BACKOFF", "100ms")
}

func teardownConfigTestEnv() {
	// MongoDB
	os.Unsetenv("MONGODB_URI")
	os.Unsetenv("MONGODB_DATABASE")
	os.Unsetenv("MONGODB_COLLECTION")

	// Buffer settings
	os.Unsetenv("MAX_BUFFER_SIZE")
	os.Unsetenv("BATCH_SIZE")
	os.Unsetenv("FLUSH_INTERVAL")

	// Primary Redis
	os.Unsetenv("PRIMARY_REDIS_HOST")
	os.Unsetenv("PRIMARY_REDIS_PORT")
	os.Unsetenv("PRIMARY_REDIS_PASSWORD")
	os.Unsetenv("PRIMARY_REDIS_DB")
	os.Unsetenv("PRIMARY_REDIS_QUEUE_KEY")
	os.Unsetenv("PRIMARY_REDIS_POOL_SIZE")
	os.Unsetenv("PRIMARY_REDIS_MIN_IDLE_CONNS")
	os.Unsetenv("PRIMARY_REDIS_MAX_RETRIES")
	os.Unsetenv("PRIMARY_REDIS_RETRY_BACKOFF")
	os.Unsetenv("PRIMARY_REDIS_CONN_MAX_LIFETIME")
	os.Unsetenv("PRIMARY_REDIS_SAVE_INTERVAL")
	os.Unsetenv("PRIMARY_REDIS_APPEND_ONLY")
	os.Unsetenv("PRIMARY_REDIS_APPEND_FILENAME")
	os.Unsetenv("PRIMARY_REDIS_RDB_FILENAME")

	// Backup Redis
	os.Unsetenv("BACKUP_REDIS_HOST")
	os.Unsetenv("BACKUP_REDIS_PORT")
	os.Unsetenv("BACKUP_REDIS_PASSWORD")
	os.Unsetenv("BACKUP_REDIS_DB")
	os.Unsetenv("BACKUP_REDIS_QUEUE_KEY")
	os.Unsetenv("BACKUP_REDIS_POOL_SIZE")
	os.Unsetenv("BACKUP_REDIS_MIN_IDLE_CONNS")
	os.Unsetenv("BACKUP_REDIS_MAX_RETRIES")
	os.Unsetenv("BACKUP_REDIS_RETRY_BACKOFF")
	os.Unsetenv("BACKUP_REDIS_CONN_MAX_LIFETIME")
	os.Unsetenv("BACKUP_REDIS_SAVE_INTERVAL")
	os.Unsetenv("BACKUP_REDIS_APPEND_ONLY")
	os.Unsetenv("BACKUP_REDIS_APPEND_FILENAME")
	os.Unsetenv("BACKUP_REDIS_RDB_FILENAME")

	// Typesense
	os.Unsetenv("TYPESENSE_API_KEY")
	os.Unsetenv("TYPESENSE_HOST")
	os.Unsetenv("TYPESENSE_PORT")
	os.Unsetenv("TYPESENSE_PROTOCOL")
	os.Unsetenv("TYPESENSE_COLLECTION_NAME")
	os.Unsetenv("TYPESENSE_CONNECTION_TIMEOUT")
	os.Unsetenv("TYPESENSE_KEEPALIVE_TIMEOUT")
	os.Unsetenv("TYPESENSE_MAX_RETRIES")
	os.Unsetenv("TYPESENSE_RETRY_BACKOFF")
}

func TestConfigLoader(t *testing.T) {
	t.Run("Load Config Success", func(t *testing.T) {
		setupConfigTestEnv()
		defer teardownConfigTestEnv()

		config, err := LoadConfig()
		assert.NoError(t, err)
		assert.NotNil(t, config)

		// Test MongoDB config
		assert.Equal(t, "mongodb://localhost:27017", config.MongoURI)
		assert.Equal(t, "testdb", config.MongoDatabase)
		assert.Equal(t, "testcoll", config.MongoCollection)

		// Test buffer settings
		assert.Equal(t, 1000, config.MaxBufferSize)
		assert.Equal(t, 100, config.BatchSize)
		assert.Equal(t, 5, config.FlushInterval)

		// Test Primary Redis config
		assert.Equal(t, "localhost", config.PrimaryRedis.Host)
		assert.Equal(t, "6379", config.PrimaryRedis.Port)
		assert.Equal(t, "", config.PrimaryRedis.Password)
		assert.Equal(t, 0, config.PrimaryRedis.DB)
		assert.Equal(t, "primary-queue", config.PrimaryRedis.QueueKey)
		assert.Equal(t, 10, config.PrimaryRedis.PoolSize)
		assert.Equal(t, 5, config.PrimaryRedis.MinIdleConns)
		assert.Equal(t, 3, config.PrimaryRedis.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.PrimaryRedis.RetryBackoff)
		assert.Equal(t, 30*time.Minute, config.PrimaryRedis.ConnMaxLifetime)
		assert.Equal(t, time.Minute, config.PrimaryRedis.SaveInterval)
		assert.True(t, config.PrimaryRedis.AppendOnly)
		assert.Equal(t, "appendonly.aof", config.PrimaryRedis.AppendFilename)
		assert.Equal(t, "dump.rdb", config.PrimaryRedis.RDBFilename)

		// Test Backup Redis config
		assert.Equal(t, "localhost", config.BackupRedis.Host)
		assert.Equal(t, "6380", config.BackupRedis.Port)
		assert.Equal(t, "", config.BackupRedis.Password)
		assert.Equal(t, 0, config.BackupRedis.DB)
		assert.Equal(t, "backup-queue", config.BackupRedis.QueueKey)
		assert.Equal(t, 10, config.BackupRedis.PoolSize)
		assert.Equal(t, 5, config.BackupRedis.MinIdleConns)
		assert.Equal(t, 3, config.BackupRedis.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.BackupRedis.RetryBackoff)
		assert.Equal(t, 30*time.Minute, config.BackupRedis.ConnMaxLifetime)
		assert.Equal(t, time.Minute, config.BackupRedis.SaveInterval)
		assert.True(t, config.BackupRedis.AppendOnly)
		assert.Equal(t, "appendonly.aof", config.BackupRedis.AppendFilename)
		assert.Equal(t, "dump.rdb", config.BackupRedis.RDBFilename)

		// Test Typesense config
		assert.Equal(t, "test-key", config.Typesense.APIKey)
		assert.Equal(t, "localhost", config.Typesense.Host)
		assert.Equal(t, "8108", config.Typesense.Port)
		assert.Equal(t, "http", config.Typesense.Protocol)
		assert.Equal(t, "test-collection", config.Typesense.CollectionName)
		assert.Equal(t, 5*time.Second, config.Typesense.ConnectionTimeout)
		assert.Equal(t, 30*time.Second, config.Typesense.KeepAliveTimeout)
		assert.Equal(t, 3, config.Typesense.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.Typesense.RetryBackoff)
	})

	t.Run("Missing Required Environment Variables", func(t *testing.T) {
		// Don't set any environment variables
		config, err := LoadConfig()
		assert.Error(t, err)
		assert.Nil(t, config)
	})
}
