// Package domain defines the core business models used throughout CodeTasker.
// These structs are shared by every layer (repository, service, controller)
// and are the single source of truth for the shape of data at rest (MongoDB)
// and in transit (JSON API responses).
package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskStatus represents the lifecycle state of a TODO/FIXME task discovered
// in a repository. The allowed values are constrained by the TaskStatus constants.
type TaskStatus string

const (
	// TaskStatusOpen is the default state when a task is first detected.
	TaskStatusOpen TaskStatus = "open"

	// TaskStatusInProgress indicates that work has begun on this task.
	TaskStatusInProgress TaskStatus = "in_progress"

	// TaskStatusResolved indicates the task has been completed and the
	// annotation has been removed from the codebase.
	TaskStatusResolved TaskStatus = "resolved"
)

// User represents a CodeTasker user authenticated via GitHub OAuth.
// AccessToken is intentionally omitted from JSON serialization (json:"-")
// because it is stored encrypted in MongoDB and must never be sent to clients.
type User struct {
	// ID is the MongoDB document identifier, auto-generated on insert.
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// GithubID is the immutable numeric user ID assigned by GitHub.
	// Used as the stable upsert key across OAuth sessions.
	GithubID int64 `bson:"github_id" json:"github_id"`

	// Username is the GitHub login handle (e.g. "octocat").
	Username string `bson:"username" json:"username"`

	// AvatarURL points to the user's GitHub profile picture.
	AvatarURL string `bson:"avatar_url" json:"avatar_url"`

	// AccessToken is the AES-256-GCM encrypted GitHub OAuth access token.
	// The raw (plaintext) token is NEVER stored. The field is excluded from
	// JSON responses via the `json:"-"` tag.
	AccessToken string `bson:"access_token" json:"-"`

	// CreatedAt is set once when the user document is first inserted.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`

	// UpdatedAt is refreshed on every upsert (e.g. token refresh).
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// Task represents a single TODO/FIXME/HACK/BUG/NOTE annotation found in a
// repository file. Tasks are upserted on every webhook push event so the
// database reflects the current state of the codebase.
type Task struct {
	// ID is the MongoDB document identifier.
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// RepoID is the GitHub repository numeric ID, used as the stable
	// foreign key (repo names can be renamed; IDs cannot).
	RepoID int64 `bson:"repo_id" json:"repo_id"`

	// RepoName is the human-readable "owner/repo" full name stored for display.
	RepoName string `bson:"repo_name" json:"repo_name"`

	// FilePath is the repository-relative path to the file containing the task,
	// e.g. "src/handlers/auth.go".
	FilePath string `bson:"file_path" json:"file_path"`

	// LineNumber is the 1-based line number of the annotation.
	LineNumber int `bson:"line_number" json:"line_number"`

	// Content is the trimmed text following the annotation keyword,
	// e.g. "refactor this to use context cancellation".
	Content string `bson:"content" json:"content"`

	// Type is one of: TODO, FIXME, HACK, BUG, NOTE.
	Type string `bson:"type" json:"type"`

	// Status tracks the lifecycle of this task (open → in_progress → resolved).
	Status TaskStatus `bson:"status" json:"status"`

	// PullRequestURL is the linked GitHub Pull Request URL.
	PullRequestURL string `bson:"pr_url,omitempty" json:"pr_url,omitempty"`

	// CommitSHA is the Git commit hash at which this annotation was last seen.
	// Useful for linking to the exact commit on GitHub.
	CommitSHA string `bson:"commit_sha" json:"commit_sha"`

	// CreatedAt is set once when the task document is first inserted.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`

	// UpdatedAt is refreshed on every upsert or status change.
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// InjectTaskRequest is the request body for POST /api/tasks/inject.
// It describes where and what TODO comment the user wants to insert
// into their repository via the GitHub API.
type InjectTaskRequest struct {
	// RepoOwner is the GitHub username or organisation that owns the repository.
	RepoOwner string `json:"repo_owner" validate:"required"`

	// RepoName is the repository name (not full name, just the repo part).
	RepoName string `json:"repo_name" validate:"required"`

	// FilePath is the repository-relative path of the file to modify.
	FilePath string `json:"file_path" validate:"required"`

	// LineNumber is the 1-based line number at which the TODO comment is inserted.
	LineNumber int `json:"line_number" validate:"required,min=1"`

	// Description is the human-readable text that will appear after the TODO keyword.
	Description string `json:"description" validate:"required"`

	// Branch is the base branch to read from and create the PR against.
	Branch string `json:"branch" validate:"required"`

	// Type is the comment tag type, e.g. "TODO", "FIXME", "BUG", "HACK", "NOTE".
	Type string `json:"type"`
}

// UpdateTaskRequest is the request body for PATCH /api/tasks/:id.
// It allows updating the lifecycle status and/or linking a PR URL to the task.
type UpdateTaskRequest struct {
	// Status is the new lifecycle state to apply to the task.
	Status TaskStatus `json:"status,omitempty"`

	// PullRequestURL is the Pull Request URL linked to the task.
	PullRequestURL string `json:"pr_url,omitempty"`
}

// SyncedRepo represents a repository that has been activated/synced by a user.
// This is used to display the active sync state on the frontend dashboard.
type SyncedRepo struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RepoID    int64              `bson:"repo_id" json:"repo_id"`
	RepoName  string             `bson:"repo_name" json:"repo_name"`
	Owner     string             `bson:"owner" json:"owner"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// RepoRole represents the permission level of a collaborator on a repository.
type RepoRole string

const (
	RoleOwner      RepoRole = "owner"
	RoleMaintainer RepoRole = "maintainer"
	RoleDeveloper  RepoRole = "developer"
	RoleViewer     RepoRole = "viewer"
)

// Collaborator links a CodeTasker user to a repository with specific permissions/roles.
type Collaborator struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RepoID    int64              `bson:"repo_id" json:"repo_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	AvatarURL string             `bson:"avatar_url" json:"avatar_url"`
	Role      RepoRole           `bson:"role" json:"role"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
