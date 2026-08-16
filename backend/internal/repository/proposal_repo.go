// Package repository implements the database storage layer for CodeTasker.
// proposal_repo.go handles database operations for task proposals & discussions.
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

// ProposalRepository manages the storage lifecycle of task proposals.
type ProposalRepository struct {
	collection *mongo.Collection
}

// NewProposalRepository constructs a ProposalRepository.
func NewProposalRepository(db *database.Database) *ProposalRepository {
	return &ProposalRepository{
		collection: db.Collection("proposals"),
	}
}

// Create inserts a new proposal document into MongoDB.
func (r *ProposalRepository) Create(ctx context.Context, p *domain.TaskProposal) error {
	if p.ID.IsZero() {
		p.ID = primitive.NewObjectID()
	}
	now := time.Now()
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now
	}
	p.UpdatedAt = now
	if p.Status == "" {
		p.Status = domain.ProposalStatusPending
	}
	if p.VotedBy == nil {
		p.VotedBy = []string{}
	}
	_, err := r.collection.InsertOne(ctx, p)
	return err
}

// FindByTaskID retrieves all proposals for a given task ID, sorted chronologically.
func (r *ProposalRepository) FindByTaskID(ctx context.Context, taskID primitive.ObjectID) ([]domain.TaskProposal, error) {
	cursor, err := r.collection.Find(ctx, bson.M{"task_id": taskID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var proposals []domain.TaskProposal
	if err := cursor.All(ctx, &proposals); err != nil {
		return nil, err
	}
	if proposals == nil {
		proposals = []domain.TaskProposal{}
	}
	return proposals, nil
}

// FindByID retrieves a single proposal by its ObjectID.
func (r *ProposalRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.TaskProposal, error) {
	var p domain.TaskProposal
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&p)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return &p, nil
}

// UpdateStatus updates the approval/rejection status of a proposal.
func (r *ProposalRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status domain.ProposalStatus, voterUsername string) error {
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now(),
		},
	}
	if voterUsername != "" {
		update["$addToSet"] = bson.M{
			"voted_by": voterUsername,
		}
	}
	_, err := r.collection.UpdateOne(ctx, bson.M{"_id": id}, update)
	return err
}

// Delete removes a proposal document by ID.
func (r *ProposalRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	return err
}
