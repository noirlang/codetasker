// Package repository provides data access objects for CodeTasker's MongoDB
// collections. activity_repo.go implements persistence for the activity_logs
// collection, recording a human-readable audit trail of repository events.
package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/codetasker/backend/internal/database"
	"github.com/codetasker/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ActivityRepository wraps the MongoDB activity_logs collection.
type ActivityRepository struct {
	col *mongo.Collection
}

// NewActivityRepository creates an ActivityRepository backed by the "activity_logs" collection.
func NewActivityRepository(db *database.Database) *ActivityRepository {
	return &ActivityRepository{col: db.Collection("activity_logs")}
}

// Log inserts a new activity log entry, auto-assigning ID and CreatedAt.
func (r *ActivityRepository) Log(ctx context.Context, entry *domain.ActivityLog) error {
	entry.ID = primitive.NewObjectID()
	entry.CreatedAt = time.Now().UTC()
	_, err := r.col.InsertOne(ctx, entry)
	if err != nil {
		return fmt.Errorf("ActivityRepository.Log: %w", err)
	}
	return nil
}

// FindByRepo returns the most recent 100 activity logs for a repository, ordered newest-first.
// Returns an empty (non-nil) slice if none exist.
func (r *ActivityRepository) FindByRepo(ctx context.Context, repoID int64) ([]domain.ActivityLog, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(100)
	cursor, err := r.col.Find(ctx, bson.M{"repo_id": repoID}, opts)
	if err != nil {
		return nil, fmt.Errorf("ActivityRepository.FindByRepo: %w", err)
	}
	defer cursor.Close(ctx)
	var logs []domain.ActivityLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, fmt.Errorf("ActivityRepository.FindByRepo decode: %w", err)
	}
	if logs == nil {
		logs = []domain.ActivityLog{}
	}
	return logs, nil
}
