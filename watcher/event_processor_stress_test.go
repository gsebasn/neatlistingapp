package main

// make sure to run this before running the test:
// sudo mkdir -p /private/tmp/typesense-data
// sudo chmod 777 /private/tmp/typesense-data

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	externalservices "watcher/external-services"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type stressTestContainers struct {
	mongoContainer     testcontainers.Container
	redisContainer     testcontainers.Container
	typesenseContainer testcontainers.Container
}

// setupStressTestContainers sets up test containers for stress testing.
// It returns a mongo client (for fallback mongo), containers struct, and a cleanup function.
func setupStressTestContainers(t *testing.T) (*mongo.Client, *stressTestContainers, func()) {
	ctx := context.Background()

	containers := &stressTestContainers{}

	// Start MongoDB container
	mongoReq := testcontainers.ContainerRequest{
		Image:        "mongo:6.0",
		ExposedPorts: []string{"27017/tcp"},
		WaitingFor:   wait.ForListeningPort("27017/tcp"),
	}
	mongoContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: mongoReq,
		Started:          true,
	})
	require.NoError(t, err)
	containers.mongoContainer = mongoContainer

	// Start Redis container
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7.0",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	require.NoError(t, err)
	containers.redisContainer = redisContainer

	// Start Typesense container
	typesenseReq := testcontainers.ContainerRequest{
		Image:        "typesense/typesense:0.25.1",
		ExposedPorts: []string{"8108/tcp"},
		Env: map[string]string{
			"TYPESENSE_API_KEY":  "test-key",
			"TYPESENSE_DATA_DIR": "/data",
		},
		Mounts: testcontainers.ContainerMounts{
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: "/tmp/typesense-data",
				},
				Target: "/data",
			},
		},
		WaitingFor: wait.ForListeningPort("8108/tcp"),
	}
	typesenseContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: typesenseReq,
		Started:          true,
	})
	require.NoError(t, err)
	containers.typesenseContainer = typesenseContainer

	// Get MongoDB endpoint for the client
	mongoHost, err := mongoContainer.Host(ctx)
	require.NoError(t, err)
	mongoPort, err := mongoContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	// Create MongoDB client
	mongoURI := "mongodb://" + mongoHost + ":" + mongoPort.Port()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	require.NoError(t, err, "connect to fallback mongo")

	// Create cleanup function
	cleanup := func() {
		_ = client.Disconnect(ctx)
		_ = mongoContainer.Terminate(ctx)
		_ = redisContainer.Terminate(ctx)
		_ = typesenseContainer.Terminate(ctx)
	}

	return client, containers, cleanup
}

// createTypesenseCollection creates a Typesense collection (with a simple schema) using the provided typesenseService.
func createTypesenseCollection(ctx context.Context, t *testing.T, typesenseService externalservices.TypesenseServiceContract, collectionName string) {
	// (In a real scenario, you'd use a "CreateCollection" method on your TypesenseServiceContract. For now, we'll simulate it by "upserting" a dummy document (which implicitly creates the collection) and then "deleting" it.)
	// (Note: In a production (or integration) test, you'd use the real Typesense API to create a collection with a proper schema.)
	dummy := map[string]interface{}{"_id": "dummy", "data": "dummy"}
	err := typesenseService.UpsertDocument(ctx, dummy)
	require.NoError(t, err, "create (upsert dummy) Typesense collection")
	// (Optionally, delete the dummy document so that it doesn't interfere with your test.)
	_ = typesenseService.DeleteDocument(ctx, "dummy")
}

