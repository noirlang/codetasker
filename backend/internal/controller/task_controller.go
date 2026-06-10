// Package controller implements the HTTP handler layer of CodeTasker.
// task_controller.go exposes endpoints for listing tasks by repository,
// updating task status, and injecting new TODO comments into repositories.
// All routes require a valid JWT (Protected middleware applied at group level).
package controller

import (
	"strconv"

	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/repository"
	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
)

// TaskController handles all task-related HTTP endpoints, delegating business
// logic to TaskService (for persistence) and GithubService (for code injection).
type TaskController struct {
	taskService      *service.TaskService
	githubService    *service.GithubService
	syncedRepoRepo   *repository.SyncedRepository
	collaboratorRepo *repository.CollaboratorRepository
}

// NewTaskController constructs a TaskController with its dependencies injected.
func NewTaskController(
	taskService *service.TaskService,
	githubService *service.GithubService,
	syncedRepoRepo *repository.SyncedRepository,
	collaboratorRepo *repository.CollaboratorRepository,
) *TaskController {
	return &TaskController{
		taskService:      taskService,
		githubService:    githubService,
		syncedRepoRepo:   syncedRepoRepo,
		collaboratorRepo: collaboratorRepo,
	}
}

// RegisterRoutes mounts all task routes onto the provided Fiber router group.
// The group is expected to already have the Protected JWT middleware applied.
func (tc *TaskController) RegisterRoutes(group fiber.Router) {
	group.Get("/tasks", tc.ListTasksByRepo)
	group.Patch("/tasks/:id", tc.UpdateTaskStatus)
	group.Post("/tasks/inject", tc.InjectTODO)
}

// ListTasksByRepo returns all tasks for a repository, identified by the
// numeric GitHub repository ID in the `?repo_id=` query parameter.
//
// Route: GET /api/tasks?repo_id=<id>
func (tc *TaskController) ListTasksByRepo(c *fiber.Ctx) error {
	// Validate and parse repo_id query parameter.
	repoIDStr := c.Query("repo_id")
	if repoIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "query parameter 'repo_id' is required",
		})
	}

	repoID, err := strconv.ParseInt(repoIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_parameter",
			"message": "repo_id must be a valid integer",
		})
	}

	tasks, err := tc.taskService.GetTasksByRepo(c.Context(), repoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_tasks_failed",
			"message": err.Error(),
		})
	}

	if tasks == nil {
		tasks = []domain.Task{}
	}

	return c.JSON(fiber.Map{
		"tasks":   tasks,
		"count":   len(tasks),
		"repo_id": repoID,
	})
}

// UpdateTaskStatus changes the lifecycle status of a task.
// The request body must contain a JSON object with the "status" field set to
// one of: "open", "in_progress", or "resolved".
//
// Route: PATCH /api/tasks/:id
func (tc *TaskController) UpdateTaskStatus(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	taskID := c.Params("id")
	if taskID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "task ID is required",
		})
	}

	task, err := tc.taskService.GetTaskByID(c.Context(), taskID)
	if err != nil || task == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "task not found",
		})
	}

	// Verify collaborator permissions
	synced, err := tc.syncedRepoRepo.FindByRepoIDOnly(c.Context(), task.RepoID)
	if err == nil && synced != nil {
		if synced.UserID != userID {
			collab, _ := tc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
			if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "forbidden",
					"message": "You do not have permissions to modify tasks in this repository",
				})
			}
		}
	}

	var req domain.UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_body",
			"message": "request body must be valid JSON with 'status' and/or 'pr_url' fields",
		})
	}

	if req.Status == "" && req.PullRequestURL == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_fields",
			"message": "at least one of 'status' or 'pr_url' field is required",
		})
	}

	updatedTask, err := tc.taskService.UpdateTask(c.Context(), taskID, req.Status, req.PullRequestURL)
	if err != nil {
		// If the error message indicates the task was not found, return 404.
		if isNotFoundError(err.Error()) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "task not found: " + taskID,
			})
		}
		// Invalid ID format or unknown status — return 400.
		if isValidationError(err.Error()) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "validation_error",
				"message": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"task":    updatedTask,
		"message": "task status updated successfully",
	})
}

// InjectTODO parses an InjectTaskRequest, calls GithubService.InjectTODO to
// create a PR with the new comment, then upserts the task record in the database
// so it immediately appears in the user's task dashboard.
//
// Route: POST /api/tasks/inject
func (tc *TaskController) InjectTODO(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	var req domain.InjectTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_body",
			"message": "request body must be valid JSON",
		})
	}

	// Validate required fields.
	if req.RepoOwner == "" || req.RepoName == "" || req.FilePath == "" ||
		req.Description == "" || req.Branch == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_fields",
			"message": "repo_owner, repo_name, file_path, description, and branch are required",
		})
	}

	if req.LineNumber < 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_field",
			"message": "line_number must be >= 1",
		})
	}

	// Verify collaborator permissions before injecting TODO
	synced, err := tc.syncedRepoRepo.FindByRepoName(c.Context(), req.RepoOwner+"/"+req.RepoName)
	if err == nil && synced != nil {
		if synced.UserID != userID {
			collab, _ := tc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
			if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "forbidden",
					"message": "You do not have permissions to inject tasks in this repository",
				})
			}
		}
	}

	// Inject the TODO via the GitHub API pipeline and get back the PR URL.
	prURL, err := tc.githubService.InjectTODO(c.Context(), userID, &req)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "inject_failed",
			"message": err.Error(),
		})
	}

	// Upsert the new task in MongoDB so it appears immediately without waiting
	// for the webhook to fire and process the PR merge.
	task := &domain.Task{
		RepoID:     synced.RepoID,
		RepoName:   req.RepoOwner + "/" + req.RepoName,
		FilePath:   req.FilePath,
		LineNumber: req.LineNumber,
		Content:    req.Description,
		Type:       "TODO",
		Status:     domain.TaskStatusOpen,
	}

	if err := tc.taskService.UpsertInjectedTask(c.Context(), task); err != nil {
		// Log but don't fail — the PR was created successfully; the task will
		// appear in the DB after the webhook processes the merged commit.
		_ = err
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "TODO injected successfully",
		"pr_url":  prURL,
	})
}

// isNotFoundError checks whether an error message indicates a "not found" condition.
// This avoids importing a custom error type just for this check.
func isNotFoundError(msg string) bool {
	return len(msg) > 9 && (msg[len(msg)-9:] == "not found" ||
		containsStr(msg, "not found"))
}

// isValidationError checks whether an error message indicates a validation problem.
func isValidationError(msg string) bool {
	return containsStr(msg, "invalid") || containsStr(msg, "unknown status")
}

// containsStr is a simple helper wrapping a substring check to avoid importing strings
// package just for Contains (already available via fiber/utils in some builds).
func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
