package externalservices

import (
	"context"
	"fmt"

	"watcher/interfaces"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoService struct {
	client     *mongo.Client
	database   string
	collection string
	uri        string
}

func NewMongoService(uri, database, collection string) *MongoService {
	return &MongoService{
		uri:        uri,
		database:   database,
		collection: collection,
	}
}

func (m *MongoService) Connect(ctx context.Context) error {
	clientOptions := options.Client().ApplyURI(m.uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return err
	}
	m.client = client
	return nil
}

func (m *MongoService) Disconnect(ctx context.Context) error {
	if m.client != nil {
		return m.client.Disconnect(ctx)
	}
	return nil
}

func (m *MongoService) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (interfaces.ChangeStream, error) {
	if m.client == nil {
		return nil, fmt.Errorf("client not connected")
	}

	collection := m.client.Database(m.database).Collection(m.collection)
	stream, err := collection.Watch(ctx, pipeline, opts...)
	if err != nil {
		return nil, err
	}

	return &MongoChangeStream{stream: stream}, nil
}

type MongoChangeStream struct {
	stream *mongo.ChangeStream
}

func (m *MongoChangeStream) Next(ctx context.Context) bool {
	return m.stream.Next(ctx)
}

func (m *MongoChangeStream) Decode(val interface{}) error {
	return m.stream.Decode(val)
}

func (m *MongoChangeStream) Close(ctx context.Context) error {
	return m.stream.Close(ctx)
}

func (m *MongoChangeStream) Err() error {
	return m.stream.Err()
}

// Ensure MongoService implements interfaces.MongoDBClient
var _ interfaces.MongoDBClient = (*MongoService)(nil)

// Ensure MongoChangeStream implements interfaces.ChangeStream
var _ interfaces.ChangeStream = (*MongoChangeStream)(nil)
