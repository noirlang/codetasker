// Package controller implements the HTTP handler layer of CodeTasker.
// repo_controller.go handles repository listing and file tree/content
// endpoints. All routes require a valid JWT (Protected middleware).
package controller

import (
	"fmt"
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
}

// NewRepoController constructs a RepoController with its dependencies.
func NewRepoController(
	cfg *config.Config,
	githubService *service.GithubService,
	taskService *service.TaskService,
	syncedRepoRepo *repository.SyncedRepository,
	collaboratorRepo *repository.CollaboratorRepository,
	userRepo *repository.UserRepository,
) *RepoController {
	return &RepoController{
		cfg:              cfg,
		githubService:    githubService,
		taskService:      taskService,
		syncedRepoRepo:   syncedRepoRepo,
		collaboratorRepo: collaboratorRepo,
		userRepo:         userRepo,
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
	group.Get("/repos/:owner/:repo/pulls", rc.GetPulls)
	group.Post("/repos/:owner/:repo/merge", rc.MergeBranch)

	// Collaborators management
	group.Get("/repos/:owner/:repo/collaborators", rc.GetCollaborators)
	group.Post("/repos/:owner/:repo/collaborators", rc.AddCollaborator)
	group.Patch("/repos/:owner/:repo/collaborators/:id", rc.UpdateCollaboratorRole)
	group.Delete("/repos/:owner/:repo/collaborators/:id", rc.RemoveCollaborator)
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

	tree, err := rc.githubService.GetTree(c.Context(), userID, owner, repo, branch)
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

	content, err := rc.githubService.GetContents(c.Context(), userID, owner, repo, path, ref)
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

	commits, err := rc.githubService.ListCommits(c.Context(), userID, owner, repo, branch)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "get_commits_failed",
			"message": err.Error(),
		})
	}

	type commitResponse struct {
		SHA       string `json:"sha"`
		Message   string `json:"message"`
		Author    string `json:"author"`
		AvatarURL string `json:"avatar_url"`
		Date      string `json:"date"`
	}

	response := make([]commitResponse, 0, len(commits))
	for _, commit := range commits {
		sha := commit.GetSHA()
		msg := ""
		author := ""
		avatar := ""
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

		if commit.Author != nil {
			avatar = commit.Author.GetAvatarURL()
			if author == "" {
				author = commit.Author.GetLogin()
			}
		}

		response = append(response, commitResponse{
			SHA:       sha,
			Message:   msg,
			Author:    author,
			AvatarURL: avatar,
			Date:      date,
		})
	}

	return c.JSON(fiber.Map{
		"commits": response,
		"count":   len(response),
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
		fmt.Printf("[DEBUG] Collaborator invitee '%s' not found in database\n", body.Username)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "user_not_found",
			"message": fmt.Sprintf("The user '%s' must log in to CodeTasker with GitHub before they can be added as a collaborator", body.Username),
		})
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

	return c.JSON(fiber.Map{
		"message":      "collaborator added successfully",
		"collaborator": collab,
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
