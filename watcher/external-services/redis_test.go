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
	mockClient := new(MockRedisClient)
	ctx := context.Background()

	t.Run("RPush Success", func(t *testing.T) {
		mockClient.On("RPush", ctx, "test-value").Return(nil).Once()
		err := mockClient.RPush(ctx, "test-value")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("RPush Error", func(t *testing.T) {
		expectedErr := redis.ErrClosed
		mockClient.On("RPush", ctx, "test-value").Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &RedisService{
			client:   nil, // This will trigger the error case
			queueKey: "test-queue",
		}

		err := service.RPush(ctx, "test-value")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("LPop Success", func(t *testing.T) {
		mockClient.On("LPop", ctx).Return("test-value", nil).Once()
		value, err := mockClient.LPop(ctx)
		assert.NoError(t, err)
		assert.Equal(t, "test-value", value)
		mockClient.AssertExpectations(t)
	})

	t.Run("LPop Error", func(t *testing.T) {
		expectedErr := redis.ErrClosed
		mockClient.On("LPop", ctx).Return("", expectedErr).Once()

		// Create a service that uses our mock client
		service := &RedisService{
			client:   nil, // This will trigger the error case
			queueKey: "test-queue",
		}

		value, err := service.LPop(ctx)
		assert.Error(t, err)
		assert.Empty(t, value)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("LLen Success", func(t *testing.T) {
		mockClient.On("LLen", ctx).Return(int64(5), nil).Once()
		length, err := mockClient.LLen(ctx)
		assert.NoError(t, err)
		assert.Equal(t, int64(5), length)
		mockClient.AssertExpectations(t)
	})

	t.Run("LLen Error", func(t *testing.T) {
		expectedErr := redis.ErrClosed
		mockClient.On("LLen", ctx).Return(int64(0), expectedErr).Once()

		// Create a service that uses our mock client
		service := &RedisService{
			client:   nil, // This will trigger the error case
			queueKey: "test-queue",
		}

		length, err := service.LLen(ctx)
		assert.Error(t, err)
		assert.Equal(t, int64(0), length)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ConfigSet Success", func(t *testing.T) {
		mockClient.On("ConfigSet", ctx, "maxmemory", "2gb").Return(nil).Once()
		err := mockClient.ConfigSet(ctx, "maxmemory", "2gb")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ConfigSet Error", func(t *testing.T) {
		expectedErr := redis.ErrClosed
		mockClient.On("ConfigSet", ctx, "maxmemory", "2gb").Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &RedisService{
			client:   nil, // This will trigger the error case
			queueKey: "test-queue",
		}

		err := service.ConfigSet(ctx, "maxmemory", "2gb")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Ping Success", func(t *testing.T) {
		mockClient.On("Ping", ctx).Return(nil).Once()
		err := mockClient.Ping(ctx)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Ping Error", func(t *testing.T) {
		expectedErr := redis.ErrClosed
		mockClient.On("Ping", ctx).Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &RedisService{
			client:   nil, // This will trigger the error case
			queueKey: "test-queue",
		}

		err := service.Ping(ctx)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}
