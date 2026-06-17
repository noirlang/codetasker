// Package controller implements the HTTP handler layer of CodeTasker.
// repo_controller.go handles repository listing and file tree/content
// endpoints. All routes require a valid JWT (Protected middleware).
package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/codetasker/backend/internal/config"
	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/middleware"
	"github.com/codetasker/backend/internal/repository"
	"github.com/codetasker/backend/internal/service"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RepoController handles repository-related HTTP endpoints, delegating all
// GitHub API interactions to GithubService.
type RepoController struct {
	cfg              *config.Config
	githubService    *service.GithubService
	taskService      *service.TaskService
	syncedRepoRepo   *repository.SyncedRepository
	collaboratorRepo *repository.CollaboratorRepository
	userRepo         *repository.UserRepository
	activityRepo     *repository.ActivityRepository
	taskRepo         *repository.TaskRepository
}

// NewRepoController constructs a RepoController with its dependencies.
func NewRepoController(
	cfg *config.Config,
	githubService *service.GithubService,
	taskService *service.TaskService,
	syncedRepoRepo *repository.SyncedRepository,
	collaboratorRepo *repository.CollaboratorRepository,
	userRepo *repository.UserRepository,
	activityRepo *repository.ActivityRepository,
	taskRepo *repository.TaskRepository,
) *RepoController {
	return &RepoController{
		cfg:              cfg,
		githubService:    githubService,
		taskService:      taskService,
		syncedRepoRepo:   syncedRepoRepo,
		collaboratorRepo: collaboratorRepo,
		userRepo:         userRepo,
		activityRepo:     activityRepo,
		taskRepo:         taskRepo,
	}
}

// RegisterRoutes mounts all repo routes onto the provided Fiber router group.
func (rc *RepoController) RegisterRoutes(group fiber.Router) {
	group.Get("/repos", rc.ListRepos)
	group.Get("/orgs", rc.ListOrgs)
	group.Get("/orgs/:org/repos", rc.ListOrgRepos)
	group.Get("/repos/:owner/:repo/tree", rc.GetTree)
	group.Get("/repos/:owner/:repo/contents", rc.GetContents)
	group.Put("/repos/:owner/:repo/contents", rc.UpdateContents)
	group.Post("/repos/:owner/:repo/webhook", rc.CreateWebhook)
	group.Post("/repos/:owner/:repo/sync", rc.SyncRepoTasks)
	group.Get("/repos/:owner/:repo/commits", rc.GetCommits)
	group.Get("/repos/:owner/:repo/actions/workflows", rc.GetActionWorkflows)
	group.Get("/repos/:owner/:repo/actions/runs", rc.GetActionRuns)
	group.Get("/repos/:owner/:repo/pulls", rc.GetPulls)
	group.Post("/repos/:owner/:repo/merge", rc.MergeBranch)

	// Collaborators management
	group.Get("/repos/:owner/:repo/collaborators", rc.GetCollaborators)
	group.Post("/repos/:owner/:repo/collaborators", rc.AddCollaborator)
	group.Patch("/repos/:owner/:repo/collaborators/:id", rc.UpdateCollaboratorRole)
	group.Delete("/repos/:owner/:repo/collaborators/:id", rc.RemoveCollaborator)

	// Issues
	group.Get("/repos/:owner/:repo/issues", rc.ListIssues)
	group.Post("/repos/:owner/:repo/issues", rc.CreateIssue)
	group.Patch("/repos/:owner/:repo/issues/:number", rc.UpdateIssue)

	// Branches
	group.Get("/repos/:owner/:repo/branches", rc.ListBranches)
	group.Post("/repos/:owner/:repo/branches", rc.CreateBranch)
	group.Delete("/repos/:owner/:repo/branches/:branch", rc.DeleteBranch)

	// Commit diff
	group.Get("/repos/:owner/:repo/commits/:sha", rc.GetCommitDiff)

	// Repository statistics
	group.Get("/repos/:owner/:repo/stats", rc.GetRepoStats)

	// Activity feed
	group.Get("/repos/:owner/:repo/activity", rc.GetActivity)
}

// ListRepos returns the authenticated user's GitHub repositories.
// The response includes: name, full_name, description, private flag,
// updated_at timestamp, primary language, and stargazers count.
//
// Route: GET /api/repos
func (rc *RepoController) ListRepos(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	repos, err := rc.githubService.ListRepos(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_repos_failed",
			"message": err.Error(),
		})
	}

	// Fetch synced repositories to determine active sync statuses
	syncedRepos, err := rc.syncedRepoRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		// Log the error but keep the list operation functional
		syncedRepos = []domain.SyncedRepo{}
	}

	// Fetch collaborations where this user is added
	collaborations, err := rc.collaboratorRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		collaborations = []domain.Collaborator{}
	}

	// Map to a lookup set of repoIDs
	syncedSet := make(map[int64]bool)
	for _, sr := range syncedRepos {
		syncedSet[sr.RepoID] = true
	}
	for _, col := range collaborations {
		syncedSet[col.RepoID] = true
	}

	// Map to a lean response struct — expose only the fields the frontend needs.
	type repoResponse struct {
		ID              int64    `json:"id"`
		Name            string   `json:"name"`
		FullName        string   `json:"full_name"`
		Description     string   `json:"description"`
		Private         bool     `json:"private"`
		UpdatedAt       string   `json:"updated_at"`
		Language        string   `json:"language"`
		StargazersCount int      `json:"stargazers_count"`
		HTMLURL         string   `json:"html_url"`
		IsSynced        bool     `json:"is_synced"`
		Topics          []string `json:"topics"`
	}

	includedSet := make(map[int64]bool)
	response := make([]repoResponse, 0, len(repos))
	for _, r := range repos {
		repoID := r.GetID()
		updatedAt := ""
		if r.UpdatedAt != nil {
			updatedAt = r.UpdatedAt.Format("2006-01-02T15:04:05Z")
		}

		description := ""
		if r.Description != nil {
			description = *r.Description
		}

		language := ""
		if r.Language != nil {
			language = *r.Language
		}

		response = append(response, repoResponse{
			ID:              repoID,
			Name:            r.GetName(),
			FullName:        r.GetFullName(),
			Description:     description,
			Private:         r.GetPrivate(),
			UpdatedAt:       updatedAt,
			Language:        language,
			StargazersCount: r.GetStargazersCount(),
			HTMLURL:         r.GetHTMLURL(),
			IsSynced:        syncedSet[repoID],
			Topics:          r.Topics,
		})
		includedSet[repoID] = true
	}

	// Append collaborator repositories that are not in the user's personal GitHub list
	for _, col := range collaborations {
		if !includedSet[col.RepoID] {
			synced, err := rc.syncedRepoRepo.FindByRepoIDOnly(c.Context(), col.RepoID)
			if err == nil && synced != nil {
				shortName := synced.RepoName
				if slashIdx := strings.Index(synced.RepoName, "/"); slashIdx != -1 {
					shortName = synced.RepoName[slashIdx+1:]
				}

				response = append(response, repoResponse{
					ID:              synced.RepoID,
					Name:            shortName,
					FullName:        synced.RepoName,
					Description:     "Collaborator repository",
					Private:         true,
					UpdatedAt:       synced.CreatedAt.Format("2006-01-02T15:04:05Z"),
					Language:        "Go/JS/TS",
					StargazersCount: 0,
					HTMLURL:         fmt.Sprintf("https://github.com/%s", synced.RepoName),
					IsSynced:        true,
					Topics:          []string{"collaboration"},
				})
				includedSet[col.RepoID] = true
			}
		}
	}

	return c.JSON(fiber.Map{
		"repos": response,
		"count": len(response),
	})
}

