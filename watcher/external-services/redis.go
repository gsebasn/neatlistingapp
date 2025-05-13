package externalservices

import (
	"context"
	"encoding/json"

	"github.com/redis/go-redis/v9"
)

// RedisServiceContract defines the interface for Redis operations
type RedisServiceContract interface {
	RPush(ctx context.Context, value interface{}) error
	LPop(ctx context.Context) (string, error)
	LLen(ctx context.Context) (int64, error)
	ConfigSet(ctx context.Context, parameter, value string) error
	Ping(ctx context.Context) error
}

type RedisService struct {
	client   *redis.Client
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
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.RPush(ctx, r.queueKey, data).Err()
}

func (r *RedisService) LPop(ctx context.Context) (string, error) {
	return r.client.LPop(ctx, r.queueKey).Result()
}

func (r *RedisService) LLen(ctx context.Context) (int64, error) {
	return r.client.LLen(ctx, r.queueKey).Result()
}

func (r *RedisService) ConfigSet(ctx context.Context, parameter, value string) error {
	return r.client.ConfigSet(ctx, parameter, value).Err()
}

func (r *RedisService) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Ensure RedisService implements RedisServiceContract
var _ RedisServiceContract = (*RedisService)(nil)