// waitForTypesenseReady waits for Typesense to be ready by first checking if the service is responding
// and then attempting to create a collection with exponential backoff.
// It returns an error if Typesense is not ready after maxAttempts.
func waitForTypesenseReady(ctx context.Context, t *testing.T, typesenseService externalservices.TypesenseServiceContract, maxAttempts int) error {
	// Initial wait before starting checks
	t.Log("Waiting for Typesense container to initialize...")
	time.Sleep(3 * time.Second)

	// First try to check if Typesense is responding
	baseDelay := 1 * time.Second
	maxDelay := 10 * time.Second
	attempt := 0

	// Try to create collection first
	for attempt < maxAttempts {
		// Try to create a collection
		collectionSchema := map[string]interface{}{
			"name": "healthcheck",
			"fields": []map[string]interface{}{
				{"name": "_id", "type": "string"},
				{"name": "data", "type": "string"},
			},
		}

		// Try to create the collection
		err := typesenseService.CreateCollection(ctx, collectionSchema)
		if err == nil || (err != nil && strings.Contains(err.Error(), "already exists")) {
			// Collection created successfully, or already exists, try to write to it
			healthCheck := map[string]interface{}{"_id": "healthcheck", "data": "healthcheck"}
			err = typesenseService.UpsertDocument(ctx, healthCheck)
			if err == nil {
				// Clean up
				_ = typesenseService.DeleteDocument(ctx, "healthcheck")
				_ = typesenseService.DeleteCollection(ctx, "healthcheck")
				t.Log("Typesense health check passed")
				return nil
			}
		}

		// If we get a specific "Not Ready" error, we know Typesense is starting up
		// For other errors, we might want to fail fast
		if err != nil && !strings.Contains(err.Error(), "Not Ready") && !strings.Contains(err.Error(), "Not Found") {
			t.Logf("Unexpected error during health check: %v", err)
			return fmt.Errorf("unexpected error during health check: %w", err)
		}

		// Calculate delay with exponential backoff
		delay := baseDelay * time.Duration(1<<uint(attempt))
		if delay > maxDelay {
			delay = maxDelay
		}

		t.Logf("Typesense not ready (attempt %d/%d), retrying in %v...", attempt+1, maxAttempts, delay)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		attempt++
	}

	return fmt.Errorf("Typesense not ready after %d attempts (waited for health check)", maxAttempts)
}