// GetTree returns the full recursive file tree for a repository branch.
// The optional `?branch=` query parameter selects a specific branch;
// if omitted the repository's default branch is used.
//
// Route: GET /api/repos/:owner/:repo/tree
func (rc *RepoController) GetTree(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	branch := c.Query("branch", "HEAD")

	tree, err := rc.githubService.GetTree(c.Context(), rc.resolveTargetUserID(c.Context(), userID, owner, repo), owner, repo, branch)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_tree_failed",
			"message": err.Error(),
		})
	}

	// Map tree entries to a lean response format.
	type treeEntry struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
		Type string `json:"type"`
		SHA  string `json:"sha"`
		Size int    `json:"size,omitempty"`
	}

	entries := make([]treeEntry, 0, len(tree.Entries))
	for _, e := range tree.Entries {
		entries = append(entries, treeEntry{
			Path: e.GetPath(),
			Mode: e.GetMode(),
			Type: e.GetType(),
			SHA:  e.GetSHA(),
			Size: e.GetSize(),
		})
	}

	return c.JSON(fiber.Map{
		"sha":       tree.GetSHA(),
		"truncated": tree.GetTruncated(),
		"entries":   entries,
		"count":     len(entries),
	})
}

// GetContents fetches and returns the decoded text content of a single file.
// Requires `?path=` query parameter. Optional `?ref=` selects a commit SHA,
// branch name, or tag; defaults to the repo's default branch if omitted.
//
// Route: GET /api/repos/:owner/:repo/contents
func (rc *RepoController) GetContents(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	path := c.Query("path")
	ref := c.Query("ref", "")

	if path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "query parameter 'path' is required",
		})
	}

	content, err := rc.githubService.GetContents(c.Context(), rc.resolveTargetUserID(c.Context(), userID, owner, repo), owner, repo, path, ref)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_contents_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"path":    path,
		"ref":     ref,
		"content": content,
	})
}

// CreateWebhook registers a webhook on GitHub for the specified repository.
// The payload URL points to the backend webhook handler.
//
// Route: POST /api/repos/:owner/:repo/webhook
func (rc *RepoController) CreateWebhook(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	// Use configured webhook proxy URL, fallback to request BaseURL
	payloadURL := c.BaseURL() + "/api/webhooks/github"
	if rc.cfg.WebhookProxyURL != "" {
		payloadURL = rc.cfg.WebhookProxyURL
	}

	type requestBody struct {
		PayloadURL string `json:"payload_url"`
	}
	var body requestBody
	if err := c.BodyParser(&body); err == nil && body.PayloadURL != "" {
		payloadURL = body.PayloadURL
	}

	repoID, err := rc.githubService.CreateWebhook(c.Context(), userID, owner, repo, payloadURL)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "create_webhook_failed",
			"message": err.Error(),
		})
	}

	synced := &domain.SyncedRepo{
		RepoID:   repoID,
		RepoName: owner + "/" + repo,
		Owner:    owner,
		UserID:   userID,
	}

	_ = rc.syncedRepoRepo.Create(c.Context(), synced)

	// Add the syncing user as the owner collaborator
	user, err := rc.userRepo.FindByObjectID(c.Context(), userID)
	if err == nil && user != nil {
		collab := &domain.Collaborator{
			RepoID:    repoID,
			UserID:    userID,
			Username:  user.Username,
			AvatarURL: user.AvatarURL,
			Role:      domain.RoleOwner,
		}
		_ = rc.collaboratorRepo.Create(c.Context(), collab)
	}

	return c.JSON(fiber.Map{
		"message":     "webhook created successfully",
		"payload_url": payloadURL,
		"repo_id":     repoID,
	})
}

// UpdateContents commits a file modification back to GitHub.
//
// Route: PUT /api/repos/:owner/:repo/contents
func (rc *RepoController) UpdateContents(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	type requestBody struct {
		Path      string   `json:"path"`
		Content   string   `json:"content"`
		Branch    string   `json:"branch"`
		Message   string   `json:"message"`
		CoAuthors []string `json:"co_authors"`
	}

	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if body.Path == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "field 'path' is required",
		})
	}

	if body.Message == "" {
		body.Message = "Update " + body.Path + " via CodeTasker"
	}

	if body.Branch == "" {
		body.Branch = "main"
	}

	// Verify collaborator permissions
	syncedRepo, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err == nil && syncedRepo != nil {
		if syncedRepo.UserID != userID {
			collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, syncedRepo.RepoID)
			if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "forbidden",
					"message": "You do not have write access to this repository in CodeTasker",
				})
			}
		}
	}

	commitSHA, err := rc.githubService.UpdateFile(c.Context(), userID, owner, repo, body.Path, body.Content, body.Branch, body.Message, body.CoAuthors)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "update_contents_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":    "file updated successfully",
		"commit_sha": commitSHA,
	})
}

