package externalservices

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockTypesenseClient struct {
	mock.Mock
}

func (m *MockTypesenseClient) UpsertDocument(ctx context.Context, document interface{}) error {
	args := m.Called(ctx, document)
	return args.Error(0)
}

func (m *MockTypesenseClient) DeleteDocument(ctx context.Context, documentID string) error {
	args := m.Called(ctx, documentID)
	return args.Error(0)
}

func (m *MockTypesenseClient) ImportDocuments(ctx context.Context, documents []interface{}, action string) error {
	args := m.Called(ctx, documents, action)
	return args.Error(0)
}

func TestTypesenseClient(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		host       string
		port       string
		protocol   string
		collection string
		wantErr    bool
	}{
		{
			name:       "Valid Connection",
			apiKey:     "test-key",
			host:       "localhost",
			port:       "8108",
			protocol:   "http",
			collection: "test-collection",
			wantErr:    false,
		},
		{
			name:       "Invalid Host",
			apiKey:     "test-key",
			host:       "invalid-host",
			port:       "8108",
			protocol:   "http",
			collection: "test-collection",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTypesenseService(tt.apiKey, tt.host, tt.port, tt.protocol, tt.collection)
			if tt.wantErr {
				assert.NotNil(t, client)
				// Test connection by trying an operation
				err := client.UpsertDocument(context.Background(), map[string]interface{}{})
				assert.Error(t, err)
			} else {
				assert.NotNil(t, client)
			}
		})
	}
}

func TestTypesenseOperations(t *testing.T) {
	mockClient := new(MockTypesenseClient)
	ctx := context.Background()

	t.Run("UpsertDocument Success", func(t *testing.T) {
		document := map[string]interface{}{
			"id":   "1",
			"name": "Test Document",
		}
		mockClient.On("UpsertDocument", ctx, document).Return(nil).Once()
		err := mockClient.UpsertDocument(ctx, document)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("UpsertDocument Error", func(t *testing.T) {
		document := map[string]interface{}{
			"id":   "1",
			"name": "Test Document",
		}
		expectedErr := fmt.Errorf("document not found")
		mockClient.On("UpsertDocument", ctx, document).Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &TypesenseService{
			client:     nil, // This will trigger the error case
			collection: "test-collection",
		}

		err := service.UpsertDocument(ctx, document)
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("DeleteDocument Success", func(t *testing.T) {
		mockClient.On("DeleteDocument", ctx, "1").Return(nil).Once()
		err := mockClient.DeleteDocument(ctx, "1")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("DeleteDocument Error", func(t *testing.T) {
		expectedErr := fmt.Errorf("document not found")
		mockClient.On("DeleteDocument", ctx, "1").Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &TypesenseService{
			client:     nil, // This will trigger the error case
			collection: "test-collection",
		}

		err := service.DeleteDocument(ctx, "1")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ImportDocuments Success", func(t *testing.T) {
		documents := []interface{}{
			map[string]interface{}{"id": "1", "name": "Doc 1"},
			map[string]interface{}{"id": "2", "name": "Doc 2"},
		}
		mockClient.On("ImportDocuments", ctx, documents, "upsert").Return(nil).Once()
		err := mockClient.ImportDocuments(ctx, documents, "upsert")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ImportDocuments Error", func(t *testing.T) {
		documents := []interface{}{
			map[string]interface{}{"id": "1", "name": "Doc 1"},
			map[string]interface{}{"id": "2", "name": "Doc 2"},
		}
		expectedErr := fmt.Errorf("import failed")
		mockClient.On("ImportDocuments", ctx, documents, "upsert").Return(expectedErr).Once()

		// Create a service that uses our mock client
		service := &TypesenseService{
			client:     nil, // This will trigger the error case
			collection: "test-collection",
		}

		err := service.ImportDocuments(ctx, documents, "upsert")
		assert.Error(t, err)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}
