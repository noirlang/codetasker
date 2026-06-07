// Package repository implements the database storage layer for CodeTasker.
// collaborator_repo.go handles database operations for repository collaborators.
package repository

import (
	"context"
	"time"

	"github.com/codetasker/backend/internal/database"
	"github.com/codetasker/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// CollaboratorRepository manages the storage lifecycle of repository collaborators.
type CollaboratorRepository struct {
	collection *mongo.Collection
}

// NewCollaboratorRepository constructs a CollaboratorRepository.
func NewCollaboratorRepository(db *database.Database) *CollaboratorRepository {
	return &CollaboratorRepository{
		collection: db.Collection("collaborators"),
	}
}

// Create inserts a new collaborator relationship into the database.
func (r *CollaboratorRepository) Create(ctx context.Context, c *domain.Collaborator) error {
	if c.ID.IsZero() {
		c.ID = primitive.NewObjectID()
	}
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now()
	}
	_, err := r.collection.InsertOne(ctx, c)
	return err
}

// FindByRepoID retrieves all collaborators for a given repository ID.
func (r *CollaboratorRepository) FindByRepoID(ctx context.Context, repoID int64) ([]domain.Collaborator, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"repo_id": repoID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.Collaborator
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}

// FindByUserAndRepo checks if a user has a collaborator record for a repository.
func (r *CollaboratorRepository) FindByUserAndRepo(ctx context.Context, userID primitive.ObjectID, repoID int64) (*domain.Collaborator, error) {
	var c domain.Collaborator
	err := r.collection.FindOne(ctx, bson.M{"user_id": userID, "repo_id": repoID}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &c, nil
}

// UpdateRole updates a collaborator's access level role.
func (r *CollaboratorRepository) UpdateRole(ctx context.Context, id primitive.ObjectID, role domain.RepoRole) error {
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"role": role}})
	return err
}

// Delete removes a collaborator record from the database.
func (r *CollaboratorRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}

// FindByUserID retrieves all collaborator records for a given user.
func (r *CollaboratorRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]domain.Collaborator, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var results []domain.Collaborator
	if err := cursor.All(ctx, &results); err != nil {
		return nil, err
	}
	return results, nil
}