// GetCommits returns the commits for a repository on a given branch.
// Route: GET /api/repos/:owner/:repo/commits
func (rc *RepoController) GetCommits(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	branch := c.Query("branch") // optional, defaults to default branch / HEAD
	page := parsePositiveQueryInt(c.Query("page"), 1)
	perPage := parsePositiveQueryInt(c.Query("per_page"), 50)
	if perPage > 100 {
		perPage = 100
	}

	result, err := rc.githubService.ListCommitsPage(c.Context(), userID, owner, repo, branch, page, perPage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_commits_failed",
			"message": err.Error(),
		})
	}
	commits := result.Commits

	type commitResponse struct {
		SHA                string                          `json:"sha"`
		Message            string                          `json:"message"`
		Author             string                          `json:"author"`
		AuthorEmail        string                          `json:"author_email"`
		AvatarURL          string                          `json:"avatar_url"`
		Committer          string                          `json:"committer"`
		CommitterEmail     string                          `json:"committer_email"`
		CommitterAvatarURL string                          `json:"committer_avatar_url"`
		Date               string                          `json:"date"`
		HTMLURL            string                          `json:"html_url"`
		Verified           bool                            `json:"verified"`
		VerificationReason string                          `json:"verification_reason"`
		CheckState         string                          `json:"check_state"`
		CheckTotal         int                             `json:"check_total"`
		CheckRuns          []service.CommitCheckRunSummary `json:"check_runs"`
		Statuses           []service.CommitStatusSummary   `json:"statuses"`
		CheckError         string                          `json:"check_error,omitempty"`
	}

	shas := make([]string, 0, len(commits))
	for _, commit := range commits {
		if commit == nil {
			continue
		}
		if sha := commit.GetSHA(); sha != "" {
			shas = append(shas, sha)
		}
	}

	healthBySHA, healthErr := rc.githubService.GetCommitHealthSummaries(c.Context(), userID, owner, repo, shas)

	response := make([]commitResponse, 0, len(commits))
	for _, commit := range commits {
		if commit == nil {
			continue
		}

		sha := commit.GetSHA()
		msg := ""
		author := ""
		authorEmail := ""
		avatar := ""
		committer := ""
		committerEmail := ""
		committerAvatar := ""
		date := ""
		verified := false
		verificationReason := ""

		if commit.Commit != nil {
			msg = commit.Commit.GetMessage()
			if commit.Commit.Author != nil {
				author = commit.Commit.Author.GetName()
				authorEmail = commit.Commit.Author.GetEmail()
				if commit.Commit.Author.Date != nil {
					date = commit.Commit.Author.Date.Format(time.RFC3339)
				}
			}
			if commit.Commit.Committer != nil {
				committer = commit.Commit.Committer.GetName()
				committerEmail = commit.Commit.Committer.GetEmail()
			}
			if verification := commit.Commit.GetVerification(); verification != nil {
				verified = verification.GetVerified()
				verificationReason = verification.GetReason()
			}
		}

		if commit.Author != nil {
			avatar = commit.Author.GetAvatarURL()
			if author == "" {
				author = commit.Author.GetLogin()
			}
		}
		if commit.Committer != nil {
			committerAvatar = commit.Committer.GetAvatarURL()
			if committer == "" {
				committer = commit.Committer.GetLogin()
			}
		}

		health := service.CommitHealthSummary{
			State:     "unknown",
			CheckRuns: []service.CommitCheckRunSummary{},
			Statuses:  []service.CommitStatusSummary{},
		}
		if healthErr != nil {
			health.Error = healthErr.Error()
		} else if healthBySHA != nil {
			if summary, ok := healthBySHA[sha]; ok {
				health = summary
			}
		}

		response = append(response, commitResponse{
			SHA:                sha,
			Message:            msg,
			Author:             author,
			AuthorEmail:        authorEmail,
			AvatarURL:          avatar,
			Committer:          committer,
			CommitterEmail:     committerEmail,
			CommitterAvatarURL: committerAvatar,
			Date:               date,
			HTMLURL:            commit.GetHTMLURL(),
			Verified:           verified,
			VerificationReason: verificationReason,
			CheckState:         health.State,
			CheckTotal:         health.Total,
			CheckRuns:          health.CheckRuns,
			Statuses:           health.Statuses,
			CheckError:         health.Error,
		})
	}

	return c.JSON(fiber.Map{
		"commits":   response,
		"count":     len(response),
		"page":      page,
		"per_page":  perPage,
		"next_page": result.NextPage,
	})
}

func parsePositiveQueryInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

