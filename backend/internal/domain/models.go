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

	// Email is the user's email address used for notification delivery.
	Email string `bson:"email,omitempty" json:"email,omitempty"`

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

	// Telegram integration settings
	TelegramBotToken string `bson:"telegram_bot_token,omitempty" json:"telegram_bot_token,omitempty"`
	TelegramChatID   string `bson:"telegram_chat_id,omitempty" json:"telegram_chat_id,omitempty"`
	TelegramEnabled  bool   `bson:"telegram_enabled" json:"telegram_enabled"`
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

	// IssueURL is the linked GitHub Issue URL.
	IssueURL string `bson:"issue_url,omitempty" json:"issue_url,omitempty"`

	// CommitSHA is the Git commit hash at which this annotation was last seen.
	// Useful for linking to the exact commit on GitHub.
	CommitSHA string `bson:"commit_sha" json:"commit_sha"`

	// CreatedAt is set once when the task document is first inserted.
	CreatedAt time.Time `bson:"created_at" json:"created_at"`

	// UpdatedAt is refreshed on every upsert or status change.
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`

	// AssigneeID is the MongoDB ObjectID of the assigned user (optional).
	AssigneeID *primitive.ObjectID `bson:"assignee_id,omitempty" json:"assignee_id,omitempty"`

	// AssigneeUsername is the GitHub login of the assigned user.
	AssigneeUsername string `bson:"assignee_username,omitempty" json:"assignee_username,omitempty"`

	// AssigneeAvatarURL is the avatar of the assigned user.
	AssigneeAvatarURL string `bson:"assignee_avatar_url,omitempty" json:"assignee_avatar_url,omitempty"`

	// CreatedByUsername is the GitHub login of the user who injected this task.
	// Only populated for tasks created via the inject endpoint (not webhook-parsed).
	CreatedByUsername string `bson:"created_by_username,omitempty" json:"created_by_username,omitempty"`

	// CreatedByAvatarURL is the avatar of the task creator.
	CreatedByAvatarURL string `bson:"created_by_avatar_url,omitempty" json:"created_by_avatar_url,omitempty"`

	// CompletedByUsername is the GitHub login of the user who resolved this task.
	// Set when a user manually marks the task as "resolved" via the API.
	CompletedByUsername string `bson:"completed_by_username,omitempty" json:"completed_by_username,omitempty"`

	// CompletedByAvatarURL is the avatar of the user who completed the task.
	CompletedByAvatarURL string `bson:"completed_by_avatar_url,omitempty" json:"completed_by_avatar_url,omitempty"`

	// CompletedAt is the timestamp when the task was marked resolved.
	CompletedAt *time.Time `bson:"completed_at,omitempty" json:"completed_at,omitempty"`

	// MaintainerUsername is the GitHub login of the maintainer responsible for this file path.
	// Resolved via CODEOWNERS lookup on task creation/sync.
	MaintainerUsername string `bson:"maintainer_username,omitempty" json:"maintainer_username,omitempty"`

	// MaintainerEmail is the email address of the maintainer, used for email notifications.
	MaintainerEmail string `bson:"maintainer_email,omitempty" json:"maintainer_email,omitempty"`
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

	// IssueURL is the linked GitHub Issue URL.
	IssueURL string `json:"issue_url,omitempty"`
}

// UpdateTaskRequest is the request body for PATCH /api/tasks/:id.
// It allows updating the lifecycle status, linking a PR URL, assigning a user, or clearing the assignee.
type UpdateTaskRequest struct {
	// Status is the new lifecycle state to apply to the task.
	Status TaskStatus `json:"status,omitempty"`

	// PullRequestURL is the Pull Request URL linked to the task.
	PullRequestURL string `json:"pr_url,omitempty"`

	// IssueURL is the GitHub Issue URL linked to the task.
	IssueURL string `json:"issue_url,omitempty"`

	// AssigneeUsername is the GitHub login of the user to assign to this task.
	AssigneeUsername string `json:"assignee_username,omitempty"`

	// ClearAssignee when true removes any existing assignee from the task.
	ClearAssignee bool `json:"clear_assignee,omitempty"`
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

// DebtAnalysisRun stores one technical-debt analysis execution for a repository.
// Source code is intentionally not stored; only metrics, file paths, and
// generated task descriptions are persisted.
type DebtAnalysisRun struct {
	ID                    primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RepoID                int64              `bson:"repo_id" json:"repo_id"`
	RepoName              string             `bson:"repo_name" json:"repo_name"`
	UserID                primitive.ObjectID `bson:"user_id" json:"user_id"`
	AnalyzedAt            time.Time          `bson:"analyzed_at" json:"analyzed_at"`
	Days                  int                `bson:"days" json:"days"`
	HourlyEngineerCostUSD float64            `bson:"hourly_engineer_cost_usd" json:"hourly_engineer_cost_usd"`
	Summary               DebtSummary        `bson:"summary" json:"summary"`
}

type DebtSummary struct {
	FilesAnalyzed        int     `bson:"files_analyzed" json:"files_analyzed"`
	Critical             int     `bson:"critical" json:"critical"`
	High                 int     `bson:"high" json:"high"`
	Medium               int     `bson:"medium" json:"medium"`
	Low                  int     `bson:"low" json:"low"`
	EstimatedMonthlyCost float64 `bson:"estimated_monthly_cost" json:"estimated_monthly_cost"`
}

// DebtFileMetric stores per-file debt metrics for a run.
type DebtFileMetric struct {
	ID                   primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RunID                primitive.ObjectID `bson:"run_id" json:"run_id"`
	RepoID               int64              `bson:"repo_id" json:"repo_id"`
	RepoName             string             `bson:"repo_name" json:"repo_name"`
	FilePath             string             `bson:"file_path" json:"file_path"`
	DebtScore            int                `bson:"debt_score" json:"debt_score"`
	Level                string             `bson:"level" json:"level"`
	Metrics              DebtMetrics        `bson:"metrics" json:"metrics"`
	EstimatedMonthlyCost float64            `bson:"estimated_monthly_cost" json:"estimated_monthly_cost"`
	Reasons              []string           `bson:"reasons" json:"reasons"`
	CreatedAt            time.Time          `bson:"created_at" json:"created_at"`
}

type DebtMetrics struct {
	CommitCount                  int        `bson:"commit_count" json:"commit_count"`
	ChurnAdded                   int        `bson:"churn_added" json:"churn_added"`
	ChurnDeleted                 int        `bson:"churn_deleted" json:"churn_deleted"`
	TotalChurn                   int        `bson:"total_churn" json:"total_churn"`
	AuthorCount                  int        `bson:"author_count" json:"author_count"`
	LastTouchedAt                *time.Time `bson:"last_touched_at,omitempty" json:"last_touched_at,omitempty"`
	BugfixCommitCount            int        `bson:"bugfix_commit_count" json:"bugfix_commit_count"`
	LOC                          int        `bson:"loc" json:"loc"`
	FunctionCount                int        `bson:"function_count" json:"function_count"`
	AvgFunctionLength            float64    `bson:"avg_function_length" json:"avg_function_length"`
	MaxFunctionLength            int        `bson:"max_function_length" json:"max_function_length"`
	NestingDepthEstimate         int        `bson:"nesting_depth_estimate" json:"nesting_depth_estimate"`
	CyclomaticComplexityEstimate int        `bson:"cyclomatic_complexity_estimate" json:"cyclomatic_complexity_estimate"`
	TodoCount                    int        `bson:"todo_count" json:"todo_count"`
	DuplicateImportCount         int        `bson:"duplicate_import_count" json:"duplicate_import_count"`
	HasTests                     bool       `bson:"has_tests" json:"has_tests"`
	CoverageStatus               string     `bson:"coverage_status" json:"coverage_status"`
}

// DebtSuggestedTask stores a generated task proposal for a high/critical file.
type DebtSuggestedTask struct {
	ID                   primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	RunID                primitive.ObjectID  `bson:"run_id" json:"run_id"`
	RepoID               int64               `bson:"repo_id" json:"repo_id"`
	RepoName             string              `bson:"repo_name" json:"repo_name"`
	FilePath             string              `bson:"file_path" json:"file_path"`
	Title                string              `bson:"title" json:"title"`
	Description          string              `bson:"description" json:"description"`
	Actions              []string            `bson:"actions" json:"actions"`
	DebtScore            int                 `bson:"debt_score" json:"debt_score"`
	Level                string              `bson:"level" json:"level"`
	EstimatedMonthlyCost float64             `bson:"estimated_monthly_cost" json:"estimated_monthly_cost"`
	Status               string              `bson:"status" json:"status"`
	CreatedTaskID        *primitive.ObjectID `bson:"created_task_id,omitempty" json:"created_task_id,omitempty"`
	CreatedAt            time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt            time.Time           `bson:"updated_at" json:"updated_at"`
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

// Comment represents a user comment on a task.
type Comment struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	TaskID    primitive.ObjectID `bson:"task_id" json:"task_id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Username  string             `bson:"username" json:"username"`
	AvatarURL string             `bson:"avatar_url" json:"avatar_url"`
	Content   string             `bson:"content" json:"content"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}

// NotificationType classifies the kind of event that triggered a notification.
type NotificationType string

const (
	// NotifTaskAssigned is sent when a user is assigned to a task.
	NotifTaskAssigned NotificationType = "task_assigned"

	// NotifCommentAdded is sent when someone comments on a task the user is involved with.
	NotifCommentAdded NotificationType = "comment_added"

	// NotifPRMerged is sent when a pull request linked to a task is merged.
	NotifPRMerged NotificationType = "pr_merged"

	// NotifTaskCompleted is sent to the maintainer when a developer marks a task as resolved.
	NotifTaskCompleted NotificationType = "task_completed"
)

// Notification represents a notification sent to a user.
type Notification struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Type      NotificationType   `bson:"type" json:"type"`
	Title     string             `bson:"title" json:"title"`
	Message   string             `bson:"message" json:"message"`
	Link      string             `bson:"link,omitempty" json:"link,omitempty"`
	Read      bool               `bson:"read" json:"read"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

