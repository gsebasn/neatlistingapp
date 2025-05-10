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

func (m *MockRedisClient) RPush(ctx context.Context, key string, value interface{}) error {
	args := m.Called(ctx, key, value)
	return args.Error(0)
}

func (m *MockRedisClient) LPop(ctx context.Context, key string) (string, error) {
	args := m.Called(ctx, key)
	return args.String(0), args.Error(1)
}

func (m *MockRedisClient) LLen(ctx context.Context, key string) (int64, error) {
	args := m.Called(ctx, key)
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

func TestRedisClient(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		port    string
		pass    string
		db      int
		wantErr bool
	}{
		{
			name:    "Valid Connection",
			host:    "localhost",
			port:    "6379",
			pass:    "",
			db:      0,
			wantErr: false,
		},
		{
			name:    "Invalid Host",
			host:    "invalid",
			port:    "6379",
			pass:    "",
			db:      0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewRedisService(tt.host, tt.port, tt.pass, tt.db)
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
	mockClient := new(MockRedisClient)
	ctx := context.Background()

	t.Run("RPush Success", func(t *testing.T) {
		mockClient.On("RPush", ctx, "test-key", "test-value").Return(nil)
		err := mockClient.RPush(ctx, "test-key", "test-value")
		assert.NoError(t, err)
	})

	t.Run("RPush Error", func(t *testing.T) {
		mockClient.On("RPush", ctx, "test-key", "test-value").Return(redis.ErrClosed)
		err := mockClient.RPush(ctx, "test-key", "test-value")
		assert.Error(t, err)
	})

	t.Run("LPop Success", func(t *testing.T) {
		mockClient.On("LPop", ctx, "test-key").Return("test-value", nil)
		value, err := mockClient.LPop(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, "test-value", value)
	})

	t.Run("LPop Error", func(t *testing.T) {
		mockClient.On("LPop", ctx, "test-key").Return("", redis.ErrClosed)
		value, err := mockClient.LPop(ctx, "test-key")
		assert.Error(t, err)
		assert.Empty(t, value)
	})

	t.Run("LLen Success", func(t *testing.T) {
		mockClient.On("LLen", ctx, "test-key").Return(int64(5), nil)
		length, err := mockClient.LLen(ctx, "test-key")
		assert.NoError(t, err)
		assert.Equal(t, int64(5), length)
	})

	t.Run("LLen Error", func(t *testing.T) {
		mockClient.On("LLen", ctx, "test-key").Return(int64(0), redis.ErrClosed)
		length, err := mockClient.LLen(ctx, "test-key")
		assert.Error(t, err)
		assert.Equal(t, int64(0), length)
	})

	t.Run("ConfigSet Success", func(t *testing.T) {
		mockClient.On("ConfigSet", ctx, "maxmemory", "2gb").Return(nil)
		err := mockClient.ConfigSet(ctx, "maxmemory", "2gb")
		assert.NoError(t, err)
	})

	t.Run("ConfigSet Error", func(t *testing.T) {
		mockClient.On("ConfigSet", ctx, "maxmemory", "2gb").Return(redis.ErrClosed)
		err := mockClient.ConfigSet(ctx, "maxmemory", "2gb")
		assert.Error(t, err)
	})

	t.Run("Ping Success", func(t *testing.T) {
		mockClient.On("Ping", ctx).Return(nil)
		err := mockClient.Ping(ctx)
		assert.NoError(t, err)
	})

	t.Run("Ping Error", func(t *testing.T) {
		mockClient.On("Ping", ctx).Return(redis.ErrClosed)
		err := mockClient.Ping(ctx)
		assert.Error(t, err)
	})
}
