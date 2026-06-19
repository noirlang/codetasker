// Package controller implements the HTTP handler layer of CodeTasker.
// task_controller.go exposes endpoints for listing tasks by repository,
// updating task status/assignee, injecting new TODO comments, and managing
// per-task comments. All routes require a valid JWT (Protected middleware
// applied at group level).
package controller

import (
	"fmt"
	"strconv"

	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/repository"
	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TaskController handles all task-related HTTP endpoints, delegating business
// logic to TaskService (for persistence) and GithubService (for code injection).
type TaskController struct {
	taskService      *service.TaskService
	githubService    *service.GithubService
	syncedRepoRepo   *repository.SyncedRepository
	collaboratorRepo *repository.CollaboratorRepository
	commentRepo      *repository.CommentRepository
	notifRepo        *repository.NotificationRepository
	activityRepo     *repository.ActivityRepository
	userRepo         *repository.UserRepository
	emailService     *service.EmailService
	codeOwnerService *service.CodeOwnerService
	telegramService  *service.TelegramService
	// taskRepo is kept for direct UpdateAssignee calls.
	taskRepo *repository.TaskRepository
}

// NewTaskController constructs a TaskController with its dependencies injected.
func NewTaskController(
	taskService *service.TaskService,
	githubService *service.GithubService,
	syncedRepoRepo *repository.SyncedRepository,
	collaboratorRepo *repository.CollaboratorRepository,
	commentRepo *repository.CommentRepository,
	notifRepo *repository.NotificationRepository,
	activityRepo *repository.ActivityRepository,
	userRepo *repository.UserRepository,
	emailService *service.EmailService,
	codeOwnerService *service.CodeOwnerService,
	telegramService *service.TelegramService,
	taskRepo *repository.TaskRepository,
) *TaskController {
	return &TaskController{
		taskService:      taskService,
		githubService:    githubService,
		syncedRepoRepo:   syncedRepoRepo,
		collaboratorRepo: collaboratorRepo,
		commentRepo:      commentRepo,
		notifRepo:        notifRepo,
		activityRepo:     activityRepo,
		userRepo:         userRepo,
		emailService:     emailService,
		codeOwnerService: codeOwnerService,
		telegramService:  telegramService,
		taskRepo:         taskRepo,
	}
}

// RegisterRoutes mounts all task routes onto the provided Fiber router group.
// The group is expected to already have the Protected JWT middleware applied.
func (tc *TaskController) RegisterRoutes(group fiber.Router) {
	group.Get("/tasks", tc.ListTasksByRepo)
	group.Patch("/tasks/:id", tc.UpdateTaskStatus)
	group.Post("/tasks/inject", tc.InjectTODO)
	// Comment sub-resources on tasks.
	group.Get("/tasks/:id/comments", tc.ListComments)
	group.Post("/tasks/:id/comments", tc.AddComment)
	group.Delete("/tasks/:id/comments/:commentId", tc.DeleteComment)
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

// UpdateTaskStatus changes the lifecycle status of a task, optionally sets a PR URL,
// and handles assignee updates (set or clear). Activity logs and notifications are
// created for assignee changes.
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

	// Verify collaborator permissions.
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
			"message": "request body must be valid JSON with 'status', 'pr_url', 'issue_url', 'assignee_username', or 'clear_assignee' fields",
		})
	}

	if req.Status == "" && req.PullRequestURL == "" && req.IssueURL == "" && req.AssigneeUsername == "" && !req.ClearAssignee {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_fields",
			"message": "at least one of 'status', 'pr_url', 'issue_url', 'assignee_username', or 'clear_assignee' is required",
		})
	}

	// ── Handle assignee update ─────────────────────────────────────────────────
	taskObjID, _ := primitive.ObjectIDFromHex(taskID)

	// Look up the current user for activity logging.
	actor, _ := tc.userRepo.FindByObjectID(c.Context(), userID)
	actorName := ""
	actorAvatar := ""
	if actor != nil {
		actorName = actor.Username
		actorAvatar = actor.AvatarURL
	}

	if req.ClearAssignee {
		// Clear the assignee fields.
		if err := tc.taskRepo.UpdateAssignee(c.Context(), taskObjID, nil, "", ""); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "update_failed",
				"message": err.Error(),
			})
		}
		// Log the clear-assignee activity.
		_ = tc.activityRepo.Log(c.Context(), &domain.ActivityLog{
			RepoID:      task.RepoID,
			RepoName:    task.RepoName,
			ActorID:     userID,
			ActorName:   actorName,
			ActorAvatar: actorAvatar,
			Action:      "assignee_cleared",
			TargetType:  "task",
			TargetID:    taskID,
			TargetLabel: task.Content,
		})
	} else if req.AssigneeUsername != "" {
		// Look up the assignee in the user repository.
		assignee, err := tc.userRepo.FindByUsername(c.Context(), req.AssigneeUsername)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "database_error",
				"message": err.Error(),
			})
		}
		if assignee == nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "user_not_found",
				"message": fmt.Sprintf("user '%s' is not registered in CodeTasker", req.AssigneeUsername),
			})
		}

		// Persist the assignee.
		if err := tc.taskRepo.UpdateAssignee(c.Context(), taskObjID, &assignee.ID, assignee.Username, assignee.AvatarURL); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "update_failed",
				"message": err.Error(),
			})
		}

		// Create an in-app notification for the assignee (skip if self-assigning).
		if assignee.ID != userID {
			_ = tc.notifRepo.Create(c.Context(), &domain.Notification{
				UserID:  assignee.ID,
				Type:    domain.NotifTaskAssigned,
				Title:   "You've been assigned to a task",
				Message: fmt.Sprintf("%s assigned you to: %s", actorName, task.Content),
				Link:    fmt.Sprintf("/repos/%s/tasks", task.RepoName),
			})
		}

		// Log the assignment activity.
		_ = tc.activityRepo.Log(c.Context(), &domain.ActivityLog{
			RepoID:      task.RepoID,
			RepoName:    task.RepoName,
			ActorID:     userID,
			ActorName:   actorName,
			ActorAvatar: actorAvatar,
			Action:      "task_assigned",
			TargetType:  "task",
			TargetID:    taskID,
			TargetLabel: task.Content,
			Meta:        map[string]string{"assignee": req.AssigneeUsername},
		})

		// Send email notification (non-fatal if SMTP is not configured).
		_ = tc.emailService.SendTaskAssigned(
			assignee.Email,
			assignee.Username,
			actorName,
			task.Content,
			task.RepoName,
			"",
		)

		// Send Telegram notification (non-fatal if not configured).
		if assignee.TelegramEnabled && assignee.TelegramBotToken != "" && assignee.TelegramChatID != "" {
			_ = tc.telegramService.SendTaskAssigned(
				c.Context(),
				assignee.TelegramBotToken,
				assignee.TelegramChatID,
				assignee.Username,
				actorName,
				task.Content,
				task.RepoName,
			)
		}
	}

	// ── Handle completion tracking (when status becomes resolved) ──────────────
	if req.Status == domain.TaskStatusResolved && task.Status != domain.TaskStatusResolved {
		completingUser, _ := tc.userRepo.FindByObjectID(c.Context(), userID)
		completingUsername := ""
		completingAvatarURL := ""
		if completingUser != nil {
			completingUsername = completingUser.Username
			completingAvatarURL = completingUser.AvatarURL
		}
		taskObjIDForCompletion, _ := primitive.ObjectIDFromHex(taskID)
		_ = tc.taskRepo.UpdateCompletedBy(c.Context(), taskObjIDForCompletion, completingUsername, completingAvatarURL)

		// Notify maintainer via email.
		if task.MaintainerEmail != "" {
			_ = tc.emailService.SendTaskCompleted(
				task.MaintainerEmail,
				task.MaintainerUsername,
				completingUsername,
				task.Content,
				task.RepoName,
				task.FilePath,
				"",
			)
		}

		// In-app notification for maintainer.
		if task.MaintainerUsername != "" {
			maintainerUser, _ := tc.userRepo.FindByUsername(c.Context(), task.MaintainerUsername)
			if maintainerUser != nil {
				_ = tc.notifRepo.Create(c.Context(), &domain.Notification{
					UserID:  maintainerUser.ID,
					Type:    domain.NotifTaskCompleted,
					Title:   "Task Completed",
					Message: fmt.Sprintf("Task resolved by %s: %s", completingUsername, task.Content),
					Link:    fmt.Sprintf("/repos/%s/tasks", task.RepoName),
				})

				// Send Telegram notification to maintainer
				if maintainerUser.TelegramEnabled && maintainerUser.TelegramBotToken != "" && maintainerUser.TelegramChatID != "" {
					_ = tc.telegramService.SendTaskCompleted(
						c.Context(),
						maintainerUser.TelegramBotToken,
						maintainerUser.TelegramChatID,
						maintainerUser.Username,
						completingUsername,
						task.Content,
						task.RepoName,
						task.FilePath,
					)
				}
			}
		}
	}

	// ── Handle status / PR URL / Issue URL update (if provided) ────────────────
	var updatedTask *domain.Task
	if req.Status != "" || req.PullRequestURL != "" || req.IssueURL != "" {
		updatedTask, err = tc.taskService.UpdateTask(c.Context(), taskID, req.Status, req.PullRequestURL, req.IssueURL)
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
	} else {
		// Re-fetch the task so we return the current state.
		updatedTask, err = tc.taskService.GetTaskByID(c.Context(), taskID)
		if err != nil || updatedTask == nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error":   "not_found",
				"message": "task not found: " + taskID,
			})
		}
	}

	return c.JSON(fiber.Map{
		"task":    updatedTask,
		"message": "task updated successfully",
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

	// Verify collaborator permissions before injecting TODO.
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

	// Look up the actor so we can set creator fields on the task.
	actor, _ := tc.userRepo.FindByObjectID(c.Context(), userID)

	// Upsert the new task in MongoDB so it appears immediately without waiting
	// for the webhook to fire and process the PR merge.
	taskType := req.Type
	if taskType == "" {
		taskType = "TODO"
	}

	// Resolve maintainer via CODEOWNERS before building the task struct.
	maintainerUsername, maintainerEmail := tc.codeOwnerService.ResolveMaintainer(
		c.Context(), userID, req.RepoOwner, req.RepoName, req.FilePath,
	)

	// Build task with creator and maintainer fields already populated so they
	// are written into $setOnInsert on first insert.
	createdByUsername := ""
	createdByAvatarURL := ""
	if actor != nil {
		createdByUsername = actor.Username
		createdByAvatarURL = actor.AvatarURL
	}

	task := &domain.Task{
		RepoID:             synced.RepoID,
		RepoName:           req.RepoOwner + "/" + req.RepoName,
		FilePath:           req.FilePath,
		LineNumber:         req.LineNumber,
		Content:            req.Description,
		Type:               taskType,
		Status:             domain.TaskStatusOpen,
		IssueURL:           req.IssueURL,
		CreatedByUsername:  createdByUsername,
		CreatedByAvatarURL: createdByAvatarURL,
		MaintainerUsername: maintainerUsername,
		MaintainerEmail:    maintainerEmail,
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

// ListComments returns all comments for a task.
// Only repository owners, maintainers, developers, and viewers may view comments.
//
// Route: GET /api/tasks/:id/comments
func (tc *TaskController) ListComments(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	taskIDStr := c.Params("id")
	taskObjID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "task ID is not a valid ObjectID",
		})
	}

	task, err := tc.taskService.GetTaskByID(c.Context(), taskIDStr)
	if err != nil || task == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "task not found",
		})
	}

	// Verify collaborator permissions.
	synced, err := tc.syncedRepoRepo.FindByRepoIDOnly(c.Context(), task.RepoID)
	if err != nil || synced == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Repository not found or access denied",
		})
	}
	if synced.UserID != userID {
		collab, _ := tc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "You do not have permissions to view comments in this repository",
			})
		}
	}

	comments, err := tc.commentRepo.FindByTask(c.Context(), taskObjID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "fetch_comments_failed",
			"message": err.Error(),
		})
	}
	return c.JSON(fiber.Map{"comments": comments, "count": len(comments)})
}

