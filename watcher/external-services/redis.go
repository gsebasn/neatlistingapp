package externalservices

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// RedisClientInterface defines the interface for Redis client operations
type RedisClientInterface interface {
	RPush(ctx context.Context, key string, value ...interface{}) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	ConfigSet(ctx context.Context, parameter, value string) *redis.StatusCmd
	Ping(ctx context.Context) *redis.StatusCmd
}

// RedisServiceContract defines the interface for Redis operations
type RedisServiceContract interface {
	RPush(ctx context.Context, value interface{}) error
	LPop(ctx context.Context) (string, error)
	LLen(ctx context.Context) (int64, error)
	ConfigSet(ctx context.Context, parameter, value string) error
	Ping(ctx context.Context) error
}

type RedisService struct {
	client   RedisClientInterface
	queueKey string
}

func NewRedisService(host, port, password string, db int, queueKey string) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
	return &RedisService{
		client:   client,
		queueKey: queueKey,
	}
}

func (r *RedisService) RPush(ctx context.Context, value interface{}) error {
	if r.client == nil {
		return redis.ErrClosed
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.RPush(ctx, r.queueKey, data).Err()
}

func (r *RedisService) LPop(ctx context.Context) (string, error) {
	if r.client == nil {
		return "", redis.ErrClosed
	}
	return r.client.LPop(ctx, r.queueKey).Result()
}

func (r *RedisService) LLen(ctx context.Context) (int64, error) {
	if r.client == nil {
		return 0, redis.ErrClosed
	}
	return r.client.LLen(ctx, r.queueKey).Result()
}

func (r *RedisService) ConfigSet(ctx context.Context, parameter, value string) error {
	if r.client == nil {
		return redis.ErrClosed
	}
	return r.client.ConfigSet(ctx, parameter, value).Err()
}

func (r *RedisService) Ping(ctx context.Context) error {
	if r.client == nil {
		return redis.ErrClosed
	}
	return r.client.Ping(ctx).Err()
}

// Ensure RedisService implements RedisServiceContract
var _ RedisServiceContract = (*RedisService)(nil)