// GetActionWorkflows returns the GitHub Actions workflows configured for a repository.
// Route: GET /api/repos/:owner/:repo/actions/workflows
func (rc *RepoController) GetActionWorkflows(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	workflows, err := rc.githubService.ListActionWorkflows(c.Context(), userID, owner, repo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_action_workflows_failed",
			"message": err.Error(),
		})
	}

	type workflowResponse struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		Path      string `json:"path"`
		State     string `json:"state"`
		HTMLURL   string `json:"html_url"`
		BadgeURL  string `json:"badge_url"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
	}

	response := make([]workflowResponse, 0, len(workflows.Workflows))
	for _, workflow := range workflows.Workflows {
		if workflow == nil {
			continue
		}

		createdAt := ""
		if workflow.CreatedAt != nil {
			createdAt = workflow.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if workflow.UpdatedAt != nil {
			updatedAt = workflow.UpdatedAt.Format(time.RFC3339)
		}

		response = append(response, workflowResponse{
			ID:        workflow.GetID(),
			Name:      workflow.GetName(),
			Path:      workflow.GetPath(),
			State:     workflow.GetState(),
			HTMLURL:   workflow.GetHTMLURL(),
			BadgeURL:  workflow.GetBadgeURL(),
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"workflows": response,
		"count":     len(response),
	})
}

// GetActionRuns returns recent GitHub Actions workflow runs for a repository.
// Route: GET /api/repos/:owner/:repo/actions/runs
func (rc *RepoController) GetActionRuns(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	branch := c.Query("branch")
	status := c.Query("status")

	runs, err := rc.githubService.ListActionRuns(c.Context(), userID, owner, repo, branch, status)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_action_runs_failed",
			"message": err.Error(),
		})
	}

	type actionRunResponse struct {
		ID             int64  `json:"id"`
		Name           string `json:"name"`
		DisplayTitle   string `json:"display_title"`
		Status         string `json:"status"`
		Conclusion     string `json:"conclusion"`
		WorkflowID     int64  `json:"workflow_id"`
		RunNumber      int    `json:"run_number"`
		RunAttempt     int    `json:"run_attempt"`
		Event          string `json:"event"`
		HeadBranch     string `json:"head_branch"`
		HeadSHA        string `json:"head_sha"`
		HTMLURL        string `json:"html_url"`
		Actor          string `json:"actor"`
		ActorAvatarURL string `json:"actor_avatar_url"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
		RunStartedAt   string `json:"run_started_at"`
	}

	response := make([]actionRunResponse, 0, len(runs.WorkflowRuns))
	for _, run := range runs.WorkflowRuns {
		if run == nil {
			continue
		}

		actor := ""
		actorAvatar := ""
		if run.Actor != nil {
			actor = run.Actor.GetLogin()
			actorAvatar = run.Actor.GetAvatarURL()
		}

		createdAt := ""
		if run.CreatedAt != nil {
			createdAt = run.CreatedAt.Format(time.RFC3339)
		}

		updatedAt := ""
		if run.UpdatedAt != nil {
			updatedAt = run.UpdatedAt.Format(time.RFC3339)
		}

		runStartedAt := ""
		if run.RunStartedAt != nil {
			runStartedAt = run.RunStartedAt.Format(time.RFC3339)
		}

		response = append(response, actionRunResponse{
			ID:             run.GetID(),
			Name:           run.GetName(),
			DisplayTitle:   run.GetDisplayTitle(),
			Status:         run.GetStatus(),
			Conclusion:     run.GetConclusion(),
			WorkflowID:     run.GetWorkflowID(),
			RunNumber:      run.GetRunNumber(),
			RunAttempt:     run.GetRunAttempt(),
			Event:          run.GetEvent(),
			HeadBranch:     run.GetHeadBranch(),
			HeadSHA:        run.GetHeadSHA(),
			HTMLURL:        run.GetHTMLURL(),
			Actor:          actor,
			ActorAvatarURL: actorAvatar,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			RunStartedAt:   runStartedAt,
		})
	}

	return c.JSON(fiber.Map{
		"runs":        response,
		"count":       len(response),
		"total_count": runs.GetTotalCount(),
	})
}

// GetPulls returns the pull requests for a repository.
// Route: GET /api/repos/:owner/:repo/pulls
func (rc *RepoController) GetPulls(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	state := c.Query("state", "open")

	pulls, err := rc.githubService.ListPullRequests(c.Context(), userID, owner, repo, state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_pulls_failed",
			"message": err.Error(),
		})
	}

	type pullResponse struct {
		ID        int64  `json:"id"`
		Number    int    `json:"number"`
		Title     string `json:"title"`
		State     string `json:"state"`
		HTMLURL   string `json:"html_url"`
		Branch    string `json:"branch"`
		Base      string `json:"base"`
		Creator   string `json:"creator"`
		AvatarURL string `json:"avatar_url"`
		CreatedAt string `json:"created_at"`
	}

	response := make([]pullResponse, 0, len(pulls))
	for _, pr := range pulls {
		creator := ""
		avatar := ""
		if pr.User != nil {
			creator = pr.User.GetLogin()
			avatar = pr.User.GetAvatarURL()
		}

		createdAt := ""
		if pr.CreatedAt != nil {
			createdAt = pr.CreatedAt.Format(time.RFC3339)
		}

		branch := ""
		if pr.Head != nil {
			branch = pr.Head.GetRef()
		}

		base := ""
		if pr.Base != nil {
			base = pr.Base.GetRef()
		}

		response = append(response, pullResponse{
			ID:        pr.GetID(),
			Number:    pr.GetNumber(),
			Title:     pr.GetTitle(),
			State:     pr.GetState(),
			HTMLURL:   pr.GetHTMLURL(),
			Branch:    branch,
			Base:      base,
			Creator:   creator,
			AvatarURL: avatar,
			CreatedAt: createdAt,
		})
	}

	return c.JSON(fiber.Map{
		"pulls": response,
		"count": len(response),
	})
}

// MergeBranch merges head branch into base branch.
// Route: POST /api/repos/:owner/:repo/merge
func (rc *RepoController) MergeBranch(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	type requestBody struct {
		Base          string `json:"base"`
		Head          string `json:"head"`
		CommitMessage string `json:"commit_message"`
	}

	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if body.Base == "" || body.Head == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "fields 'base' and 'head' are required",
		})
	}

	// Verify collaborator permissions
	syncedRepo, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err == nil && syncedRepo != nil {
		if syncedRepo.UserID != userID {
			collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, syncedRepo.RepoID)
			if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
				return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
					"error":   "forbidden",
					"message": "You do not have write access to merge branches in this repository",
				})
			}
		}
	}

	resp, err := rc.githubService.MergeBranch(c.Context(), userID, owner, repo, body.Base, body.Head, body.CommitMessage)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "merge_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message":    "branch merged successfully",
		"commit_sha": resp.GetSHA(),
	})
}

// GetCollaborators retrieves the collaborator list for a repository.
//
// Route: GET /api/repos/:owner/:repo/collaborators
func (rc *RepoController) GetCollaborators(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	fullName := owner + "/" + repo

	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), fullName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "Repository is not synced in CodeTasker",
		})
	}

	// Verify the current user is a collaborator or owner
	isAuthorized := false
	if synced.UserID == userID {
		isAuthorized = true
	} else {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab != nil {
			isAuthorized = true
		}
	}

	if !isAuthorized {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "You do not have access to this repository's collaborators list",
		})
	}

	collaborators, err := rc.collaboratorRepo.FindByRepoID(c.Context(), synced.RepoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_fetch_collaborators",
			"message": err.Error(),
		})
	}

	// Check if owner has a collaborator record
	hasOwner := false
	for _, col := range collaborators {
		if col.Role == domain.RoleOwner {
			hasOwner = true
			break
		}
	}

	if !hasOwner {
		ownerUser, err := rc.userRepo.FindByObjectID(c.Context(), synced.UserID)
		if err == nil && ownerUser != nil {
			ownerCollab := domain.Collaborator{
				ID:        primitive.NewObjectID(),
				RepoID:    synced.RepoID,
				UserID:    synced.UserID,
				Username:  ownerUser.Username,
				AvatarURL: ownerUser.AvatarURL,
				Role:      domain.RoleOwner,
			}
			_ = rc.collaboratorRepo.Create(c.Context(), &ownerCollab)
			collaborators = append([]domain.Collaborator{ownerCollab}, collaborators...)
		}
	}

	return c.JSON(fiber.Map{
		"collaborators": collaborators,
		"count":         len(collaborators),
	})
}

