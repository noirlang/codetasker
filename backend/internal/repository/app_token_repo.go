package repository

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/codetasker/backend/internal/database"
	"github.com/codetasker/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AppTokenRepository handles MongoDB persistence for API AppTokens in the "app_tokens" collection.
type AppTokenRepository struct {
	col *mongo.Collection
}

// NewAppTokenRepository initializes the AppTokenRepository.
func NewAppTokenRepository(db *database.Database) *AppTokenRepository {
	return &AppTokenRepository{
		col: db.Collection("app_tokens"),
	}
}

// HashToken computes a SHA-256 hash string for raw tokens before storing or checking.
func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}

// Create inserts a new AppToken document into MongoDB.
func (r *AppTokenRepository) Create(ctx context.Context, token *domain.AppToken) error {
	if token.ID.IsZero() {
		token.ID = primitive.NewObjectID()
	}
	token.CreatedAt = time.Now()

	_, err := r.col.InsertOne(ctx, token)
	if err != nil {
		return fmt.Errorf("AppTokenRepository.Create: %w", err)
	}
	return nil
}

// FindByUserID returns all active AppTokens for a given user.
func (r *AppTokenRepository) FindByUserID(ctx context.Context, userID primitive.ObjectID) ([]domain.AppToken, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("AppTokenRepository.FindByUserID: %w", err)
	}
	defer cursor.Close(ctx)

	var tokens []domain.AppToken
	if err := cursor.All(ctx, &tokens); err != nil {
		return nil, fmt.Errorf("AppTokenRepository.FindByUserID.All: %w", err)
	}
	if tokens == nil {
		tokens = []domain.AppToken{}
	}
	return tokens, nil
}

// FindByTokenHash looks up an AppToken by its SHA-256 token hash.
func (r *AppTokenRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*domain.AppToken, error) {
	var token domain.AppToken
	err := r.col.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&token)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("AppTokenRepository.FindByTokenHash: %w", err)
	}
	return &token, nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token.
func (r *AppTokenRepository) UpdateLastUsed(ctx context.Context, tokenID primitive.ObjectID) error {
	now := time.Now()
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": tokenID}, bson.M{"$set": bson.M{"last_used_at": now}})
	return err
}

// Delete removes an AppToken belonging to a specific user.
func (r *AppTokenRepository) Delete(ctx context.Context, tokenID, userID primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": tokenID, "user_id": userID})
	if err != nil {
		return fmt.Errorf("AppTokenRepository.Delete: %w", err)
	}
	return nil
}
