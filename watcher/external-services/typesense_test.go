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

func (m *MockTypesenseClient) UpsertDocument(ctx context.Context, collection string, document interface{}) error {
	args := m.Called(ctx, collection, document)
	return args.Error(0)
}

func (m *MockTypesenseClient) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	args := m.Called(ctx, collection, documentID)
	return args.Error(0)
}

func TestTypesenseClient(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		host     string
		port     string
		protocol string
		wantErr  bool
	}{
		{
			name:     "Valid Connection",
			apiKey:   "test-key",
			host:     "localhost",
			port:     "8108",
			protocol: "http",
			wantErr:  false,
		},
		{
			name:     "Invalid Host",
			apiKey:   "test-key",
			host:     "invalid-host",
			port:     "8108",
			protocol: "http",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewTypesenseService(tt.apiKey, tt.host, tt.port, tt.protocol)
			if tt.wantErr {
				assert.NotNil(t, client)
				// Test connection by trying an operation
				err := client.UpsertDocument(context.Background(), "test", map[string]interface{}{})
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
		mockClient.On("UpsertDocument", ctx, "test-collection", document).Return(nil)
		err := mockClient.UpsertDocument(ctx, "test-collection", document)
		assert.NoError(t, err)
	})

	t.Run("UpsertDocument Error", func(t *testing.T) {
		document := map[string]interface{}{
			"id":   "1",
			"name": "Test Document",
		}
		mockClient.On("UpsertDocument", ctx, "test-collection", document).Return(fmt.Errorf("document not found"))
		err := mockClient.UpsertDocument(ctx, "test-collection", document)
		assert.Error(t, err)
	})

	t.Run("DeleteDocument Success", func(t *testing.T) {
		mockClient.On("DeleteDocument", ctx, "test-collection", "1").Return(nil)
		err := mockClient.DeleteDocument(ctx, "test-collection", "1")
		assert.NoError(t, err)
	})

	t.Run("DeleteDocument Error", func(t *testing.T) {
		mockClient.On("DeleteDocument", ctx, "test-collection", "1").Return(fmt.Errorf("document not found"))
		err := mockClient.DeleteDocument(ctx, "test-collection", "1")
		assert.Error(t, err)
	})
}