// AddComment adds a new comment to a task. It also creates a notification for
// the task's assignee (if different from the commenter) and logs the activity.
//
// Route: POST /api/tasks/:id/comments
func (tc *TaskController) AddComment(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	taskIDStr := c.Params("id")
	taskObjID, err := primitive.ObjectIDFromHex(taskIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "task ID is not a valid ObjectID",
		})
	}

	// Fetch task to validate it exists and to use its metadata.
	task, err := tc.taskService.GetTaskByID(c.Context(), taskIDStr)
	if err != nil || task == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "task not found",
		})
	}

	// Verify collaborator permissions.
	synced, err := tc.syncedRepoRepo.FindByRepoIDOnly(c.Context(), task.RepoID)
	if err != nil || synced == nil {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Repository not found or access denied",
		})
	}
	if synced.UserID != userID {
		collab, _ := tc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "You do not have permissions to comment on tasks in this repository",
			})
		}
	}

	// Parse comment body.
	type addCommentRequest struct {
		Content string `json:"content"`
	}
	var req addCommentRequest
	if err := c.BodyParser(&req); err != nil || req.Content == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_body",
			"message": "field 'content' is required",
		})
	}

	// Look up the commenter's profile.
	commenter, err := tc.userRepo.FindByObjectID(c.Context(), userID)
	if err != nil || commenter == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "user not found"})
	}

	// Persist the comment.
	comment := &domain.Comment{
		TaskID:    taskObjID,
		UserID:    userID,
		Username:  commenter.Username,
		AvatarURL: commenter.AvatarURL,
		Content:   req.Content,
	}
	if err := tc.commentRepo.Create(c.Context(), comment); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_comment_failed",
			"message": err.Error(),
		})
	}

	// Notify the task's assignee if they are different from the commenter.
	if task.AssigneeID != nil && *task.AssigneeID != userID {
		_ = tc.notifRepo.Create(c.Context(), &domain.Notification{
			UserID:  *task.AssigneeID,
			Type:    domain.NotifCommentAdded,
			Title:   "New comment on your task",
			Message: fmt.Sprintf("%s commented: %s", commenter.Username, req.Content),
			Link:    fmt.Sprintf("/repos/%s/tasks", task.RepoName),
		})

		// Send email notification to the assignee
		if assignee, err := tc.userRepo.FindByObjectID(c.Context(), *task.AssigneeID); err == nil && assignee != nil {
			_ = tc.emailService.SendCommentNotification(
				assignee.Email,
				assignee.Username,
				commenter.Username,
				task.Content,
				req.Content,
				task.RepoName,
				"",
			)

			// Send Telegram notification to the assignee
			if assignee.TelegramEnabled && assignee.TelegramBotToken != "" && assignee.TelegramChatID != "" {
				_ = tc.telegramService.SendCommentNotification(
					c.Context(),
					assignee.TelegramBotToken,
					assignee.TelegramChatID,
					assignee.Username,
					commenter.Username,
					task.Content,
					req.Content,
					task.RepoName,
				)
			}
		}
	}

	// Log the comment activity.
	_ = tc.activityRepo.Log(c.Context(), &domain.ActivityLog{
		RepoID:      task.RepoID,
		RepoName:    task.RepoName,
		ActorID:     userID,
		ActorName:   commenter.Username,
		ActorAvatar: commenter.AvatarURL,
		Action:      "comment_added",
		TargetType:  "task",
		TargetID:    taskIDStr,
		TargetLabel: task.Content,
		Meta:        map[string]string{"comment_id": comment.ID.Hex()},
	})

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"comment": comment,
		"message": "comment added successfully",
	})
}

