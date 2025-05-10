package externalservices

import (
	"context"
	"encoding/json"

	"watcher/interfaces"

	"github.com/redis/go-redis/v9"
)

type RedisService struct {
	client *redis.Client
}

func NewRedisService(host, port, password string, db int) *RedisService {
	client := redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       db,
	})
	return &RedisService{client: client}
}

func (r *RedisService) RPush(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return r.client.RPush(ctx, key, data).Err()
}

func (r *RedisService) LPop(ctx context.Context, key string) (string, error) {
	return r.client.LPop(ctx, key).Result()
}

func (r *RedisService) LLen(ctx context.Context, key string) (int64, error) {
	return r.client.LLen(ctx, key).Result()
}

func (r *RedisService) ConfigSet(ctx context.Context, parameter, value string) error {
	return r.client.ConfigSet(ctx, parameter, value).Err()
}

func (r *RedisService) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

// Ensure RedisService implements interfaces.RedisClient
var _ interfaces.RedisClient = (*RedisService)(nil)
