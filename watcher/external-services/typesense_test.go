package externalservices

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/typesense/typesense-go/typesense/api"
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

// MockTypesenseDocuments implements TypesenseDocumentsInterface
type MockTypesenseDocuments struct {
	mock.Mock
}

func (m *MockTypesenseDocuments) Upsert(ctx context.Context, document interface{}) (interface{}, error) {
	args := m.Called(ctx, document)
	return args.Get(0), args.Error(1)
}

func (m *MockTypesenseDocuments) Import(ctx context.Context, documents []interface{}, params *api.ImportDocumentsParams) (interface{}, error) {
	args := m.Called(ctx, documents, params)
	return args.Get(0), args.Error(1)
}

// MockTypesenseDocument implements TypesenseDocumentInterface
type MockTypesenseDocument struct {
	mock.Mock
}

func (m *MockTypesenseDocument) Delete(ctx context.Context) (interface{}, error) {
	args := m.Called(ctx)
	return args.Get(0), args.Error(1)
}

// MockTypesenseCollection implements TypesenseCollectionInterface
type MockTypesenseCollection struct {
	mock.Mock
	DocumentsMock TypesenseDocumentsInterface
	DocumentMock  TypesenseDocumentInterface
}

func (m *MockTypesenseCollection) Documents() TypesenseDocumentsInterface {
	return m.DocumentsMock
}

func (m *MockTypesenseCollection) Document(documentID string) TypesenseDocumentInterface {
	return m.DocumentMock
}

// MockTypesenseClientWrapper implements TypesenseClientInterface
type MockTypesenseClientWrapper struct {
	mock.Mock
	CollectionMock TypesenseCollectionInterface
}

func (m *MockTypesenseClientWrapper) Collection(collectionName string) TypesenseCollectionInterface {
	return m.CollectionMock
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
	ctx := context.Background()

	document := map[string]interface{}{
		"id":   "1",
		"name": "Test Document",
	}
	documents := []interface{}{
		map[string]interface{}{"id": "1", "name": "Doc 1"},
		map[string]interface{}{"id": "2", "name": "Doc 2"},
	}

	t.Run("UpsertDocument Success", func(t *testing.T) {
		docsMock := new(MockTypesenseDocuments)
		docsMock.On("Upsert", ctx, document).Return(document, nil).Once()
		collMock := &MockTypesenseCollection{DocumentsMock: docsMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.UpsertDocument(ctx, document)
		assert.NoError(t, err)
		docsMock.AssertExpectations(t)
	})

	t.Run("UpsertDocument Error", func(t *testing.T) {
		docsMock := new(MockTypesenseDocuments)
		docsMock.On("Upsert", ctx, document).Return(nil, fmt.Errorf("document not found")).Once()
		collMock := &MockTypesenseCollection{DocumentsMock: docsMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.UpsertDocument(ctx, document)
		assert.Error(t, err)
		assert.EqualError(t, err, "document not found")
		docsMock.AssertExpectations(t)
	})

	t.Run("DeleteDocument Success", func(t *testing.T) {
		docMock := new(MockTypesenseDocument)
		docMock.On("Delete", ctx).Return(document, nil).Once()
		collMock := &MockTypesenseCollection{DocumentMock: docMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.DeleteDocument(ctx, "1")
		assert.NoError(t, err)
		docMock.AssertExpectations(t)
	})

	t.Run("DeleteDocument Error", func(t *testing.T) {
		docMock := new(MockTypesenseDocument)
		docMock.On("Delete", ctx).Return(nil, fmt.Errorf("document not found")).Once()
		collMock := &MockTypesenseCollection{DocumentMock: docMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.DeleteDocument(ctx, "1")
		assert.Error(t, err)
		assert.EqualError(t, err, "document not found")
		docMock.AssertExpectations(t)
	})

	t.Run("ImportDocuments Success", func(t *testing.T) {
		docsMock := new(MockTypesenseDocuments)
		params := &api.ImportDocumentsParams{Action: ptrString("upsert")}
		docsMock.On("Import", ctx, documents, params).Return(nil, nil).Once()
		collMock := &MockTypesenseCollection{DocumentsMock: docsMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.ImportDocuments(ctx, documents, "upsert")
		assert.NoError(t, err)
		docsMock.AssertExpectations(t)
	})

	t.Run("ImportDocuments Error", func(t *testing.T) {
		docsMock := new(MockTypesenseDocuments)
		params := &api.ImportDocumentsParams{Action: ptrString("upsert")}
		docsMock.On("Import", ctx, documents, params).Return(nil, fmt.Errorf("import failed")).Once()
		collMock := &MockTypesenseCollection{DocumentsMock: docsMock}
		clientMock := &MockTypesenseClientWrapper{CollectionMock: collMock}
		service := &TypesenseService{client: clientMock, collection: "test-collection"}
		err := service.ImportDocuments(ctx, documents, "upsert")
		assert.Error(t, err)
		assert.EqualError(t, err, "import failed")
		docsMock.AssertExpectations(t)
	})
}

func ptrString(s string) *string { return &s }
