// Package repository provides data access objects for CodeTasker's MongoDB
// collections. The synced_repo.go file implements all read/write operations for
// the synced_repos collection, storing which repositories have active webhook sync.
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
)

// SyncedRepository wraps the MongoDB synced_repos collection.
type SyncedRepository struct {
	col *mongo.Collection
}

// NewSyncedRepository constructs a SyncedRepository backed by the "synced_repos" collection.
func NewSyncedRepository(db *database.Database) *SyncedRepository {
	return &SyncedRepository{
		col: db.Collection("synced_repos"),
	}
}

// Create inserts a new SyncedRepo document into the database.
func (r *SyncedRepository) Create(ctx context.Context, repo *domain.SyncedRepo) error {
	repo.ID = primitive.NewObjectID()
	repo.CreatedAt = time.Now().UTC()

	_, err := r.col.InsertOne(ctx, repo)
	if err != nil {
		return fmt.Errorf("SyncedRepository.Create: %w", err)
	}
	return nil
}

// FindByUserID retrieves all synced repositories activated by a specific user.
func (r *SyncedRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]domain.SyncedRepo, error) {
	var results []domain.SyncedRepo

	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("SyncedRepository.FindByUserID: %w", err)
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("SyncedRepository.FindByUserID decode: %w", err)
	}

	if results == nil {
		results = []domain.SyncedRepo{}
	}
	return results, nil
}

// FindByRepoID looks up a synced repo record by user ID and GitHub repository ID.
func (r *SyncedRepository) FindByRepoID(ctx context.Context, userID primitive.ObjectID, repoID int64) (*domain.SyncedRepo, error) {
	var repo domain.SyncedRepo

	filter := bson.M{
		"user_id": userID,
		"repo_id": repoID,
	}

	err := r.col.FindOne(ctx, filter).Decode(&repo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("SyncedRepository.FindByRepoID: %w", err)
	}

	return &repo, nil
}

// FindByRepoIDOnly looks up a synced repo record by GitHub repository ID.
func (r *SyncedRepository) FindByRepoIDOnly(ctx context.Context, repoID int64) (*domain.SyncedRepo, error) {
	var repo domain.SyncedRepo
	err := r.col.FindOne(ctx, bson.M{"repo_id": repoID}).Decode(&repo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("SyncedRepository.FindByRepoIDOnly: %w", err)
	}
	return &repo, nil
}

// Delete removes a synced repo record, disabling sync trackers.
func (r *SyncedRepository) Delete(ctx context.Context, userID primitive.ObjectID, repoID int64) error {
	filter := bson.M{
		"user_id": userID,
		"repo_id": repoID,
	}

	_, err := r.col.DeleteOne(ctx, filter)
	if err != nil {
		return fmt.Errorf("SyncedRepository.Delete: %w", err)
	}
	return nil
}

// FindByRepoName looks up a synced repo record by its full name (e.g. "owner/repo").
func (r *SyncedRepository) FindByRepoName(ctx context.Context, repoName string) (*domain.SyncedRepo, error) {
	var repo domain.SyncedRepo
	err := r.col.FindOne(ctx, bson.M{"repo_name": repoName}).Decode(&repo)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("SyncedRepository.FindByRepoName: %w", err)
	}
	return &repo, nil
}
