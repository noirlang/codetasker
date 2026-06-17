// Package repository provides data access objects for CodeTasker's MongoDB
// collections. The task_repo.go file implements all CRUD operations for the
// tasks collection, keeping the service layer free of raw driver calls.
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

// TaskRepository wraps the MongoDB tasks collection and exposes typed methods
// for all task-related persistence operations used by the webhook pipeline,
// the task service, and the inject workflow.
type TaskRepository struct {
	col *mongo.Collection
}

// NewTaskRepository creates a TaskRepository backed by the "tasks" collection
// in the provided database.
func NewTaskRepository(db *database.Database) *TaskRepository {
	return &TaskRepository{
		col: db.Collection("tasks"),
	}
}

// UpsertTask creates or updates a task document identified by the combination
// of (repo_id, file_path, line_number). This triple uniquely identifies a
// comment annotation within a repository, so repeated webhook events for the
// same file do not create duplicate documents.
//
// On insert, created_at and status are set. On update, only mutable fields
// (content, type, commit_sha, updated_at) are refreshed so that a user's
// manual status change (e.g. "in_progress") is preserved across pushes.
func (r *TaskRepository) UpsertTask(ctx context.Context, task *domain.Task) error {
	now := time.Now().UTC()

	filter := bson.M{
		"repo_id":     task.RepoID,
		"file_path":   task.FilePath,
		"line_number": task.LineNumber,
	}

	update := bson.M{
		"$set": bson.M{
			"repo_name":  task.RepoName,
			"content":    task.Content,
			"type":       task.Type,
			"commit_sha": task.CommitSHA,
			"updated_at": now,
		},
		// Only written on first insert — preserves manual status changes.
		"$setOnInsert": bson.M{
			"status":                domain.TaskStatusOpen,
			"created_at":            now,
			"created_by_username":   task.CreatedByUsername,
			"created_by_avatar_url": task.CreatedByAvatarURL,
			"maintainer_username":   task.MaintainerUsername,
			"maintainer_email":      task.MaintainerEmail,
		},
	}

	opts := options.Update().SetUpsert(true)

	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("UpsertTask (repo=%d file=%s line=%d): %w",
			task.RepoID, task.FilePath, task.LineNumber, err)
	}

	return nil
}

// FindByRepo returns all task documents for the given repository ID, ordered
// by file path then line number for consistent display ordering. If no tasks
// exist for the repository yet an empty (non-nil) slice is returned.
func (r *TaskRepository) FindByRepo(ctx context.Context, repoID int64) ([]domain.Task, error) {
	filter := bson.M{"repo_id": repoID}
	findOpts := options.Find().SetSort(bson.D{
		{Key: "file_path", Value: 1},
		{Key: "line_number", Value: 1},
	})

	cursor, err := r.col.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, fmt.Errorf("FindByRepo(%d): %w", repoID, err)
	}
	defer cursor.Close(ctx)

	var tasks []domain.Task
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("FindByRepo cursor decode: %w", err)
	}

	// Return an empty slice rather than nil so JSON marshalling yields [] not null.
	if tasks == nil {
		tasks = []domain.Task{}
	}

	return tasks, nil
}

// FindByID retrieves a single task by its MongoDB ObjectID.
// Returns (nil, nil) if no document with that ID exists.
func (r *TaskRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*domain.Task, error) {
	var task domain.Task

	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("FindByID(%s): %w", id.Hex(), err)
	}

	return &task, nil
}

// UpdateStatus atomically updates the status and updated_at fields of the task
// identified by id. It does not touch any other fields, preserving all
// pipeline-managed data (content, type, commit_sha, etc.).
func (r *TaskRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status domain.TaskStatus) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"status":     status,
			"updated_at": time.Now().UTC(),
		},
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("UpdateStatus(%s, %s): %w", id.Hex(), status, err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("task %s not found", id.Hex())
	}

	return nil
}

// UpdateTask atomically updates the status, pr_url, issue_url, and updated_at fields of the task
// identified by id.
func (r *TaskRepository) UpdateTask(ctx context.Context, id primitive.ObjectID, status domain.TaskStatus, prURL string, issueURL string) error {
	filter := bson.M{"_id": id}
	setFields := bson.M{
		"updated_at": time.Now().UTC(),
	}
	if status != "" {
		setFields["status"] = status
	}
	if prURL != "" {
		if prURL == "clear" {
			setFields["pr_url"] = ""
		} else {
			setFields["pr_url"] = prURL
		}
	}
	if issueURL != "" {
		if issueURL == "clear" {
			setFields["issue_url"] = ""
		} else {
			setFields["issue_url"] = issueURL
		}
	}

	update := bson.M{
		"$set": setFields,
	}

	result, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("UpdateTask(%s): %w", id.Hex(), err)
	}

	if result.MatchedCount == 0 {
		return fmt.Errorf("task %s not found", id.Hex())
	}

	return nil
}

// DeleteByRepo removes all task documents belonging to the given repository.
// This is called when a GitHub repository delete event is received so the
// database does not accumulate stale task records.
func (r *TaskRepository) DeleteByRepo(ctx context.Context, repoID int64) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"repo_id": repoID})
	if err != nil {
		return fmt.Errorf("DeleteByRepo(%d): %w", repoID, err)
	}

	return nil
}

// DeleteTask removes a single task by its ObjectID.
func (r *TaskRepository) DeleteTask(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("DeleteTask(%s): %w", id.Hex(), err)
	}
	return nil
}

// UpdateAssignee sets or clears the assignee for a task identified by id.
// When assigneeID is non-nil the username and avatarURL are written alongside it.
// When assigneeID is nil all three assignee fields are cleared (empty strings / nil).
func (r *TaskRepository) UpdateAssignee(ctx context.Context, id primitive.ObjectID, assigneeID *primitive.ObjectID, username, avatarURL string) error {
	filter := bson.M{"_id": id}
	setFields := bson.M{"updated_at": time.Now().UTC()}
	if assigneeID != nil {
		setFields["assignee_id"] = assigneeID
		setFields["assignee_username"] = username
		setFields["assignee_avatar_url"] = avatarURL
	} else {
		setFields["assignee_id"] = nil
		setFields["assignee_username"] = ""
		setFields["assignee_avatar_url"] = ""
	}
	update := bson.M{"$set": setFields}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("UpdateAssignee(%s): %w", id.Hex(), err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("task %s not found", id.Hex())
	}
	return nil
}

// UpdateCompletedBy sets the completed_by fields and completed_at timestamp for a resolved task.
func (r *TaskRepository) UpdateCompletedBy(ctx context.Context, id primitive.ObjectID, username, avatarURL string) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"completed_by_username":   username,
			"completed_by_avatar_url": avatarURL,
			"completed_at":            now,
			"updated_at":              now,
		},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("UpdateCompletedBy(%s): %w", id.Hex(), err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("task %s not found", id.Hex())
	}
	return nil
}

// UpdateMaintainer sets the maintainer_username and maintainer_email for a task.
func (r *TaskRepository) UpdateMaintainer(ctx context.Context, id primitive.ObjectID, username, email string) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"maintainer_username": username,
			"maintainer_email":    email,
			"updated_at":          time.Now().UTC(),
		},
	}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("UpdateMaintainer(%s): %w", id.Hex(), err)
	}
	if res.MatchedCount == 0 {
		return fmt.Errorf("task %s not found", id.Hex())
	}
	return nil
}