// DeleteComment removes a comment from a task. Only the comment author or a
// repo owner/maintainer may delete a comment.
//
// Route: DELETE /api/tasks/:id/comments/:commentId
func (tc *TaskController) DeleteComment(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	commentIDStr := c.Params("commentId")
	commentObjID, err := primitive.ObjectIDFromHex(commentIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "comment ID is not a valid ObjectID",
		})
	}

	// Fetch the comment to verify ownership.
	comment, err := tc.commentRepo.FindByID(c.Context(), commentObjID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "fetch_failed",
			"message": err.Error(),
		})
	}
	if comment == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "comment not found",
		})
	}

	// Allow if the user is the comment author.
	if comment.UserID != userID {
		// Otherwise check for owner/maintainer role on the repo.
		task, _ := tc.taskService.GetTaskByID(c.Context(), c.Params("id"))
		authorized := false
		if task != nil {
			synced, _ := tc.syncedRepoRepo.FindByRepoIDOnly(c.Context(), task.RepoID)
			if synced != nil {
				if synced.UserID == userID {
					authorized = true
				} else {
					collab, _ := tc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
					if collab != nil && (collab.Role == domain.RoleOwner || collab.Role == domain.RoleMaintainer) {
						authorized = true
					}
				}
			}
		}
		if !authorized {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "You are not allowed to delete this comment",
			})
		}
	}

	if err := tc.commentRepo.Delete(c.Context(), commentObjID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "delete_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{"message": "comment deleted successfully"})
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
