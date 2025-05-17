package main

// make sure to run this before running the test:
// sudo mkdir -p /private/tmp/typesense-data
// sudo chmod 777 /private/tmp/typesense-data

import (
	"context"
	"testing"
	"time"

	externalservices "watcher/external-services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type testContainers struct {
	mongoContainer     testcontainers.Container
	redisContainer     testcontainers.Container
	typesenseContainer testcontainers.Container
}

func setupTestContainers(t *testing.T) *testContainers {
	ctx := context.Background()

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

	return &testContainers{
		mongoContainer:     mongoContainer,
		redisContainer:     redisContainer,
		typesenseContainer: typesenseContainer,
	}
}

func (tc *testContainers) cleanup(t *testing.T) {
	ctx := context.Background()
	if tc.mongoContainer != nil {
		require.NoError(t, tc.mongoContainer.Terminate(ctx))
	}
	if tc.redisContainer != nil {
		require.NoError(t, tc.redisContainer.Terminate(ctx))
	}
	if tc.typesenseContainer != nil {
		require.NoError(t, tc.typesenseContainer.Terminate(ctx))
	}
}

func TestEventProcessorIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	containers := setupTestContainers(t)
	defer containers.cleanup(t)

	ctx := context.Background()

	// Get container endpoints
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

	// Create event processor
	processor, err := NewEventProcessor(
		typesenseService,
		redisService,
		redisService, // Using same Redis for primary and backup in test
		mongoService,
		mongoService, // Using same MongoDB for watching and fallback in test
		1000,
		time.Second,
		time.Second*2,
	)
	require.NoError(t, err)
	defer processor.Stop()

	// Connect to MongoDB
	require.NoError(t, mongoService.Connect(ctx))
	defer mongoService.Disconnect(ctx)

	// Create test collection in Typesense
	// Note: In a real test, you'd want to create the collection schema here
	// For this example, we'll assume the collection exists

	// Test event processing
	t.Run("Process Insert Event", func(t *testing.T) {
		event := externalservices.MongoChangeEvent{
			OperationType: "insert",
			DocumentID:    "test-doc-1",
			Document: map[string]interface{}{
				"id":    "test-doc-1",
				"title": "Test Document",
			},
			Timestamp: time.Now().Unix(),
		}

		err := processor.Enqueue(ctx, event)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(time.Second * 2)

		// Verify metrics
		// Note: In a real test, you'd want to verify the metrics
		// For example, check that event_processor_events_processed_total{operation_type="insert",status="success"} increased
	})

	t.Run("Process Update Event", func(t *testing.T) {
		event := externalservices.MongoChangeEvent{
			OperationType: "update",
			DocumentID:    "test-doc-1",
			Document: map[string]interface{}{
				"id":    "test-doc-1",
				"title": "Updated Test Document",
			},
			Timestamp: time.Now().Unix(),
		}

		err := processor.Enqueue(ctx, event)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(time.Second * 2)
	})

	t.Run("Process Delete Event", func(t *testing.T) {
		event := externalservices.MongoChangeEvent{
			OperationType: "delete",
			DocumentID:    "test-doc-1",
			Timestamp:     time.Now().Unix(),
		}

		err := processor.Enqueue(ctx, event)
		require.NoError(t, err)

		// Wait for processing
		time.Sleep(time.Second * 2)
	})

	t.Run("Process Batch of Events", func(t *testing.T) {
		events := []externalservices.MongoChangeEvent{
			{
				OperationType: "insert",
				DocumentID:    "batch-doc-1",
				Document: map[string]interface{}{
					"id":    "batch-doc-1",
					"title": "Batch Document 1",
				},
				Timestamp: time.Now().Unix(),
			},
			{
				OperationType: "insert",
				DocumentID:    "batch-doc-2",
				Document: map[string]interface{}{
					"id":    "batch-doc-2",
					"title": "Batch Document 2",
				},
				Timestamp: time.Now().Unix(),
			},
		}

		for _, event := range events {
			err := processor.Enqueue(ctx, event)
			require.NoError(t, err)
		}

		// Wait for processing
		time.Sleep(time.Second * 2)
	})

	// Test error handling
	t.Run("Process Invalid Event", func(t *testing.T) {
		event := externalservices.MongoChangeEvent{
			OperationType: "insert",
			DocumentID:    "invalid-doc",
			Document:      nil, // This should trigger an error
			Timestamp:     time.Now().Unix(),
		}

		err := processor.Enqueue(ctx, event)
		require.NoError(t, err) // Enqueue should succeed

		// Wait for processing
		time.Sleep(time.Second * 2)

		// Verify metrics for error
		// Note: In a real test, you'd want to verify the error metrics
	})

	// Test graceful shutdown
	t.Run("Graceful Shutdown", func(t *testing.T) {
		// Enqueue some events
		event := externalservices.MongoChangeEvent{
			OperationType: "insert",
			DocumentID:    "shutdown-doc",
			Document: map[string]interface{}{
				"id":    "shutdown-doc",
				"title": "Shutdown Test Document",
			},
			Timestamp: time.Now().Unix(),
		}

		err := processor.Enqueue(ctx, event)
		require.NoError(t, err)

		// Start shutdown
		processor.FlushAndClose()

		// Verify that no new events can be enqueued
		err = processor.Enqueue(ctx, event)
		assert.Error(t, err) // Should error as the processor is closed
	})
}
