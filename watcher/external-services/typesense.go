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
	Collections() TypesenseCollectionsInterface
}

// TypesenseCollectionsInterface abstracts the collections methods used by the service
type TypesenseCollectionsInterface interface {
	Create(ctx context.Context, schema *api.CollectionSchema) (interface{}, error)
	Retrieve(ctx context.Context) (interface{}, error)
}

// TypesenseCollectionInterface abstracts the collection methods used by the service
type TypesenseCollectionInterface interface {
	Documents() TypesenseDocumentsInterface
	Document(documentID string) TypesenseDocumentInterface
	Delete() (interface{}, error)
}

type TypesenseDocumentsInterface interface {
	Upsert(ctx context.Context, document interface{}) (interface{}, error)
	Import(ctx context.Context, documents []interface{}, params *api.ImportDocumentsParams) (interface{}, error)
	Search(ctx context.Context, searchParameters *api.SearchCollectionParams) (interface{}, error)
}

type TypesenseDocumentInterface interface {
	Delete(ctx context.Context) (interface{}, error)
}

// TypesenseServiceContract defines the interface for Typesense operations
type TypesenseServiceContract interface {
	UpsertDocument(ctx context.Context, document interface{}) error
	DeleteDocument(ctx context.Context, documentID string) error
	ImportDocuments(ctx context.Context, documents []interface{}, action string) error
	SearchDocuments(ctx context.Context, searchParameters map[string]interface{}) ([]map[string]interface{}, error)
	CreateCollection(ctx context.Context, schema map[string]interface{}) error
	DeleteCollection(ctx context.Context, collectionName string) error
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

func (r *RealTypesenseClient) Collections() TypesenseCollectionsInterface {
	return &RealTypesenseCollections{inner: r.inner.Collections()}
}

// RealTypesenseCollection wraps the real typesense.CollectionInterface
type RealTypesenseCollection struct {
	inner typesense.CollectionInterface
}

func (r *RealTypesenseCollection) Documents() TypesenseDocumentsInterface {
	return &RealTypesenseDocuments{inner: r.inner.Documents()}
}

func (r *RealTypesenseCollection) Document(documentID string) TypesenseDocumentInterface {
	return &RealTypesenseDocument{inner: r.inner.Document(documentID)}
}

func (r *RealTypesenseCollection) Delete() (interface{}, error) {
	return r.inner.Delete()
}

// RealTypesenseDocuments wraps the real typesense.DocumentsInterface
type RealTypesenseDocuments struct {
	inner typesense.DocumentsInterface
}

func (r *RealTypesenseDocuments) Upsert(ctx context.Context, document interface{}) (interface{}, error) {
	return r.inner.Upsert(document)
}

func (r *RealTypesenseDocuments) Import(ctx context.Context, documents []interface{}, params *api.ImportDocumentsParams) (interface{}, error) {
	return r.inner.Import(documents, params)
}

func (r *RealTypesenseDocuments) Search(ctx context.Context, searchParameters *api.SearchCollectionParams) (interface{}, error) {
	return r.inner.Search(searchParameters)
}

// RealTypesenseDocument wraps the real typesense.DocumentInterface
type RealTypesenseDocument struct {
	inner typesense.DocumentInterface
}

func (r *RealTypesenseDocument) Delete(ctx context.Context) (interface{}, error) {
	return r.inner.Delete()
}

// RealTypesenseCollections wraps the real typesense.CollectionsInterface
type RealTypesenseCollections struct {
	inner typesense.CollectionsInterface
}

func (r *RealTypesenseCollections) Create(ctx context.Context, schema *api.CollectionSchema) (interface{}, error) {
	return r.inner.Create(schema)
}

func (r *RealTypesenseCollections) Retrieve(ctx context.Context) (interface{}, error) {
	return r.inner.Retrieve()
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

// SearchDocuments searches for documents in Typesense
func (t *TypesenseService) SearchDocuments(ctx context.Context, searchParameters map[string]interface{}) ([]map[string]interface{}, error) {
	if t.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	collection := t.client.Collection(t.collection)

	// Convert map to SearchCollectionParams
	params := &api.SearchCollectionParams{
		Q:       searchParameters["q"].(string),
		QueryBy: searchParameters["query_by"].(string),
	}

	searchResults, err := collection.Documents().Search(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert search results to []map[string]interface{}
	results := make([]map[string]interface{}, 0)

	// Type assert the search results
	if sr, ok := searchResults.(map[string]interface{}); ok {
		if hits, ok := sr["hits"].([]interface{}); ok {
			for _, hit := range hits {
				if hitMap, ok := hit.(map[string]interface{}); ok {
					if doc, ok := hitMap["document"].(map[string]interface{}); ok {
						results = append(results, doc)
					}
				}
			}
		}
	}

	return results, nil
}

func (t *TypesenseService) CreateCollection(ctx context.Context, schema map[string]interface{}) error {
	if t.client == nil {
		return fmt.Errorf("client not connected")
	}

	// Convert map to CollectionSchema
	collectionSchema := &api.CollectionSchema{
		Name:   schema["name"].(string),
		Fields: make([]api.Field, 0),
	}

	// Convert fields robustly
	if fields, ok := schema["fields"]; ok {
		switch f := fields.(type) {
		case []map[string]interface{}:
			for _, fieldMap := range f {
				collectionSchema.Fields = append(collectionSchema.Fields, api.Field{
					Name: fieldMap["name"].(string),
					Type: fieldMap["type"].(string),
				})
			}
		case []interface{}:
			for _, field := range f {
				if fieldMap, ok := field.(map[string]interface{}); ok {
					collectionSchema.Fields = append(collectionSchema.Fields, api.Field{
						Name: fieldMap["name"].(string),
						Type: fieldMap["type"].(string),
					})
				}
			}
		}
	}

	_, err := t.client.Collections().Create(ctx, collectionSchema)
	return err
}

func (t *TypesenseService) DeleteCollection(ctx context.Context, collectionName string) error {
	if t.client == nil {
		return fmt.Errorf("client not connected")
	}

	// Get the collection and delete it
	collection := t.client.Collection(collectionName)
	_, err := collection.Delete()
	return err
}

// Ensure TypesenseService implements TypesenseServiceContract
var _ TypesenseServiceContract = (*TypesenseService)(nil)
