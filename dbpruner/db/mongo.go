package db

import (
	"context"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Connect establishes a connection to MongoDB using environment variables for configuration.
// This is a standalone version for the dbpruner tool.
func Connect() (*mongo.Client, error) {
	// Get credentials from environment variables if present
	user := os.Getenv("MONGODB_USER")
	password := os.Getenv("MONGODB_PASSWORD")

	// Base URI without authentication
	host := "localhost"
	if os.Getenv("MONGODB_HOST") != "" {
		host = os.Getenv("MONGODB_HOST")
	}

	// Support for direct URI override
	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		clientOptions := options.Client().ApplyURI(uri)
		client, err := mongo.Connect(context.Background(), clientOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
		}

		// Ping the database to verify connection
		err = client.Ping(context.Background(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
		}

		return client, nil
	}

	// Build URI from components
	uri := fmt.Sprintf("mongodb://%s:27017", host)

	// If credentials are provided, modify URI to include authentication
	if user != "" && password != "" {
		uri = fmt.Sprintf("mongodb://%s:%s@%s:27017/?authSource=admin", user, password, host)
	}

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	err = client.Ping(context.Background(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %v", err)
	}

	return client, nil
}
