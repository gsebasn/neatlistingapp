package externalservices

import (
	"context"
	"fmt"
	"time"

	"watcher/interfaces"

	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
)

type TypesenseService struct {
	client *typesense.Client
}

func NewTypesenseService(apiKey, host, port, protocol string) *TypesenseService {
	client := typesense.NewClient(
		typesense.WithAPIKey(apiKey),
		typesense.WithServer(fmt.Sprintf("%s://%s:%s", protocol, host, port)),
		typesense.WithConnectionTimeout(5*time.Second),
	)
	return &TypesenseService{client: client}
}

func (t *TypesenseService) UpsertDocument(ctx context.Context, collection string, document interface{}) error {
	_, err := t.client.Collection(collection).Documents().Upsert(ctx, document)
	return err
}

func (t *TypesenseService) DeleteDocument(ctx context.Context, collection string, documentID string) error {
	_, err := t.client.Collection(collection).Document(documentID).Delete(ctx)
	return err
}

func (t *TypesenseService) ImportDocuments(ctx context.Context, collection string, documents []interface{}, action string) error {
	_, err := t.client.Collection(collection).Documents().Import(ctx, documents, &api.ImportDocumentsParams{
		Action: &action,
	})
	return err
}

// Ensure TypesenseService implements interfaces.TypesenseClient
var _ interfaces.TypesenseClient = (*TypesenseService)(nil)
