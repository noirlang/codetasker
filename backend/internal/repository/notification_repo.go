// Package repository provides data access objects for CodeTasker's MongoDB
// collections. notification_repo.go implements persistence operations for the
// notifications collection, storing per-user in-app notification records.
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

// NotificationRepository wraps the MongoDB notifications collection.
type NotificationRepository struct {
	col *mongo.Collection
}

// NewNotificationRepository creates a NotificationRepository backed by the "notifications" collection.
func NewNotificationRepository(db *database.Database) *NotificationRepository {
	return &NotificationRepository{col: db.Collection("notifications")}
}

// Create inserts a new notification. It auto-assigns the ID, CreatedAt, and sets Read = false.
func (r *NotificationRepository) Create(ctx context.Context, n *domain.Notification) error {
	n.ID = primitive.NewObjectID()
	n.CreatedAt = time.Now().UTC()
	n.Read = false
	_, err := r.col.InsertOne(ctx, n)
	if err != nil {
		return fmt.Errorf("NotificationRepository.Create: %w", err)
	}
	return nil
}

// FindByUser returns the most recent 50 notifications for a user, ordered newest-first.
// Returns an empty (non-nil) slice if none exist.
func (r *NotificationRepository) FindByUser(ctx context.Context, userID primitive.ObjectID) ([]domain.Notification, error) {
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(50)
	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepository.FindByUser: %w", err)
	}
	defer cursor.Close(ctx)
	var notifs []domain.Notification
	if err := cursor.All(ctx, &notifs); err != nil {
		return nil, fmt.Errorf("NotificationRepository.FindByUser decode: %w", err)
	}
	if notifs == nil {
		notifs = []domain.Notification{}
	}
	return notifs, nil
}

// MarkRead marks a single notification as read by its ID.
// Returns an error if the notification is not found.
func (r *NotificationRepository) MarkRead(ctx context.Context, id primitive.ObjectID) error {
	res, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{"read": true}})
	if err != nil {
		return fmt.Errorf("NotificationRepository.MarkRead: %w", err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("notification %s not found", id.Hex())
	}
	return nil
}

// MarkAllRead marks all unread notifications as read for a user.
func (r *NotificationRepository) MarkAllRead(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.col.UpdateMany(
		ctx,
		bson.M{"user_id": userID, "read": false},
		bson.M{"$set": bson.M{"read": true}},
	)
	return err
}

// UnreadCount returns the number of unread notifications for a user.
func (r *NotificationRepository) UnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error) {
	count, err := r.col.CountDocuments(ctx, bson.M{"user_id": userID, "read": false})
	if err != nil {
		return 0, fmt.Errorf("NotificationRepository.UnreadCount: %w", err)
	}
	return count, nil
}

// FindByID returns a notification by its ObjectID.
// Returns (nil, nil) if no document with that ID exists.
func (r *NotificationRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Notification, error) {
	var n domain.Notification
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&n)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, fmt.Errorf("NotificationRepository.FindByID: %w", err)
	}
	return &n, nil
}