// AddCollaborator adds a new collaborator to the repository.
//
// Route: POST /api/repos/:owner/:repo/collaborators
func (rc *RepoController) AddCollaborator(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	fullName := owner + "/" + repo

	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), fullName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "Repository is not synced in CodeTasker",
		})
	}

	hasWriteAccess := false
	if synced.UserID == userID {
		hasWriteAccess = true
	} else {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab != nil && (collab.Role == domain.RoleOwner || collab.Role == domain.RoleMaintainer) {
			hasWriteAccess = true
		}
	}

	if !hasWriteAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Only repository owners and maintainers can add collaborators",
		})
	}

	type requestBody struct {
		Username string          `json:"username"`
		Role     domain.RepoRole `json:"role"`
	}

	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if body.Username == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "field 'username' is required",
		})
	}

	if body.Role != domain.RoleMaintainer && body.Role != domain.RoleDeveloper && body.Role != domain.RoleViewer {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_role",
			"message": "role must be 'maintainer', 'developer' or 'viewer'",
		})
	}

	invitee, err := rc.userRepo.FindByUsername(c.Context(), body.Username)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if invitee == nil {
		fmt.Printf("[DEBUG] Collaborator invitee '%s' not found in database, fetching from GitHub\n", body.Username)
		ghUser, err := rc.githubService.GetUserByUsername(c.Context(), userID, body.Username)
		if err != nil {
			fmt.Printf("[DEBUG] GitHub lookup failed for user '%s': %v\n", body.Username, err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error":   "user_not_found",
				"message": fmt.Sprintf("The user '%s' must log in to CodeTasker with GitHub before they can be added as a collaborator", body.Username),
			})
		}

		newUser := &domain.User{
			GithubID:  ghUser.GetID(),
			Username:  ghUser.GetLogin(),
			AvatarURL: ghUser.GetAvatarURL(),
		}

		err = rc.userRepo.Upsert(c.Context(), newUser)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "failed_to_create_user",
				"message": err.Error(),
			})
		}

		invitee, err = rc.userRepo.FindByGithubID(c.Context(), newUser.GithubID)
		if err != nil || invitee == nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error":   "failed_to_retrieve_created_user",
				"message": "User was created but could not be retrieved",
			})
		}
	}

	existing, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), invitee.ID, synced.RepoID)
	if existing != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "already_collaborator",
			"message": "This user is already a collaborator on this repository",
		})
	}

	collab := &domain.Collaborator{
		RepoID:    synced.RepoID,
		UserID:    invitee.ID,
		Username:  invitee.Username,
		AvatarURL: invitee.AvatarURL,
		Role:      body.Role,
	}

	if err := rc.collaboratorRepo.Create(c.Context(), collab); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_create_collaborator",
			"message": err.Error(),
		})
	}

	// Check if the user is a collaborator on the actual GitHub repository
	isGHCollab, err := rc.githubService.IsCollaborator(c.Context(), userID, owner, repo, body.Username)
	var warning string
	if err != nil {
		fmt.Printf("[DEBUG] Failed to check GitHub collaborator status for user '%s': %v\n", body.Username, err)
	} else if !isGHCollab {
		warning = fmt.Sprintf("Note: '%s' is not listed as a collaborator on the GitHub repository. They will not be able to view tasks or push commits until you invite them on GitHub.", body.Username)
	}

	return c.JSON(fiber.Map{
		"message":      "collaborator added successfully",
		"collaborator": collab,
		"warning":      warning,
	})
}

// UpdateCollaboratorRole updates a collaborator's role.
//
// Route: PATCH /api/repos/:owner/:repo/collaborators/:id
func (rc *RepoController) UpdateCollaboratorRole(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	collabIDStr := c.Params("id")
	fullName := owner + "/" + repo

	collabID, err := primitive.ObjectIDFromHex(collabIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "collaborator ID is not a valid ObjectID",
		})
	}

	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), fullName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "Repository is not synced in CodeTasker",
		})
	}

	currentUserRole := domain.RepoRole("")
	if synced.UserID == userID {
		currentUserRole = domain.RoleOwner
	} else {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab != nil {
			currentUserRole = collab.Role
		}
	}

	if currentUserRole != domain.RoleOwner && currentUserRole != domain.RoleMaintainer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Only repository owners and maintainers can update collaborator roles",
		})
	}

	type requestBody struct {
		Role domain.RepoRole `json:"role"`
	}

	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_request",
			"message": err.Error(),
		})
	}

	if body.Role != domain.RoleMaintainer && body.Role != domain.RoleDeveloper && body.Role != domain.RoleViewer {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_role",
			"message": "role must be 'maintainer', 'developer' or 'viewer'",
		})
	}

	collaborators, err := rc.collaboratorRepo.FindByRepoID(c.Context(), synced.RepoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}

	var targetCollab *domain.Collaborator
	for i := range collaborators {
		if collaborators[i].ID == collabID {
			targetCollab = &collaborators[i]
			break
		}
	}

	if targetCollab == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "collaborator_not_found",
			"message": "Collaborator record not found",
		})
	}

	if targetCollab.Role == domain.RoleOwner || targetCollab.UserID == synced.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Cannot modify the owner's role",
		})
	}

	if currentUserRole == domain.RoleMaintainer && (targetCollab.Role == domain.RoleMaintainer || body.Role == domain.RoleMaintainer) {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Maintainers cannot modify maintainer roles",
		})
	}

	if err := rc.collaboratorRepo.UpdateRole(c.Context(), collabID, body.Role); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_update_role",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "collaborator role updated successfully",
	})
}

