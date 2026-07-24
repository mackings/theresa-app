package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

// Connect dials MongoDB, verifies connectivity with a ping, and returns the
// database handle for the given database name.
func Connect(ctx context.Context, uri, dbName string) (*mongo.Client, *mongo.Database, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, nil, fmt.Errorf("mongo connect: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, nil, fmt.Errorf("mongo ping: %w", err)
	}

	return client, client.Database(dbName), nil
}

// EnsureIndexes creates all indexes the app depends on. Safe to call on
// every startup — index creation is idempotent.
func EnsureIndexes(ctx context.Context, database *mongo.Database) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := database.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create users email index: %w", err)
	}

	_, err = database.Collection("documents").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "owner_id", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("create documents owner_id index: %w", err)
	}

	_, err = database.Collection("sessions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "owner_id", Value: 1}, {Key: "updated_at", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("create sessions owner_id/updated_at index: %w", err)
	}

	return nil
}
