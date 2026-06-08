// Package service implements the business logic layer of CodeTasker.
// task_service.go processes GitHub webhook push events, runs the TODO parser
// across changed files, and persists discovered tasks to MongoDB. It also
// provides the task status update workflow used by the REST API.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/parser"
	"github.com/codetasker/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// WebhookPushPayload is the subset of GitHub's push event JSON payload that
// CodeTasker needs to determine which files changed and which repository they
// belong to.
//
// Reference: https://docs.github.com/en/webhooks/webhook-events-and-payloads#push
type WebhookPushPayload struct {
	// Ref is the full Git ref that was pushed (e.g. "refs/heads/main").
	Ref string `json:"ref"`

	// After is the commit SHA of the HEAD commit after the push.
	After string `json:"after"`

	// Repository contains metadata about the repository that received the push.
	Repository struct {
		// ID is the immutable numeric GitHub repository ID.
		ID int64 `json:"id"`

		// Name is the repository name without the owner prefix.
		Name string `json:"name"`

		// FullName is "owner/repo".
		FullName string `json:"full_name"`

		// Owner contains the repository owner's information.
		Owner struct {
			// Name is the owner's login (username or org name).
			Name string `json:"name"`
		} `json:"owner"`
	} `json:"repository"`

	// Commits is the list of commits included in this push event.
	// Each commit records which files were added, modified, or removed.
	Commits []struct {
		// ID is the commit SHA.
		ID string `json:"id"`

		// Added lists files that are new in this commit.
		Added []string `json:"added"`

		// Modified lists files that were changed in this commit.
		Modified []string `json:"modified"`

		// Removed lists files that were deleted in this commit.
		Removed []string `json:"removed"`
	} `json:"commits"`
}

// TaskService coordinates the webhook processing pipeline and task management
// operations. It depends on:
//   - *repository.TaskRepository for persistence.
//   - *repository.UserRepository to resolve the repo owner's account.
//   - *parser.Parser for TODO annotation extraction.
//   - *GithubService for fetching file contents via the GitHub API.
//   - *zap.Logger for structured logging.
type TaskService struct {
	taskRepo      *repository.TaskRepository
	userRepo      *repository.UserRepository
	parser        *parser.Parser
	githubService *GithubService
	log           *zap.Logger
}

// NewTaskService constructs a TaskService with its dependencies injected.
func NewTaskService(
	taskRepo *repository.TaskRepository,
	userRepo *repository.UserRepository,
	p *parser.Parser,
	githubService *GithubService,
	log *zap.Logger,
) *TaskService {
	return &TaskService{
		taskRepo:      taskRepo,
		userRepo:      userRepo,
		parser:        p,
		githubService: githubService,
		log:           log,
	}
}

// ProcessWebhookPush handles a GitHub push event by:
//  1. Collecting all unique file paths that were added or modified across all
//     commits in the event. Removed files are skipped since their tasks will
//     remain in the DB marked open/in-progress until a user resolves them.
//  2. Looking up the repo owner's CodeTasker account so we can fetch contents
//     with their OAuth token.
//  3. Fetching the current content of each changed file from the GitHub API.
//  4. Running the concurrent TODO parser over all fetched file contents.
//  5. Upserting every discovered ParsedTask as a domain.Task in MongoDB,
//     preserving any manually-set status values.
//
// The function is intentionally idempotent: receiving the same push event twice
// produces the same set of tasks in the database.
func (s *TaskService) ProcessWebhookPush(ctx context.Context, payload WebhookPushPayload) error {
	repoID := payload.Repository.ID
	repoName := payload.Repository.FullName
	owner := payload.Repository.Owner.Name
	repo := payload.Repository.Name
	commitSHA := payload.After

	// ── Collect unique changed file paths ─────────────────────────────────────
	// Use a map as a set to deduplicate files touched by multiple commits.
	changedFiles := make(map[string]struct{})
	for _, commit := range payload.Commits {
		for _, f := range commit.Added {
			changedFiles[f] = struct{}{}
		}
		for _, f := range commit.Modified {
			changedFiles[f] = struct{}{}
		}
		// Removed files are deliberately excluded — we leave their tasks in the
		// DB for the user to manually resolve.
	}

	if len(changedFiles) == 0 {
		s.log.Info("webhook push: no changed files", zap.String("repo", repoName))
		return nil
	}

	s.log.Info("webhook push: processing files",
		zap.String("repo", repoName),
		zap.Int("file_count", len(changedFiles)),
		zap.String("commit_sha", commitSHA),
	)

	// ── Resolve the repository owner's user account ───────────────────────────
	// We need a valid userID to authenticate GitHub API calls for file content.
	// If the owner hasn't signed up for CodeTasker, we skip — the next push
	// after they register will be processed normally.
	ownerUser, err := s.userRepo.FindByUsername(ctx, owner)
	if err != nil {
		return fmt.Errorf("ProcessWebhookPush FindByUsername(%s): %w", owner, err)
	}
	if ownerUser == nil {
		s.log.Warn("webhook push: owner has no CodeTasker account, skipping",
			zap.String("owner", owner),
			zap.String("repo", repoName),
		)
		return nil
	}

	// ── Fetch file contents via GitHub API ────────────────────────────────────
	var fileContents []parser.FileContent

	for filePath := range changedFiles {
		content, err := s.githubService.GetContents(ctx, ownerUser.ID, owner, repo, filePath, commitSHA)
		if err != nil {
			// Log and skip files that cannot be fetched (binary files, deleted in
			// a parallel race, permission errors) rather than aborting the whole event.
			s.log.Warn("webhook push: failed to fetch file content",
				zap.String("file", filePath),
				zap.Error(err),
			)
			continue
		}

		fileContents = append(fileContents, parser.FileContent{
			Path:    filePath,
			Content: content,
		})
	}

	if len(fileContents) == 0 {
		s.log.Info("webhook push: no fetchable files found", zap.String("repo", repoName))
		return nil
	}

	// ── Parse TODO annotations concurrently ───────────────────────────────────
	// Passing 0 workers causes ParseFiles to use runtime.NumCPU() automatically,
	// maximising throughput on multi-core servers.
	parsedTasks := s.parser.ParseFiles(fileContents, 0)

	s.log.Info("webhook push: annotation scan complete",
		zap.String("repo", repoName),
		zap.Int("annotations_found", len(parsedTasks)),
	)

	// ── Upsert discovered tasks into MongoDB ──────────────────────────────────
	now := time.Now().UTC()
	for _, pt := range parsedTasks {
		task := &domain.Task{
			RepoID:     repoID,
			RepoName:   repoName,
			FilePath:   pt.FilePath,
			LineNumber: pt.LineNumber,
			Content:    pt.Content,
			Type:       pt.Type,
			Status:     domain.TaskStatusOpen,
			CommitSHA:  commitSHA,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		if err := s.taskRepo.UpsertTask(ctx, task); err != nil {
			s.log.Error("webhook push: failed to upsert task",
				zap.String("file", pt.FilePath),
				zap.Int("line", pt.LineNumber),
				zap.Error(err),
			)
			// Continue processing remaining tasks — a single failed upsert should
			// not abort the entire event.
		}
	}

	return nil
}

// GetTasksByRepo returns all tasks for a repository identified by its numeric
// GitHub repository ID. Results are ordered by file path then line number.
func (s *TaskService) GetTasksByRepo(ctx context.Context, repoID int64) ([]domain.Task, error) {
	tasks, err := s.taskRepo.FindByRepo(ctx, repoID)
	if err != nil {
		return nil, fmt.Errorf("GetTasksByRepo(%d): %w", repoID, err)
	}

	return tasks, nil
}

// GetTaskByID retrieves a single task by its hex ObjectID string.
func (s *TaskService) GetTaskByID(ctx context.Context, id string) (*domain.Task, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("GetTaskByID: invalid task ID %q: %w", id, err)
	}
	return s.taskRepo.FindByID(ctx, objID)
}