// RemoveCollaborator removes a collaborator from a repository.
//
// Route: DELETE /api/repos/:owner/:repo/collaborators/:id
func (rc *RepoController) RemoveCollaborator(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")
	collabIDStr := c.Params("id")
	fullName := owner + "/" + repo

	collabID, err := primitive.ObjectIDFromHex(collabIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "invalid_id",
			"message": "collaborator ID is not a valid ObjectID",
		})
	}

	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), fullName)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "Repository is not synced in CodeTasker",
		})
	}

	currentUserRole := domain.RepoRole("")
	if synced.UserID == userID {
		currentUserRole = domain.RoleOwner
	} else {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab != nil {
			currentUserRole = collab.Role
		}
	}

	if currentUserRole != domain.RoleOwner && currentUserRole != domain.RoleMaintainer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Only repository owners and maintainers can remove collaborators",
		})
	}

	collaborators, err := rc.collaboratorRepo.FindByRepoID(c.Context(), synced.RepoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}

	var targetCollab *domain.Collaborator
	for i := range collaborators {
		if collaborators[i].ID == collabID {
			targetCollab = &collaborators[i]
			break
		}
	}

	if targetCollab == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "collaborator_not_found",
			"message": "Collaborator record not found",
		})
	}

	if targetCollab.Role == domain.RoleOwner || targetCollab.UserID == synced.UserID {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Cannot remove the repository owner",
		})
	}

	if currentUserRole == domain.RoleMaintainer && targetCollab.Role == domain.RoleMaintainer {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error":   "forbidden",
			"message": "Maintainers cannot remove other maintainers",
		})
	}

	if err := rc.collaboratorRepo.Delete(c.Context(), collabID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "failed_to_remove_collaborator",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "collaborator removed successfully",
	})
}

// SyncRepoTasks triggers a manual synchronization of tasks from the repository codebase.
// Route: POST /api/repos/:owner/:repo/sync
func (rc *RepoController) SyncRepoTasks(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	owner := c.Params("owner")
	repo := c.Params("repo")

	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "database_error",
			"message": err.Error(),
		})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error":   "not_found",
			"message": "Repository is not synced in CodeTasker",
		})
	}

	// Verify collaborator permissions
	if synced.UserID != userID {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer && collab.Role != domain.RoleDeveloper) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error":   "forbidden",
				"message": "You do not have permissions to sync tasks in this repository",
			})
		}
	}

	// Trigger tasks sync
	if err := rc.taskService.SyncTasks(c.Context(), userID, synced.RepoID, owner, repo); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "sync_failed",
			"message": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Repository tasks synced successfully",
	})
}

// ListOrgs returns the organizations the authenticated user belongs to.
// Route: GET /api/orgs
func (rc *RepoController) ListOrgs(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	orgs, err := rc.githubService.ListOrgs(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_orgs_failed",
			"message": err.Error(),
		})
	}

	type orgResponse struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
	}

	response := make([]orgResponse, 0, len(orgs))
	for _, o := range orgs {
		response = append(response, orgResponse{
			Login:     o.GetLogin(),
			AvatarURL: o.GetAvatarURL(),
		})
	}

	return c.JSON(fiber.Map{
		"orgs":  response,
		"count": len(response),
	})
}

// ListOrgRepos returns the repositories of a specific organization.
// Route: GET /api/orgs/:org/repos
func (rc *RepoController) ListOrgRepos(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	org := c.Params("org")
	if org == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "missing_parameter",
			"message": "parameter 'org' is required",
		})
	}

	repos, err := rc.githubService.ListOrgRepos(c.Context(), userID, org)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_org_repos_failed",
			"message": err.Error(),
		})
	}

	// Fetch synced repositories to determine active sync statuses
	syncedRepos, err := rc.syncedRepoRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		syncedRepos = []domain.SyncedRepo{}
	}

	// Fetch collaborations where this user is added
	collaborations, err := rc.collaboratorRepo.FindByUserID(c.Context(), userID)
	if err != nil {
		collaborations = []domain.Collaborator{}
	}

	syncedSet := make(map[int64]bool)
	for _, sr := range syncedRepos {
		syncedSet[sr.RepoID] = true
	}
	for _, col := range collaborations {
		syncedSet[col.RepoID] = true
	}

	type repoResponse struct {
		ID              int64    `json:"id"`
		Name            string   `json:"name"`
		FullName        string   `json:"full_name"`
		Description     string   `json:"description"`
		Private         bool     `json:"private"`
		UpdatedAt       string   `json:"updated_at"`
		Language        string   `json:"language"`
		StargazersCount int      `json:"stargazers_count"`
		HTMLURL         string   `json:"html_url"`
		IsSynced        bool     `json:"is_synced"`
		Topics          []string `json:"topics"`
	}

	response := make([]repoResponse, 0, len(repos))
	for _, r := range repos {
		updatedAt := ""
		if r.UpdatedAt != nil {
			updatedAt = r.UpdatedAt.Format("2006-01-02T15:04:05Z")
		}

		description := ""
		if r.Description != nil {
			description = *r.Description
		}

		language := ""
		if r.Language != nil {
			language = *r.Language
		}

		response = append(response, repoResponse{
			ID:              r.GetID(),
			Name:            r.GetName(),
			FullName:        r.GetFullName(),
			Description:     description,
			Private:         r.GetPrivate(),
			UpdatedAt:       updatedAt,
			Language:        language,
			StargazersCount: r.GetStargazersCount(),
			HTMLURL:         r.GetHTMLURL(),
			IsSynced:        syncedSet[r.GetID()],
			Topics:          r.Topics,
		})
	}

	return c.JSON(fiber.Map{
		"repos": response,
		"count": len(response),
	})
}

