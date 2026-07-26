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

	_, err = database.Collection("credit_transactions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		return fmt.Errorf("create credit_transactions user_id/created_at index: %w", err)
	}

	// Unique but sparse: only "purchase" rows ever set flw_tx_ref, and this
	// is exactly what makes crediting idempotent against Flutterwave's
	// webhook retries - a second insert attempt for the same tx_ref fails
	// with a duplicate-key error instead of double-crediting the account.
	_, err = database.Collection("credit_transactions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "flw_tx_ref", Value: 1}},
		Options: options.Index().SetUnique(true).SetSparse(true),
	})
	if err != nil {
		return fmt.Errorf("create credit_transactions flw_tx_ref index: %w", err)
	}

	return nil
}

// BackfillFreeTrial gives every account created before the credits/billing
// feature existed its 5 minutes of free voice, same as a brand new signup
// gets - without this, an existing user's free_trial_seconds_remaining
// field simply doesn't exist in their document yet, which decodes as a Go
// zero value (0) and would make them look like they'd already exhausted a
// trial they never had, locking them out of voice entirely. Matches
// free_trial_seconds against "doesn't exist yet" specifically, so it's safe
// to run on every startup - real trial consumption (however it reaches 0)
// is never touched a second time.
func BackfillFreeTrial(ctx context.Context, database *mongo.Database, freeTrialSeconds int) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := database.Collection("users").UpdateMany(ctx,
		bson.M{"free_trial_seconds_remaining": bson.M{"$exists": false}},
		bson.M{"$set": bson.M{"free_trial_seconds_remaining": freeTrialSeconds}},
	)
	if err != nil {
		return fmt.Errorf("backfill free trial seconds: %w", err)
	}
	return nil
}
