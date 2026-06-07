// Package database provides MongoDB connection management for CodeTasker.
// It wraps the official mongo-driver, handles connection pooling, server
// verification via Ping, and index creation to ensure the database is ready
// for production traffic before the HTTP server starts accepting requests.
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Database wraps the MongoDB client and a handle to the application database.
// All repository constructors accept *Database so they can call Collection()
// without needing the raw client.
type Database struct {
	client *mongo.Client
	db     *mongo.Database
}

// Connect dials MongoDB using the provided URI, verifies connectivity with a
// Ping, creates all required indexes, and returns a ready-to-use *Database.
//
// It uses a 10-second context for the initial connection and ping so that a
// misconfigured Mongo URI fails fast at startup rather than silently hanging.
func Connect(uri, dbName string) (*Database, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build client options; the driver handles connection pooling internally.
	clientOpts := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo.Connect: %w", err)
	}

	// Ping verifies that the mongod process is reachable and the URI is correct.
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	d := &Database{
		client: client,
		db:     client.Database(dbName),
	}

	// Create all application indexes before returning.
	if err := d.EnsureIndexes(context.Background()); err != nil {
		return nil, fmt.Errorf("EnsureIndexes: %w", err)
	}

	return d, nil
}

// Collection returns a handle to the named MongoDB collection within the
// application database. Repositories call this to obtain their collection.
func (d *Database) Collection(name string) *mongo.Collection {
	return d.db.Collection(name)
}

// Client exposes the underlying mongo.Client for operations that require it
// (e.g. transactions).
func (d *Database) Client() *mongo.Client {
	return d.client
}

// Disconnect gracefully closes the MongoDB client. Call this during server
// shutdown to release all pooled connections.
func (d *Database) Disconnect(ctx context.Context) error {
	return d.client.Disconnect(ctx)
}

// EnsureIndexes creates all MongoDB indexes required by the application.
// The function is idempotent: calling it multiple times on a collection that
// already has the indexes is a no-op (MongoDB deduplicates by index definition).
//
// Indexes created:
//   - users.github_id  — unique index enabling O(1) lookup during OAuth.
//   - tasks.{repo_id, file_path} — compound index for per-repo file queries.
//   - tasks.status — single-field index for filtering by task status.
func (d *Database) EnsureIndexes(ctx context.Context) error {
	// ── users collection ────────────────────────────────────────────────────
	usersCol := d.db.Collection("users")

	_, err := usersCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "github_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create users.github_id index: %w", err)
	}

	// ── tasks collection ────────────────────────────────────────────────────
	tasksCol := d.db.Collection("tasks")

	// Compound index on repo_id + file_path improves per-repo file queries.
	_, err = tasksCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "repo_id", Value: 1},
			{Key: "file_path", Value: 1},
		},
		Options: options.Index().SetBackground(true),
	})
	if err != nil {
		return fmt.Errorf("create tasks.{repo_id,file_path} index: %w", err)
	}

	// Single-field index on status for dashboard status-filter queries.
	_, err = tasksCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "status", Value: 1}},
		Options: options.Index().SetBackground(true),
	})
	if err != nil {
		return fmt.Errorf("create tasks.status index: %w", err)
	}

	// ── collaborators collection ────────────────────────────────────────────
	collabsCol := d.db.Collection("collaborators")
	_, err = collabsCol.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "repo_id", Value: 1},
			{Key: "user_id", Value: 1},
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("create collaborators.{repo_id,user_id} index: %w", err)
	}

	return nil
}