// ListIssues returns open (or all) GitHub issues for a repository.
// Requires ?state=open|closed|all; defaults to "open".
//
// Route: GET /api/repos/:owner/:repo/issues
func (rc *RepoController) ListIssues(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")
	state := c.Query("state", "open")

	issues, err := rc.githubService.ListIssues(c.Context(), rc.resolveTargetUserID(c.Context(), userID, owner, repo), owner, repo, state)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "list_issues_failed",
			"message": err.Error(),
		})
	}

	type issueResponse struct {
		ID        int64    `json:"id"`
		Number    int      `json:"number"`
		Title     string   `json:"title"`
		State     string   `json:"state"`
		HTMLURL   string   `json:"html_url"`
		Body      string   `json:"body"`
		Labels    []string `json:"labels"`
		Creator   string   `json:"creator"`
		AvatarURL string   `json:"avatar_url"`
		CreatedAt string   `json:"created_at"`
		UpdatedAt string   `json:"updated_at"`
	}

	response := make([]issueResponse, 0, len(issues))
	for _, issue := range issues {
		if issue == nil {
			continue
		}
		creator := ""
		avatar := ""
		if issue.User != nil {
			creator = issue.User.GetLogin()
			avatar = issue.User.GetAvatarURL()
		}
		createdAt := ""
		if issue.CreatedAt != nil {
			createdAt = issue.CreatedAt.Format(time.RFC3339)
		}
		updatedAt := ""
		if issue.UpdatedAt != nil {
			updatedAt = issue.UpdatedAt.Format(time.RFC3339)
		}
		labels := make([]string, 0, len(issue.Labels))
		for _, l := range issue.Labels {
			if l != nil {
				labels = append(labels, l.GetName())
			}
		}
		body := ""
		if issue.Body != nil {
			body = *issue.Body
		}
		response = append(response, issueResponse{
			ID:        issue.GetID(),
			Number:    issue.GetNumber(),
			Title:     issue.GetTitle(),
			State:     issue.GetState(),
			HTMLURL:   issue.GetHTMLURL(),
			Body:      body,
			Labels:    labels,
			Creator:   creator,
			AvatarURL: avatar,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		})
	}

	return c.JSON(fiber.Map{"issues": response, "count": len(response)})
}

// CreateIssue creates a new GitHub issue for a repository.
//
// Route: POST /api/repos/:owner/:repo/issues
func (rc *RepoController) CreateIssue(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")

	type requestBody struct {
		Title  string   `json:"title"`
		Body   string   `json:"body"`
		Labels []string `json:"labels"`
	}
	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": err.Error()})
	}
	if body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_parameter", "message": "field 'title' is required"})
	}

	issue, err := rc.githubService.CreateIssue(c.Context(), userID, owner, repo, body.Title, body.Body, body.Labels)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create_issue_failed", "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"issue": issue})
}

// UpdateIssue opens or closes a GitHub issue.
//
// Route: PATCH /api/repos/:owner/:repo/issues/:number
func (rc *RepoController) UpdateIssue(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")
	numberStr := c.Params("number")
	number, err := strconv.Atoi(numberStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_parameter", "message": "issue number must be an integer"})
	}

	type requestBody struct {
		State string `json:"state"`
	}
	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": err.Error()})
	}
	if body.State != "open" && body.State != "closed" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_parameter", "message": "state must be 'open' or 'closed'"})
	}

	issue, err := rc.githubService.UpdateIssueState(c.Context(), userID, owner, repo, number, body.State)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "update_issue_failed", "message": err.Error()})
	}

	return c.JSON(fiber.Map{"issue": issue})
}

// ListBranches returns all branches for a repository.
//
// Route: GET /api/repos/:owner/:repo/branches
func (rc *RepoController) ListBranches(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")

	branches, err := rc.githubService.ListBranches(c.Context(), rc.resolveTargetUserID(c.Context(), userID, owner, repo), owner, repo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "list_branches_failed", "message": err.Error()})
	}

	type branchResponse struct {
		Name string `json:"name"`
		SHA  string `json:"sha"`
	}
	response := make([]branchResponse, 0, len(branches))
	for _, b := range branches {
		if b == nil {
			continue
		}
		sha := ""
		if b.Commit != nil {
			sha = b.Commit.GetSHA()
		}
		response = append(response, branchResponse{
			Name: b.GetName(),
			SHA:  sha,
		})
	}

	return c.JSON(fiber.Map{"branches": response, "count": len(response)})
}

// CreateBranch creates a new branch from a given SHA.
// Requires owner or maintainer role.
//
// Route: POST /api/repos/:owner/:repo/branches
func (rc *RepoController) CreateBranch(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Verify owner/maintainer role.
	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err == nil && synced != nil && synced.UserID != userID {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "owner or maintainer role required"})
		}
	}

	type requestBody struct {
		Name    string `json:"name"`
		FromSHA string `json:"from_sha"`
	}
	var body requestBody
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_request", "message": err.Error()})
	}
	if body.Name == "" || body.FromSHA == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "missing_parameter", "message": "fields 'name' and 'from_sha' are required"})
	}

	if err := rc.githubService.CreateBranch(c.Context(), userID, owner, repo, body.Name, body.FromSHA); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create_branch_failed", "message": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"message": "branch created successfully", "name": body.Name})
}

// DeleteBranch deletes a branch from a repository.
// Requires owner or maintainer role.
//
// Route: DELETE /api/repos/:owner/:repo/branches/:branch
func (rc *RepoController) DeleteBranch(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")
	branchName := c.Params("branch")

	// Verify owner/maintainer role.
	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err == nil && synced != nil && synced.UserID != userID {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil || (collab.Role != domain.RoleOwner && collab.Role != domain.RoleMaintainer) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "owner or maintainer role required"})
		}
	}

	if err := rc.githubService.DeleteBranch(c.Context(), userID, owner, repo, branchName); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete_branch_failed", "message": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "branch deleted successfully"})
}

