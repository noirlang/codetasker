// Package repository provides data access objects for CodeTasker's MongoDB
// collections. The user_repo.go file implements all read/write operations for
// the users collection, insulating the service layer from direct driver usage.
package repository

import (
	"context"
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

// UserRepository wraps the MongoDB users collection and exposes typed methods
// for all user-related persistence operations.
type UserRepository struct {
	col *mongo.Collection
}

// NewUserRepository creates a UserRepository backed by the "users" collection
// in the provided database. The collection handle is obtained once and reused
// across calls; the mongo-driver manages connection pooling internally.
func NewUserRepository(db *database.Database) *UserRepository {
	return &UserRepository{
		col: db.Collection("users"),
	}
}

// FindByGithubID looks up a user by their immutable GitHub numeric user ID.
// It returns (nil, nil) when the user does not exist yet — this is the
// expected behaviour on first OAuth login and callers must check for nil.
// Any unexpected driver error is wrapped and returned as a non-nil error.
func (r *UserRepository) FindByGithubID(ctx context.Context, githubID int64) (*domain.User, error) {
	var user domain.User

	filter := bson.M{"github_id": githubID}
	err := r.col.FindOne(ctx, filter).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// User has never logged in before — return nil without an error
			// so callers can decide to create a new user document.
			return nil, nil
		}
		return nil, fmt.Errorf("FindByGithubID(%d): %w", githubID, err)
	}

	return &user, nil
}

// Upsert creates or updates a user document atomically using MongoDB's upsert
// semantics. The filter is the immutable github_id field so that a renamed
// GitHub account still maps to the same document.
//
// The operation uses:
//   - $set to update mutable fields (username, avatar_url, access_token, updated_at)
//     on every call, including the first insert.
//   - $setOnInsert to record created_at exactly once when the document is new.
//
// This approach avoids overwriting created_at on subsequent logins while still
// refreshing the encrypted access token after every OAuth exchange.
func (r *UserRepository) Upsert(ctx context.Context, user *domain.User) error {
	now := time.Now().UTC()

	filter := bson.M{"github_id": user.GithubID}

	update := bson.M{
		"$set": bson.M{
			"username":     user.Username,
			"avatar_url":   user.AvatarURL,
			"access_token": user.AccessToken,
			"updated_at":   now,
		},
		// $setOnInsert fields are only written when the document is being created.
		// On subsequent updates this operator is a no-op.
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("Upsert user (github_id=%d): %w", user.GithubID, err)
	}

	// Refresh the in-memory struct so callers see the persisted timestamps.
	user.UpdatedAt = now
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}

	return nil
}

// FindByObjectID retrieves a user document by its MongoDB ObjectID.
// This is used by GithubService to look up a user when only the JWT sub claim
// (ObjectID hex) is available, rather than the github_id.
// Returns (nil, nil) if no document with that ID exists.
func (r *UserRepository) FindByObjectID(ctx context.Context, id primitive.ObjectID) (*domain.User, error) {
	var user domain.User

	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindByObjectID(%s): %w", id.Hex(), err)
	}

	return &user, nil
}

// FindByUsername looks up a CodeTasker user by their GitHub login handle.
// This is used by the webhook processing pipeline to find the repository
// owner's account — and therefore their OAuth token — using only the owner
// name string provided in the GitHub push event payload.
// Returns (nil, nil) if no user with that username exists.
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	var user domain.User

	opts := options.FindOne().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2,
	})

	err := r.col.FindOne(ctx, bson.M{"username": username}, opts).Decode(&user)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindByUsername(%s): %w", username, err)
	}

	return &user, nil
}
