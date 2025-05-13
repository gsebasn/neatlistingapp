package externalservices

import (
	"context"
	"fmt"
	"time"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

// TypesenseServiceContract defines the interface for Typesense operations
type TypesenseServiceContract interface {
	UpsertDocument(ctx context.Context, document interface{}) error
	DeleteDocument(ctx context.Context, documentID string) error
}

type TypesenseService struct {
	client     *typesense.Client
	collection string
}

func NewTypesenseService(apiKey, host, port, protocol, collection string) *TypesenseService {
	client := typesense.NewClient(
		typesense.WithAPIKey(apiKey),
		typesense.WithServer(fmt.Sprintf("%s://%s:%s", protocol, host, port)),
		typesense.WithConnectionTimeout(5*time.Second),
	)
	return &TypesenseService{
		client:     client,
		collection: collection,
	}
}

func (t *TypesenseService) UpsertDocument(ctx context.Context, document interface{}) error {
	_, err := t.client.Collection(t.collection).Documents().Upsert(ctx, document)
	return err
}

func (t *TypesenseService) DeleteDocument(ctx context.Context, documentID string) error {
	_, err := t.client.Collection(t.collection).Document(documentID).Delete(ctx)
	return err
}

func (t *TypesenseService) ImportDocuments(ctx context.Context, documents []interface{}, action string) error {
	_, err := t.client.Collection(t.collection).Documents().Import(ctx, documents, &api.ImportDocumentsParams{
		Action: &action,
	})
	return err
}

// Ensure TypesenseService implements TypesenseServiceContract
var _ TypesenseServiceContract = (*TypesenseService)(nil)
