package externalservices

import (
	"context"
	"fmt"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// TypesenseClientInterface abstracts the typesense.Client methods used by the service
type TypesenseClientInterface interface {
	Collection(collectionName string) TypesenseCollectionInterface
}

// TypesenseCollectionInterface abstracts the collection methods used by the service
type TypesenseCollectionInterface interface {
	Documents() TypesenseDocumentsInterface
	Document(documentID string) TypesenseDocumentInterface
}

type TypesenseDocumentsInterface interface {
	Upsert(ctx context.Context, document interface{}) (interface{}, error)
	Import(ctx context.Context, documents []interface{}, params *api.ImportDocumentsParams) (interface{}, error)
}

type TypesenseDocumentInterface interface {
	Delete(ctx context.Context) (interface{}, error)
}

// TypesenseServiceContract defines the interface for Typesense operations
type TypesenseServiceContract interface {
	UpsertDocument(ctx context.Context, document interface{}) error
	DeleteDocument(ctx context.Context, documentID string) error
	ImportDocuments(ctx context.Context, documents []interface{}, action string) error
}

type TypesenseService struct {
	client     TypesenseClientInterface
	collection string
}

// RealTypesenseClient wraps the real *typesense.Client to implement TypesenseClientInterface
type RealTypesenseClient struct {
	inner *typesense.Client
}

func (r *RealTypesenseClient) Collection(collectionName string) TypesenseCollectionInterface {
	return &RealTypesenseCollection{inner: r.inner.Collection(collectionName)}
}

// RealTypesenseCollection wraps the real typesense.CollectionInterface[map[string]interface{}]
type RealTypesenseCollection struct {
	inner typesense.CollectionInterface[map[string]interface{}]
}

func (r *RealTypesenseCollection) Documents() TypesenseDocumentsInterface {
	return &RealTypesenseDocuments{inner: r.inner.Documents()}
}

func (r *RealTypesenseCollection) Document(documentID string) TypesenseDocumentInterface {
	return &RealTypesenseDocument{inner: r.inner.Document(documentID)}
}

// RealTypesenseDocuments wraps the real typesense.DocumentsInterface (non-generic)
type RealTypesenseDocuments struct {
	inner typesense.DocumentsInterface
}

func (r *RealTypesenseDocuments) Upsert(ctx context.Context, document interface{}) (interface{}, error) {
	return r.inner.Upsert(ctx, document)
}

func (r *RealTypesenseDocuments) Import(ctx context.Context, documents []interface{}, params *api.ImportDocumentsParams) (interface{}, error) {
	return r.inner.Import(ctx, documents, params)
}

// RealTypesenseDocument wraps the real typesense.DocumentInterface[map[string]interface{}]
type RealTypesenseDocument struct {
	inner typesense.DocumentInterface[map[string]interface{}]
}

func (r *RealTypesenseDocument) Delete(ctx context.Context) (interface{}, error) {
	return r.inner.Delete(ctx)
}

// Update NewTypesenseService to use the wrapper
func NewTypesenseService(apiKey, host, port, protocol, collection string) *TypesenseService {
	client := typesense.NewClient(
		typesense.WithAPIKey(apiKey),
		typesense.WithServer(fmt.Sprintf("%s://%s:%s", protocol, host, port)),
		typesense.WithConnectionTimeout(5*time.Second),
	)
	return &TypesenseService{
		client:     &RealTypesenseClient{inner: client},
		collection: collection,
	}
}

func (t *TypesenseService) UpsertDocument(ctx context.Context, document interface{}) error {
	if t.client == nil {
		return fmt.Errorf("client not connected")
	}
	collection := t.client.Collection(t.collection)
	_, err := collection.Documents().Upsert(ctx, document)
	return err
}

func (t *TypesenseService) DeleteDocument(ctx context.Context, documentID string) error {
	if t.client == nil {
		return fmt.Errorf("client not connected")
	}
	collection := t.client.Collection(t.collection)
	_, err := collection.Document(documentID).Delete(ctx)
	return err
}

func (t *TypesenseService) ImportDocuments(ctx context.Context, documents []interface{}, action string) error {
	if t.client == nil {
		return fmt.Errorf("client not connected")
	}
	_, err := t.client.Collection(t.collection).Documents().Import(ctx, documents, &api.ImportDocumentsParams{
		Action: &action,
	})
	return err
}

// Ensure TypesenseService implements TypesenseServiceContract
var _ TypesenseServiceContract = (*TypesenseService)(nil)
