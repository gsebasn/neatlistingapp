package externalservices

import (
	"context"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) RPush(ctx context.Context, value interface{}) error {
	args := m.Called(ctx, value)
	return args.Error(0)
}

func (m *MockRedisClient) LPop(ctx context.Context) (string, error) {
	args := m.Called(ctx)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) LLen(ctx context.Context) (int64, error) {
	args := m.Called(ctx)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockRedisClient) ConfigSet(ctx context.Context, parameter, value string) error {
	args := m.Called(ctx, parameter, value)
	return args.Error(0)
}

func (m *MockRedisClient) Ping(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockRedisClientWrapper implements RedisClientInterface
type MockRedisClientWrapper struct {
	mock.Mock
}

func (m *MockRedisClientWrapper) RPush(ctx context.Context, key string, value ...interface{}) *redis.IntCmd {
	args := m.Called(ctx, key, value)
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClientWrapper) LPop(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClientWrapper) LLen(ctx context.Context, key string) *redis.IntCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.IntCmd)
}

func (m *MockRedisClientWrapper) ConfigSet(ctx context.Context, parameter, value string) *redis.StatusCmd {
	args := m.Called(ctx, parameter, value)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClientWrapper) Ping(ctx context.Context) *redis.StatusCmd {
	args := m.Called(ctx)
	return args.Get(0).(*redis.StatusCmd)
}

// Ensure MockRedisClientWrapper implements RedisClientInterface
var _ RedisClientInterface = (*MockRedisClientWrapper)(nil)

func TestRedisClient(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		port     string
		pass     string
		db       int
		queueKey string
		wantErr  bool
	}{
		{
			name:     "Valid Connection",
			host:     "localhost",
			port:     "6379",
			pass:     "",
			db:       0,
			queueKey: "test-queue",
			wantErr:  false,
		},
		{
			name:     "Invalid Host",
			host:     "invalid",
			port:     "6379",
			pass:     "",
			db:       0,
			queueKey: "test-queue",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRedisService(tt.host, tt.port, tt.pass, tt.db, tt.queueKey)
			if tt.wantErr {
				assert.NotNil(t, client)
				err := client.Ping(context.Background())
				assert.Error(t, err)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}

func TestRedisOperations(t *testing.T) {
	ctx := context.Background()

	t.Run("RPush Success", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewIntCmd(ctx)
		mockCmd.SetVal(1)
		mockWrapper.On("RPush", ctx, "test-queue", mock.Anything).Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.RPush(ctx, "test-value")
		assert.NoError(t, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("RPush Error", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewIntCmd(ctx)
		mockCmd.SetErr(redis.ErrClosed)
		mockWrapper.On("RPush", ctx, "test-queue", mock.Anything).Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.RPush(ctx, "test-value")
		assert.Error(t, err)
		assert.Equal(t, redis.ErrClosed, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("LPop Success", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStringCmd(ctx)
		mockCmd.SetVal("test-value")
		mockWrapper.On("LPop", ctx, "test-queue").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		value, err := service.LPop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", value)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("LPop Error", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStringCmd(ctx)
		mockCmd.SetErr(redis.ErrClosed)
		mockWrapper.On("LPop", ctx, "test-queue").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		value, err := service.LPop(ctx)
		assert.Error(t, err)
		assert.Empty(t, value)
		assert.Equal(t, redis.ErrClosed, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("LLen Success", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewIntCmd(ctx)
		mockCmd.SetVal(5)
		mockWrapper.On("LLen", ctx, "test-queue").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		length, err := service.LLen(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), length)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("LLen Error", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewIntCmd(ctx)
		mockCmd.SetErr(redis.ErrClosed)
		mockWrapper.On("LLen", ctx, "test-queue").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		length, err := service.LLen(ctx)
		assert.Error(t, err)
		assert.Equal(t, int64(0), length)
		assert.Equal(t, redis.ErrClosed, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("ConfigSet Success", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStatusCmd(ctx)
		mockCmd.SetVal("OK")
		mockWrapper.On("ConfigSet", ctx, "maxmemory", "2gb").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.ConfigSet(ctx, "maxmemory", "2gb")
		assert.NoError(t, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("ConfigSet Error", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStatusCmd(ctx)
		mockCmd.SetErr(redis.ErrClosed)
		mockWrapper.On("ConfigSet", ctx, "maxmemory", "2gb").Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.ConfigSet(ctx, "maxmemory", "2gb")
		assert.Error(t, err)
		assert.Equal(t, redis.ErrClosed, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("Ping Success", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStatusCmd(ctx)
		mockCmd.SetVal("PONG")
		mockWrapper.On("Ping", ctx).Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.Ping(ctx)
		assert.NoError(t, err)
		mockWrapper.AssertExpectations(t)
	})

	t.Run("Ping Error", func(t *testing.T) {
		mockWrapper := new(MockRedisClientWrapper)
		mockCmd := redis.NewStatusCmd(ctx)
		mockCmd.SetErr(redis.ErrClosed)
		mockWrapper.On("Ping", ctx).Return(mockCmd)

		service := &RedisService{
			client:   mockWrapper,
			queueKey: "test-queue",
		}

		err := service.Ping(ctx)
		assert.Error(t, err)
		assert.Equal(t, redis.ErrClosed, err)
		mockWrapper.AssertExpectations(t)
	})
}
