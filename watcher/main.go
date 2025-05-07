package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/typesense"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Config struct {
	MongoURI            string
	MongoDatabase       string
	MongoCollection     string
	TypesenseAPIKey     string
	TypesenseHost       string
	TypesensePort       string
	TypesenseProtocol   string
	TypesenseCollection string
}

func loadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		return nil, fmt.Errorf("error loading .env file: %v", err)
	}

	return &Config{
		MongoURI:            os.Getenv("MONGODB_URI"),
		MongoDatabase:       os.Getenv("MONGODB_DATABASE"),
		MongoCollection:     os.Getenv("MONGODB_COLLECTION"),
		TypesenseAPIKey:     os.Getenv("TYPESENSE_API_KEY"),
		TypesenseHost:       os.Getenv("TYPESENSE_HOST"),
		TypesensePort:       os.Getenv("TYPESENSE_PORT"),
		TypesenseProtocol:   os.Getenv("TYPESENSE_PROTOCOL"),
		TypesenseCollection: os.Getenv("TYPESENSE_COLLECTION_NAME"),
	}, nil
}

func connectToMongoDB(uri string) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return client, nil
}

func connectToTypesense(config *Config) (*typesense.Client, error) {
	client := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("%s://%s:%s", config.TypesenseProtocol, config.TypesenseHost, config.TypesensePort)),
		typesense.WithAPIKey(config.TypesenseAPIKey),
		typesense.WithConnectionTimeout(10*time.Second),
	)
	return client, nil
}

func watchMongoChanges(client *mongo.Client, database, collection string, typesenseClient *typesense.Client, typesenseCollection string) error {
	ctx := context.Background()
	coll := client.Database(database).Collection(collection)

	// Create a change stream
	changeStream, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		return fmt.Errorf("failed to create change stream: %v", err)
	}
	defer changeStream.Close(ctx)

	fmt.Printf("Watching for changes in %s.%s...\n", database, collection)

	for changeStream.Next(ctx) {
		var changeDoc bson.M
		if err := changeStream.Decode(&changeDoc); err != nil {
			log.Printf("Error decoding change document: %v", err)
			continue
		}

		operationType := changeDoc["operationType"]
		fullDocument := changeDoc["fullDocument"]

		switch operationType {
		case "insert", "update":
			// Convert the document to a map for Typesense
			doc := make(map[string]interface{})
			bsonBytes, err := bson.Marshal(fullDocument)
			if err != nil {
				log.Printf("Error marshaling document: %v", err)
				continue
			}
			if err := bson.Unmarshal(bsonBytes, &doc); err != nil {
				log.Printf("Error unmarshaling document: %v", err)
				continue
			}

			// Upsert to Typesense
			_, err = typesenseClient.Collection(typesenseCollection).Documents().Upsert(ctx, doc)
			if err != nil {
				log.Printf("Error upserting to Typesense: %v", err)
			} else {
				log.Printf("Successfully synced document to Typesense: %v", doc["_id"])
			}

		case "delete":
			// Get the document ID from the change document
			docKey := changeDoc["documentKey"].(bson.M)["_id"]
			docKeyStr := fmt.Sprintf("%v", docKey)

			// Delete from Typesense
			_, err := typesenseClient.Collection(typesenseCollection).Document(docKeyStr).Delete(ctx)
			if err != nil {
				log.Printf("Error deleting from Typesense: %v", err)
			} else {
				log.Printf("Successfully deleted document from Typesense: %v", docKeyStr)
			}
		}
	}

	if err := changeStream.Err(); err != nil {
		return fmt.Errorf("change stream error: %v", err)
	}

	return nil
}

func main() {
	// Load configuration
	config, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to MongoDB
	mongoClient, err := connectToMongoDB(config.MongoURI)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	// Connect to Typesense
	typesenseClient, err := connectToTypesense(config)
	if err != nil {
		log.Fatalf("Failed to connect to Typesense: %v", err)
	}

	// Start watching for changes
	err = watchMongoChanges(mongoClient, config.MongoDatabase, config.MongoCollection, typesenseClient, config.TypesenseCollection)
	if err != nil {
		log.Fatalf("Error watching MongoDB changes: %v", err)
	}
}
