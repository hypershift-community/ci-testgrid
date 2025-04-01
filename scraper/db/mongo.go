package db

import (
	"context"
	"fmt"
	"os"

	"github.com/hypershift-community/ci-testgrid/scraper/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Connect() (*mongo.Client, error) {
	// Get credentials from environment variables if present
	user := os.Getenv("MONGODB_USER")
	password := os.Getenv("MONGODB_PASSWORD")

	// Base URI without authentication
	host := "localhost"
	if os.Getenv("MONGODB_HOST") != "" {
		host = os.Getenv("MONGODB_HOST")
	}

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

func JobExists(ctx context.Context, collection *mongo.Collection, jobID string) (bool, error) {
	var result types.Job
	err := collection.FindOne(ctx, bson.M{"_id": jobID}).Decode(&result)
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	return err == nil, err
}

func InsertJob(ctx context.Context, collection *mongo.Collection, job *types.Job) error {
	_, err := collection.InsertOne(ctx, job)
	return err
}
