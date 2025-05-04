package repository_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	"listing/internal/infrastructure/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MockCollection is a mock implementation of mongo.Collection
type MockCollection struct {
	mock.Mock
	*mongo.Collection
}

func (m *MockCollection) FindOne(ctx context.Context, filter interface{}, opts ...*options.FindOneOptions) *mongo.SingleResult {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(*mongo.SingleResult)
}

func (m *MockCollection) InsertOne(ctx context.Context, document interface{}, opts ...*options.InsertOneOptions) (*mongo.InsertOneResult, error) {
	args := m.Called(ctx, document, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.InsertOneResult), args.Error(1)
}

func (m *MockCollection) ReplaceOne(ctx context.Context, filter interface{}, replacement interface{}, opts ...*options.ReplaceOptions) (*mongo.UpdateResult, error) {
	args := m.Called(ctx, filter, replacement, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.UpdateResult), args.Error(1)
}

func (m *MockCollection) DeleteOne(ctx context.Context, filter interface{}, opts ...*options.DeleteOptions) (*mongo.DeleteResult, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.DeleteResult), args.Error(1)
}

func (m *MockCollection) Find(ctx context.Context, filter interface{}, opts ...*options.FindOptions) (*mongo.Cursor, error) {
	args := m.Called(ctx, filter, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*mongo.Cursor), args.Error(1)
}

func (m *MockCollection) CountDocuments(ctx context.Context, filter interface{}, opts ...*options.CountOptions) (int64, error) {
	args := m.Called(ctx, filter, opts)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockCollection) Name() string {
	return "mock_collection"
}

// TestDocument represents a test document for MongoDB operations
type TestDocument struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Name        string             `bson:"name"`
	Description string             `bson:"description"`
	CreatedAt   time.Time          `bson:"created_at"`
	UpdatedAt   time.Time          `bson:"updated_at"`
	Tags        []string           `bson:"tags"`
	Status      string             `bson:"status"`
}

// createTestDocument creates a test document with the given ID
func createTestDocument(id primitive.ObjectID) *TestDocument {
	now := time.Now().UTC()
	return &TestDocument{
		ID:          id,
		Name:        "Test Document",
		Description: "This is a test document",
		CreatedAt:   now,
		UpdatedAt:   now,
		Tags:        []string{"test", "mock"},
		Status:      "active",
	}
}

// MockCursor is a mock implementation of mongo.Cursor
type MockCursor struct {
	mock.Mock
	*mongo.Cursor
	docs []*TestDocument
	pos  int
}

func NewMockCursor(docs []*TestDocument) *MockCursor {
	return &MockCursor{
		docs: docs,
		pos:  -1,
	}
}

func (m *MockCursor) Next(ctx context.Context) bool {
	m.pos++
	return m.pos < len(m.docs)
}

func (m *MockCursor) Decode(val interface{}) error {
	if m.pos < 0 || m.pos >= len(m.docs) {
		return mongo.ErrNoDocuments
	}
	doc := val.(*TestDocument)
	*doc = *m.docs[m.pos]
	return nil
}

func (m *MockCursor) Close(ctx context.Context) error {
	return nil
}

func (m *MockCursor) All(ctx context.Context, results interface{}) error {
	resultsVal := reflect.ValueOf(results).Elem()
	for _, doc := range m.docs {
		resultsVal.Set(reflect.Append(resultsVal, reflect.ValueOf(doc)))
	}
	return nil
}

func TestMongoRepository(t *testing.T) {
	// Create test data
	id := primitive.NewObjectID()
	doc := createTestDocument(id)

	// Create mock collection
	mockColl := &MockCollection{}

	// Create repository instance
	ctx := context.Background()
	repo, err := repository.NewMongoRepository[TestDocument](ctx, "mongodb://localhost:27017", "test_db", "test_collection")
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Replace the collection with our mock
	repo.SetCollection(mockColl)

	// Test GetByID
	t.Run("GetByID", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("FindOne", mock.Anything, bson.M{"_id": id}, mock.Anything).Return(
			mongo.NewSingleResultFromDocument(doc, nil, nil),
		).Once()

		// Test GetByID
		result, err := repo.GetByID(id.Hex())
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, doc.ID, result.ID)
		assert.Equal(t, doc.Name, result.Name)
		assert.Equal(t, doc.Description, result.Description)
		assert.Equal(t, doc.Tags, result.Tags)
		assert.Equal(t, doc.Status, result.Status)
	})

	// Test Create
	t.Run("Create", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("InsertOne", mock.Anything, mock.AnythingOfType("*repository_test.TestDocument"), mock.Anything).Return(
			&mongo.InsertOneResult{InsertedID: id}, nil,
		).Once()

		// Test Create
		err := repo.Create(doc)
		assert.NoError(t, err)
	})

	// Test Update
	t.Run("Update", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("ReplaceOne", mock.Anything, bson.M{"_id": id}, doc, mock.Anything).Return(
			&mongo.UpdateResult{MatchedCount: 1}, nil,
		).Once()

		// Test Update
		err := repo.Update(doc)
		assert.NoError(t, err)
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("DeleteOne", mock.Anything, bson.M{"_id": id}, mock.Anything).Return(
			&mongo.DeleteResult{DeletedCount: 1}, nil,
		).Once()

		// Test Delete
		err := repo.Delete(id.Hex())
		assert.NoError(t, err)
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("Find", mock.Anything, bson.M{}, mock.Anything).Return(
			nil, mongo.ErrNoDocuments,
		).Once()

		// Test List
		results, err := repo.List()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no documents in result")
		assert.Nil(t, results)
	})

	// Test FindByQuery
	t.Run("FindByQuery", func(t *testing.T) {
		// Setup test data
		query := bson.M{"status": "active"}

		// Setup mock expectations
		mockColl.On("Find", mock.Anything, query, mock.Anything).Return(
			nil, mongo.ErrNoDocuments,
		).Once()

		// Test FindByQuery
		results, err := repo.FindByQuery(query)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no documents in result")
		assert.Nil(t, results)
	})

	// Test Count
	t.Run("Count", func(t *testing.T) {
		// Setup mock expectations
		mockColl.On("CountDocuments", mock.Anything, bson.M{}, mock.Anything).Return(
			int64(5), nil,
		).Once()

		// Test Count
		count, err := repo.Count()
		assert.NoError(t, err)
		assert.Equal(t, int64(5), count)
	})
}