// UpdateTaskStatus changes the lifecycle status of a task identified by its
// hex ObjectID string. It validates the ID format and the status value, applies
// the update, and returns the full refreshed task document for the API response.
func (s *TaskService) UpdateTaskStatus(ctx context.Context, id string, status domain.TaskStatus) (*domain.Task, error) {
	// Parse and validate the hex ObjectID format.
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("UpdateTaskStatus: invalid task ID %q: %w", id, err)
	}

	// Validate that the requested status is a known lifecycle value.
	switch status {
	case domain.TaskStatusOpen, domain.TaskStatusInProgress, domain.TaskStatusResolved:
		// valid — proceed
	default:
		return nil, fmt.Errorf("UpdateTaskStatus: unknown status value %q", status)
	}

	if err := s.taskRepo.UpdateStatus(ctx, objID, status); err != nil {
		return nil, fmt.Errorf("UpdateTaskStatus: %w", err)
	}

	// Re-fetch the updated task to return the full document to the API caller.
	task, err := s.taskRepo.FindByID(ctx, objID)
	if err != nil {
		return nil, fmt.Errorf("UpdateTaskStatus re-fetch: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task %s not found after update", id)
	}

	return task, nil
}

// UpdateTask changes the lifecycle status and/or PullRequestURL of a task.
func (s *TaskService) UpdateTask(ctx context.Context, id string, status domain.TaskStatus, prURL string) (*domain.Task, error) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return nil, fmt.Errorf("UpdateTask: invalid task ID %q: %w", id, err)
	}

	// If status is provided, validate it
	if status != "" {
		switch status {
		case domain.TaskStatusOpen, domain.TaskStatusInProgress, domain.TaskStatusResolved:
			// valid — proceed
		default:
			return nil, fmt.Errorf("UpdateTask: unknown status value %q", status)
		}
	}

	if err := s.taskRepo.UpdateTask(ctx, objID, status, prURL); err != nil {
		return nil, fmt.Errorf("UpdateTask: %w", err)
	}

	task, err := s.taskRepo.FindByID(ctx, objID)
	if err != nil {
		return nil, fmt.Errorf("UpdateTask re-fetch: %w", err)
	}
	if task == nil {
		return nil, fmt.Errorf("task %s not found after update", id)
	}

	return task, nil
}

// UpsertInjectedTask persists a task that was created via the inject endpoint
// (POST /api/tasks/inject) immediately, without waiting for the webhook push
// event to arrive after the PR is merged. This gives users instant feedback in
// the dashboard. The upsert key is (repo_id, file_path, line_number), so when
// the webhook eventually fires the document is updated in-place rather than
// duplicated.
func (s *TaskService) UpsertInjectedTask(ctx context.Context, task *domain.Task) error {
	if err := s.taskRepo.UpsertTask(ctx, task); err != nil {
		return fmt.Errorf("UpsertInjectedTask: %w", err)
	}

	s.log.Info("injected task upserted",
		zap.String("repo", task.RepoName),
		zap.String("file", task.FilePath),
		zap.Int("line", task.LineNumber),
	)

	return nil
}