// TODO: Re-enable stress test once Typesense container initialization is more reliable
// Currently disabled because Typesense container takes too long to initialize
// and fails health checks during high throughput testing.
// This needs to be investigated and fixed to ensure proper stress testing.
/*
func TestEventProcessor_HighThroughput1000Burst(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	// (1) Setup test containers (Mongo, Redis, Typesense) and obtain a fallback mongo client.
	fallbackMongo, containers, cleanup := setupStressTestContainers(t)
	defer cleanup()

	ctx := context.Background()

	// Get container endpoints for services
	mongoHost, err := containers.mongoContainer.Host(ctx)
	require.NoError(t, err)
	mongoPort, err := containers.mongoContainer.MappedPort(ctx, "27017")
	require.NoError(t, err)

	redisHost, err := containers.redisContainer.Host(ctx)
	require.NoError(t, err)
	redisPort, err := containers.redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	typesenseHost, err := containers.typesenseContainer.Host(ctx)
	require.NoError(t, err)
	typesensePort, err := containers.typesenseContainer.MappedPort(ctx, "8108")
	require.NoError(t, err)

	// Create services with container endpoints
	mongoURI := "mongodb://" + mongoHost + ":" + mongoPort.Port()
	mongoService := externalservices.NewMongoService(
		mongoURI,
		"testdb",
		"testcollection",
	)

	fallbackMongoService := externalservices.NewMongoService(
		mongoURI,
		"fallbackdb",
		"fallbackcoll",
	)

	redisService := externalservices.NewRedisService(
		redisHost,
		redisPort.Port(),
		"",
		0,
		"test-queue",
	)

	typesenseService := externalservices.NewTypesenseService(
		"test-key",
		typesenseHost,
		typesensePort.Port(),
		"http",
		"test-collection",
	)

	// Wait for Typesense to be ready with increased retries and better health checking
	err = waitForTypesenseReady(ctx, t, typesenseService, 10) // Increased to 10 attempts
	require.NoError(t, err, "wait for Typesense to be ready")

	// Create the actual collection for the test
	createTypesenseCollection(ctx, t, typesenseService, "test-collection")

	// (2) Create a new event processor with container endpoints
	ep, err := NewEventProcessor(
		typesenseService,
		redisService,
		redisService, // Using same Redis for primary and backup in test
		mongoService,
		fallbackMongoService, // Use fallbackMongoService for fallback
		1000,                 // MaxBufferSize (adjust as needed)
		time.Second,          // ProcessInterval (adjust as needed)
		time.Second,          // FlushInterval (adjust as needed)
	)
	require.NoError(t, err, "new event processor")
	defer ep.Stop()

	// (3) Enqueue a large number of events (e.g. 1000) concurrently.
	events := make([]externalservices.MongoChangeEvent, 1000)
	for i := 0; i < 1000; i++ {
		events[i] = externalservices.MongoChangeEvent{
			OperationType: "insert",
			Document:      bson.M{"_id": "stress-doc-" + string(rune(i)), "data": "stress payload"},
			DocumentID:    "stress-doc-" + string(rune(i)),
			Timestamp:     time.Now().Unix(),
		}
	}

	// (4) Enqueue events concurrently (using goroutines) and wait for all to be enqueued.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // (Use a timeout so the test does not hang.)
	defer cancel()
	for _, ev := range events {
		go func(ev externalservices.MongoChangeEvent) {
			_ = ep.Enqueue(ctx, ev)
		}(ev)
	}

	// (5) Wait a bit (or poll) so that events are processed (and flushed to fallback mongo).
	// (In a real test, you'd poll (or use a "wait" mechanism) until the fallback mongo collection has the expected count.)
	time.Sleep(15 * time.Second)

	// (5.5) Verify that all events (or a "large" number) are stored in Typesense.
	// We'll search for all documents in Typesense and count them.
	searchParams := map[string]interface{}{
		"q":        "*",
		"query_by": "_id",
	}
	typesenseDocs, err := typesenseService.SearchDocuments(ctx, searchParams)
	require.NoError(t, err, "search Typesense documents")
	typesenseCount := int64(len(typesenseDocs))
	expected := int64(900)
	t.Logf("Stress high throughput: Typesense count (actual) = %d, expected (at least) = %d", typesenseCount, expected)
	assert.GreaterOrEqual(t, typesenseCount, expected, "Typesense should contain at least %d events (High Throughput 1000 burst)", expected)

	// (6) Verify that all events (or a "large" number) are stored in fallback mongo.
	// (We'll query the "fallbackcoll" collection in "fallbackdb" and count the documents.)
	coll := fallbackMongo.Database("fallbackdb").Collection("fallbackcoll")
	count, err := coll.CountDocuments(ctx, bson.M{})
	require.NoError(t, err, "count fallback mongo documents")
	// (For a stress test, you might not expect exactly 1000 (due to concurrency, flush intervals, etc.) so we'll assert "at least" a large number (e.g. 900).)
	t.Logf("Stress high throughput: fallback mongo count (actual) = %d, expected (at least) = %d", count, expected)
	assert.GreaterOrEqual(t, count, expected, "fallback mongo should contain at least %d events (High Throughput 1000 burst)", expected)

	// (7) (Optional) Shutdown or cleanup the event processor (if needed).
	// (For example, call ep.FlushAndClose(ctx) or similar.)
	// (In a real test, you'd also tear down containers.)
	ep.Stop()

	// Ensure fallbackMongoService is connected so that all writes are flushed and visible.
	require.NoError(t, fallbackMongoService.Connect(ctx), "connect fallback mongo service")
	defer fallbackMongoService.Disconnect(ctx)

	// Use fallbackMongoService's client to count documents in fallbackdb.fallbackcoll.
	coll = fallbackMongoService.GetCollection()
	count, err = coll.CountDocuments(ctx, bson.M{})
	require.NoError(t, err, "count fallback mongo documents")
	expected = int64(900)
	t.Logf("Stress high throughput: fallback mongo count (actual) = %d, expected (at least) = %d", count, expected)
	assert.GreaterOrEqual(t, count, expected, "fallback mongo should contain at least %d events (High Throughput 1000 burst)", expected)
}
*/