// GetCommitDiff returns the details of a single commit including changed files and patches.
//
// Route: GET /api/repos/:owner/:repo/commits/:sha
func (rc *RepoController) GetCommitDiff(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")
	sha := c.Params("sha")

	commit, err := rc.githubService.GetCommitDiff(c.Context(), rc.resolveTargetUserID(c.Context(), userID, owner, repo), owner, repo, sha)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "get_commit_diff_failed", "message": err.Error()})
	}

	type fileResponse struct {
		Filename  string `json:"filename"`
		Status    string `json:"status"`
		Additions int    `json:"additions"`
		Deletions int    `json:"deletions"`
		Changes   int    `json:"changes"`
		Patch     string `json:"patch,omitempty"`
	}

	files := make([]fileResponse, 0, len(commit.Files))
	for _, f := range commit.Files {
		if f == nil {
			continue
		}
		patch := ""
		if f.Patch != nil {
			patch = *f.Patch
		}
		files = append(files, fileResponse{
			Filename:  f.GetFilename(),
			Status:    f.GetStatus(),
			Additions: f.GetAdditions(),
			Deletions: f.GetDeletions(),
			Changes:   f.GetChanges(),
			Patch:     patch,
		})
	}

	msg := ""
	author := ""
	date := ""
	if commit.Commit != nil {
		msg = commit.Commit.GetMessage()
		if commit.Commit.Author != nil {
			author = commit.Commit.Author.GetName()
			if commit.Commit.Author.Date != nil {
				date = commit.Commit.Author.Date.Format(time.RFC3339)
			}
		}
	}

	return c.JSON(fiber.Map{
		"sha":     commit.GetSHA(),
		"message": msg,
		"author":  author,
		"date":    date,
		"files":   files,
		"stats": fiber.Map{
			"additions": commit.GetStats().GetAdditions(),
			"deletions": commit.GetStats().GetDeletions(),
			"total":     commit.GetStats().GetTotal(),
		},
	})
}

// GetRepoStats returns aggregate task statistics for a repository.
//
// Route: GET /api/repos/:owner/:repo/stats
func (rc *RepoController) GetRepoStats(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Look up the synced repo to get its numeric ID.
	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database_error", "message": err.Error()})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "repository is not synced in CodeTasker"})
	}

	// Verify the caller has at least viewer access.
	isAuthorized := synced.UserID == userID
	if !isAuthorized {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		isAuthorized = collab != nil
	}
	if !isAuthorized {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "access denied"})
	}

	// Fetch all tasks and aggregate counts in memory.
	tasks, err := rc.taskRepo.FindByRepo(c.Context(), synced.RepoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "fetch_tasks_failed", "message": err.Error()})
	}

	type AssigneeStat struct {
		Username   string `json:"username"`
		AvatarURL  string `json:"avatar_url"`
		Total      int    `json:"total"`
		Open       int    `json:"open"`
		InProgress int    `json:"in_progress"`
		Resolved   int    `json:"resolved"`
	}

	type DebtStat struct {
		High   int `json:"high"`
		Medium int `json:"medium"`
		Low    int `json:"low"`
	}

	stats := struct {
		Total      int                      `json:"total"`
		Open       int                      `json:"open"`
		InProgress int                      `json:"in_progress"`
		Resolved   int                      `json:"resolved"`
		ByType     map[string]int           `json:"by_type"`
		ByAssignee map[string]*AssigneeStat `json:"by_assignee"`
		Debt       DebtStat                 `json:"debt"`
	}{
		ByType:     make(map[string]int),
		ByAssignee: make(map[string]*AssigneeStat),
	}

	for _, t := range tasks {
		stats.Total++
		switch t.Status {
		case domain.TaskStatusOpen:
			stats.Open++
		case domain.TaskStatusInProgress:
			stats.InProgress++
		case domain.TaskStatusResolved:
			stats.Resolved++
		}
		if t.Type != "" {
			stats.ByType[t.Type]++
		}

		// Priority/Debt level aggregation
		switch strings.ToUpper(t.Type) {
		case "BUG", "FIXME":
			stats.Debt.High++
		case "TODO", "HACK":
			stats.Debt.Medium++
		case "NOTE":
			stats.Debt.Low++
		default:
			stats.Debt.Low++
		}

		// Assignee aggregation
		if t.AssigneeUsername != "" {
			aStat, exists := stats.ByAssignee[t.AssigneeUsername]
			if !exists {
				aStat = &AssigneeStat{
					Username:  t.AssigneeUsername,
					AvatarURL: t.AssigneeAvatarURL,
				}
				stats.ByAssignee[t.AssigneeUsername] = aStat
			}
			aStat.Total++
			switch t.Status {
			case domain.TaskStatusOpen:
				aStat.Open++
			case domain.TaskStatusInProgress:
				aStat.InProgress++
			case domain.TaskStatusResolved:
				aStat.Resolved++
			}
		}
	}

	return c.JSON(fiber.Map{"stats": stats})
}

// GetActivity returns the recent activity log for a repository.
//
// Route: GET /api/repos/:owner/:repo/activity
func (rc *RepoController) GetActivity(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}
	owner := c.Params("owner")
	repo := c.Params("repo")

	// Look up the synced repo to get its numeric ID.
	synced, err := rc.syncedRepoRepo.FindByRepoName(c.Context(), owner+"/"+repo)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database_error", "message": err.Error()})
	}
	if synced == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not_found", "message": "repository is not synced in CodeTasker"})
	}

	// Verify the caller has at least viewer access.
	isAuthorized := synced.UserID == userID
	if !isAuthorized {
		collab, _ := rc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		isAuthorized = collab != nil
	}
	if !isAuthorized {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "access denied"})
	}

	activities, err := rc.activityRepo.FindByRepo(c.Context(), synced.RepoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "fetch_activity_failed", "message": err.Error()})
	}

	// Suppress unused import warning — primitive is used elsewhere in this file.
	_ = primitive.NilObjectID

	return c.JSON(fiber.Map{"activities": activities, "count": len(activities)})
}

// resolveTargetUserID returns the UserID of the repository owner if the requester
// is an authorized collaborator, allowing them to browse and interact using the
// owner's token.
func (rc *RepoController) resolveTargetUserID(ctx context.Context, userID primitive.ObjectID, owner, repo string) primitive.ObjectID {
	if owner == "" || repo == "" {
		return userID
	}
	repoFullName := fmt.Sprintf("%s/%s", owner, repo)
	synced, err := rc.syncedRepoRepo.FindByRepoName(ctx, repoFullName)
	if err != nil || synced == nil {
		return userID
	}
	if synced.UserID == userID {
		return userID
	}
	// Verify user is a collaborator
	collab, err := rc.collaboratorRepo.FindByUserAndRepo(ctx, userID, synced.RepoID)
	if err == nil && collab != nil {
		return synced.UserID
	}
	return userID
}