// ActivityLog represents an activity event in a repository.
type ActivityLog struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RepoID      int64              `bson:"repo_id" json:"repo_id"`
	RepoName    string             `bson:"repo_name" json:"repo_name"`
	ActorID     primitive.ObjectID `bson:"actor_id" json:"actor_id"`
	ActorName   string             `bson:"actor_name" json:"actor_name"`
	ActorAvatar string             `bson:"actor_avatar" json:"actor_avatar"`
	// Action describes the event type, e.g. "task_created", "task_assigned", "status_changed", "comment_added".
	Action string `bson:"action" json:"action"`
	// TargetType is one of: "task", "pr", "collaborator".
	TargetType string `bson:"target_type" json:"target_type"`
	TargetID   string `bson:"target_id" json:"target_id"`
	// TargetLabel is a human-readable description of the target.
	TargetLabel string            `bson:"target_label" json:"target_label"`
	Meta        map[string]string `bson:"meta,omitempty" json:"meta,omitempty"`
	CreatedAt   time.Time         `bson:"created_at" json:"created_at"`
}

// CodeOwnerRule maps a file path glob pattern to a list of GitHub usernames.
type CodeOwnerRule struct {
	ID      primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RepoID  int64              `bson:"repo_id" json:"repo_id"`
	Pattern string             `bson:"pattern" json:"pattern"`
	// Owners is a list of GitHub usernames (without the @ prefix).
	Owners    []string  `bson:"owners" json:"owners"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}
