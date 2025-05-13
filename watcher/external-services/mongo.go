package externalservices

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ChangeStream represents a MongoDB change stream
type ChangeStream interface {
	Next(ctx context.Context) bool
	Decode(val interface{}) error
	Close(ctx context.Context) error
	Err() error
}

// MongoServiceContract defines the interface for MongoDB operations
type MongoServiceContract interface {
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (ChangeStream, error)
	GetCollection() *mongo.Collection
}

type MongoService struct {
	client     *mongo.Client
	database   string
	collection string
	uri        string
}

// MongoChangeEvent represents a MongoDB change event
type MongoChangeEvent struct {
	OperationType string
	Document      interface{}
	DocumentID    string
	Timestamp     int64
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

func (m *MongoService) Watch(ctx context.Context, pipeline interface{}, opts ...*options.ChangeStreamOptions) (ChangeStream, error) {
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

func (m *MongoService) GetCollection() *mongo.Collection {
	return m.client.Database(m.database).Collection(m.collection)
}

// Ensure MongoService implements MongoServiceContract
var _ MongoServiceContract = (*MongoService)(nil)

// Ensure MongoChangeStream implements ChangeStream
var _ ChangeStream = (*MongoChangeStream)(nil)
