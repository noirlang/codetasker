// Package repository provides data access objects for CodeTasker's MongoDB
// collections. comment_repo.go implements CRUD operations for the comments
// collection, allowing users to annotate tasks with threaded remarks.
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

// CommentRepository wraps the MongoDB comments collection.
type CommentRepository struct {
	col *mongo.Collection
}

// NewCommentRepository creates a CommentRepository backed by the "comments" collection.
func NewCommentRepository(db *database.Database) *CommentRepository {
	return &CommentRepository{col: db.Collection("comments")}
}

// Create inserts a new comment document. It auto-generates the ID and timestamps.
func (r *CommentRepository) Create(ctx context.Context, c *domain.Comment) error {
	now := time.Now().UTC()
	c.ID = primitive.NewObjectID()
	c.CreatedAt = now
	c.UpdatedAt = now
	_, err := r.col.InsertOne(ctx, c)
	if err != nil {
		return fmt.Errorf("CommentRepository.Create: %w", err)
	}
	return nil
}

// FindByTask returns all comments for a task, ordered by creation time ascending.
// Returns an empty (non-nil) slice if no comments exist.
func (r *CommentRepository) FindByTask(ctx context.Context, taskID primitive.ObjectID) ([]domain.Comment, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := r.col.Find(ctx, bson.M{"task_id": taskID}, opts)
	if err != nil {
		return nil, fmt.Errorf("CommentRepository.FindByTask: %w", err)
	}
	defer cursor.Close(ctx)
	var comments []domain.Comment
	if err := cursor.All(ctx, &comments); err != nil {
		return nil, fmt.Errorf("CommentRepository.FindByTask decode: %w", err)
	}
	if comments == nil {
		comments = []domain.Comment{}
	}
	return comments, nil
}

// Delete removes a comment by ID. Returns error if the document is not found.
func (r *CommentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("CommentRepository.Delete: %w", err)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("comment %s not found", id.Hex())
	}
	return nil
}

// FindByID returns a single comment by its ObjectID.
// Returns (nil, nil) if no document with that ID exists.
func (r *CommentRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Comment, error) {
	var c domain.Comment
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&c)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("CommentRepository.FindByID: %w", err)
	}
	return &c, nil
}
