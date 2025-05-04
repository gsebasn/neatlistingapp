package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoRepository is a generic repository implementation for MongoDB
type MongoRepository[T any] struct {
	client     *mongo.Client
	database   *mongo.Database
	collection *mongo.Collection
}

// NewMongoRepository creates a new instance of MongoRepository for a specific type
func NewMongoRepository[T any](ctx context.Context, uri, databaseName, collectionName string) (*MongoRepository[T], error) {
	fmt.Printf("Connecting to MongoDB: %s, database: %s, collection: %s\n", uri, databaseName, collectionName)

	// Set client options with timeout
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerSelectionTimeout(5 * time.Second)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	fmt.Println("Successfully connected to MongoDB")

	// Get database and collection
	db := client.Database(databaseName)
	collection := db.Collection(collectionName)

	// Verify collection exists and has documents
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to count documents: %v", err)
	}
	fmt.Printf("Found %d documents in collection %s\n", count, collectionName)

	return &MongoRepository[T]{
		client:     client,
		database:   db,
		collection: collection,
	}, nil
}

// GetByID retrieves a document by ID
func (r *MongoRepository[T]) GetByID(id string) (*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("invalid ID format: %v", err)
	}

	var rawDoc bson.D
	err = r.collection.FindOne(ctx, bson.M{"_id": objectID}).Decode(&rawDoc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("error finding document: %v", err)
	}

	// Convert bson.D to bson.M for easier handling
	docMap := rawDoc.Map()

	// Create a new instance of T
	var doc T
	docBytes, err := bson.Marshal(docMap)
	if err != nil {
		return nil, fmt.Errorf("error marshaling document: %v", err)
	}

	if err := bson.Unmarshal(docBytes, &doc); err != nil {
		return nil, fmt.Errorf("error unmarshaling document: %v", err)
	}

	return &doc, nil
}

// Create stores a new document
func (r *MongoRepository[T]) Create(document *T) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert document to BSON to check ID
	doc, err := bson.Marshal(document)
	if err != nil {
		return err
	}

	var docMap bson.M
	if err := bson.Unmarshal(doc, &docMap); err != nil {
		return err
	}

	// Check if document already exists
	var existing T
	err = r.collection.FindOne(ctx, bson.M{"_id": docMap["_id"]}).Decode(&existing)
	if err == nil {
		return errors.New("document already exists")
	} else if err != mongo.ErrNoDocuments {
		return err
	}

	_, err = r.collection.InsertOne(ctx, document)
	return err
}

// Update modifies an existing document
func (r *MongoRepository[T]) Update(document *T) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert document to BSON to get ID
	doc, err := bson.Marshal(document)
	if err != nil {
		return err
	}

	var docMap bson.M
	if err := bson.Unmarshal(doc, &docMap); err != nil {
		return err
	}

	result, err := r.collection.ReplaceOne(ctx, bson.M{"_id": docMap["_id"]}, document)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("document not found")
	}
	return nil
}

// Delete removes a document by ID
func (r *MongoRepository[T]) Delete(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Convert string ID to ObjectId
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return errors.New("invalid ID format")
	}

	result, err := r.collection.DeleteOne(ctx, bson.M{"_id": objectID})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return errors.New("document not found")
	}
	return nil
}

// List returns all documents
func (r *MongoRepository[T]) List() ([]*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fmt.Printf("Starting List operation on collection: %s\n", r.collection.Name())

	// Find all documents
	cursor, err := r.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("error finding documents: %v", err)
	}
	defer cursor.Close(ctx)

	// Get all raw documents first
	var rawDocs []bson.D
	if err := cursor.All(ctx, &rawDocs); err != nil {
		return nil, fmt.Errorf("error getting raw documents: %v", err)
	}

	fmt.Printf("Found %d raw documents\n", len(rawDocs))

	// Convert raw documents to target type
	documents := make([]*T, 0, len(rawDocs))
	for i, rawDoc := range rawDocs {
		// Convert bson.D to bson.M for easier handling
		docMap := rawDoc.Map()

		// Create a new instance of T
		var doc T
		docBytes, err := bson.Marshal(docMap)
		if err != nil {
			fmt.Printf("Error marshaling document %d: %v\n", i, err)
			continue
		}

		if err := bson.Unmarshal(docBytes, &doc); err != nil {
			fmt.Printf("Error unmarshaling document %d: %v\n", i, err)
			continue
		}

		documents = append(documents, &doc)
	}

	fmt.Printf("Successfully processed %d documents\n", len(documents))
	if len(documents) > 0 {
		docValue := reflect.ValueOf(documents[0]).Elem()
		fmt.Printf("First document fields: %+v\n", docValue.Interface())
	}

	return documents, nil
}

// FindByQuery returns documents matching the given query
func (r *MongoRepository[T]) FindByQuery(query bson.M) ([]*T, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := r.collection.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var documents []*T
	if err := cursor.All(ctx, &documents); err != nil {
		return nil, err
	}
	return documents, nil
}

// Count returns the total number of documents in the collection
func (r *MongoRepository[T]) Count() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.collection.CountDocuments(ctx, bson.M{})
}

// Close closes the MongoDB connection
func (r *MongoRepository[T]) Close(ctx context.Context) error {
	return r.client.Disconnect(ctx)
}

// Ping checks if the database connection is alive
func (r *MongoRepository[T]) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return r.client.Ping(ctx, nil)
}
