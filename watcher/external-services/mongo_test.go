package externalservices

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MockMongoClient struct {
	mock.Mock
}

func (m *MockMongoClient) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (*mongo.ChangeStream, error) {
	args := m.Called(ctx, pipeline, opts)
	return args.Get(0).(*mongo.ChangeStream), args.Error(1)
}

func (m *MockMongoClient) Database(name string, opts ...*options.DatabaseOptions) *mongo.Database {
	args := m.Called(name, opts)
	return args.Get(0).(*mongo.Database)
}

func TestMongoWatcher(t *testing.T) {
	tests := []struct {
		name       string
		uri        string
		database   string
		collection string
		wantErr    bool
	}{
		{
			name:       "Valid Connection",
			uri:        "mongodb://localhost:27017",
			database:   "testdb",
			collection: "testcoll",
			wantErr:    false,
		},
		{
			name:       "Invalid URI",
			uri:        "invalid-uri",
			database:   "testdb",
			collection: "testcoll",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewMongoService(tt.uri, tt.database, tt.collection)
			assert.NotNil(t, service)

			err := service.Connect(context.Background())
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMongoOperations(t *testing.T) {
	mockClient := new(MockMongoClient)
	ctx := context.Background()

	t.Run("Watch Success", func(t *testing.T) {
		changeStream := &mongo.ChangeStream{}
		mockClient.On("Watch", ctx, mock.Anything, mock.Anything).Return(changeStream, nil)
		stream, err := mockClient.Watch(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, stream)
	})

	t.Run("Watch Error", func(t *testing.T) {
		mockClient.On("Watch", ctx, mock.Anything, mock.Anything).Return(nil, mongo.ErrClientDisconnected)
		stream, err := mockClient.Watch(ctx, nil)
		assert.Error(t, err)
		assert.Nil(t, stream)
	})

	t.Run("Database Success", func(t *testing.T) {
		db := &mongo.Database{}
		mockClient.On("Database", "testdb", mock.Anything).Return(db)
		result := mockClient.Database("testdb")
		assert.NotNil(t, result)
	})
}
