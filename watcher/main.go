package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	externalservices "watcher/external-services"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	// Configurable zerolog log level and time format
	if lvlStr := os.Getenv("LOG_LEVEL"); lvlStr != "" {
		if lvl, err := zerolog.ParseLevel(lvlStr); err == nil {
			zerolog.SetGlobalLevel(lvl)
		} else {
			log.Warn().Str("LOG_LEVEL", lvlStr).Msg("Invalid log level, using default")
		}
	}
	if tf := os.Getenv("LOG_TIME_FORMAT"); tf != "" {
		if tf == "unix" {
			zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
		} else {
			zerolog.TimeFieldFormat = tf
		}
	}

	// Start Prometheus metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		metricsPort := os.Getenv("METRICS_PORT")
		if metricsPort == "" {
			metricsPort = "2112" // Default metrics port
		}
		log.Info().Str("port", metricsPort).Msg("Starting Prometheus metrics server")
		if err := http.ListenAndServe(":"+metricsPort, nil); err != nil {
			log.Error().Err(err).Msg("Failed to start metrics server")
		}
	}()

	// Load configuration
	config, err := LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Create Redis services
	primaryRedis := externalservices.NewRedisService(
		config.PrimaryRedis.Host,
		config.PrimaryRedis.Port,
		config.PrimaryRedis.Password,
		config.PrimaryRedis.DB,
		config.PrimaryRedis.QueueKey,
	)

	backupRedis := externalservices.NewRedisService(
		config.SecondaryRedis.Host,
		config.SecondaryRedis.Port,
		config.SecondaryRedis.Password,
		config.SecondaryRedis.DB,
		config.SecondaryRedis.QueueKey,
	)

	// Create MongoDB services
	watchingMongoDBService := externalservices.NewMongoService(
		config.PrimaryMongoURI,
		config.PrimaryMongoDatabase,
		config.PrimaryMongoCollection,
	)

	fallbackMongoService := externalservices.NewMongoService(
		config.FallbackMongoURI,
		config.FallbackMongoDatabase,
		config.FallbackMongoCollection,
	)

	// Create Typesense service
	typesenseService := externalservices.NewTypesenseService(
		config.Typesense.APIKey,
		config.Typesense.Host,
		config.Typesense.Port,
		config.Typesense.Protocol,
		config.Typesense.CollectionName,
	)

	eventProcessor, err := NewEventProcessor(
		typesenseService,
		primaryRedis,
		backupRedis,
		watchingMongoDBService,
		fallbackMongoService,
		config.MaxBufferSize,
		time.Duration(config.ProcessInterval)*time.Second,
		time.Duration(config.ProcessInterval)*time.Second*2,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize event processor")
	}

	// Connect to MongoDB services
	if err := watchingMongoDBService.Connect(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to watching MongoDB")
	}
	defer watchingMongoDBService.Disconnect(context.Background())

	if err := fallbackMongoService.Connect(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to fallback MongoDB")
	}
	defer fallbackMongoService.Disconnect(context.Background())

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Info().Msg("Received shutdown signal")
		cancel()
	}()

	// Start watching MongoDB changes
	log.Info().Msg("Starting MongoDB change stream watcher...")
	opts := options.ChangeStream().SetFullDocument(options.UpdateLookup)
	stream, err := watchingMongoDBService.Watch(ctx, nil, opts)
	if err != nil {
		log.Fatal().Err(err).Msg("Error creating change stream")
	}
	defer stream.Close(ctx)

	// Get the processing state channel
	processingState := eventProcessor.GetProcessBusyState()
	isNearLimitState := eventProcessor.GetNearLimitState()
	isProcessing := false
	isNearLimit := false
	for {
		select {
		case <-ctx.Done():
			return
		case processing := <-processingState:
			isProcessing = processing
			if !isProcessing {
				log.Info().Msg("Processing finished, resuming MongoDB event reading")
			} else {
				log.Info().Msg("Processing started, pausing MongoDB event reading")
			}
		case nearLimitState := <-isNearLimitState:
			isNearLimit = nearLimitState
			if !isNearLimit {
				log.Info().Msg("Buffer is not near limit, resuming MongoDB event reading")
			} else {
				log.Info().Msg("Buffer is near limit, pausing MongoDB event reading")
			}
		default:
			// Only read from MongoDB if we're not processing
			if isProcessing || isNearLimit {
				// Sleep briefly to prevent tight loop
				time.Sleep(100 * time.Millisecond)
				continue
			}

			if !stream.Next(ctx) {
				if err := stream.Err(); err != nil {
					log.Error().Err(err).Msg("Change stream error")
				}
				continue
			}

			var changeDoc struct {
				OperationType string                 `bson:"operationType"`
				FullDocument  map[string]interface{} `bson:"fullDocument"`
				DocumentKey   map[string]interface{} `bson:"documentKey"`
				ClusterTime   time.Time              `bson:"clusterTime"`
			}

			if err := stream.Decode(&changeDoc); err != nil {
				log.Error().Err(err).Msg("Failed to decode change document")
				continue
			}

			event := externalservices.MongoChangeEvent{
				OperationType: changeDoc.OperationType,
				Document:      changeDoc.FullDocument,
				DocumentID:    changeDoc.DocumentKey["_id"].(string),
				Timestamp:     changeDoc.ClusterTime.Unix(),
			}

			if err := eventProcessor.Enqueue(ctx, event); err != nil {
				log.Error().Err(err).Msg("Failed to enqueue event")
			}
		}
	}
}
