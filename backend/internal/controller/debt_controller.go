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

type DebtController struct {
	debtService      *service.DebtService
	githubService    *service.GithubService
	syncedRepoRepo   *repository.SyncedRepository
	collaboratorRepo *repository.CollaboratorRepository
}

func NewDebtController(
	debtService *service.DebtService,
	githubService *service.GithubService,
	syncedRepoRepo *repository.SyncedRepository,
	collaboratorRepo *repository.CollaboratorRepository,
) *DebtController {
	return &DebtController{
		debtService:      debtService,
		githubService:    githubService,
		syncedRepoRepo:   syncedRepoRepo,
		collaboratorRepo: collaboratorRepo,
	}
}

func (dc *DebtController) RegisterRoutes(group fiber.Router) {
	group.Get("/repos/:owner/:repo/debt", dc.GetLatestAnalysis)
	group.Post("/repos/:owner/:repo/debt/analyze", dc.AnalyzeRepo)
	group.Post("/repos/:owner/:repo/debt/tasks/:suggestionId/create", dc.CreateSuggestedTask)
	group.Post("/repos/:owner/:repo/debt/tasks/create-all", dc.CreateAllHighPriorityTasks)
}

func (dc *DebtController) GetLatestAnalysis(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	repoCtx, err := dc.resolveRepoContext(c, userID)
	if err != nil {
		return err
	}

	result, err := dc.debtService.GetLatestAnalysis(c.Context(), repoCtx.repoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "get_debt_analysis_failed", "message": err.Error()})
	}
	if result == nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "no_analysis", "message": "No debt analysis has been run for this repository yet."})
	}
	return c.JSON(fiber.Map{"analysis": result})
}

func (dc *DebtController) AnalyzeRepo(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	repoCtx, err := dc.resolveRepoContext(c, userID)
	if err != nil {
		return err
	}

	type requestBody struct {
		Days            int     `json:"days"`
		HourlyCost      float64 `json:"hourly_cost"`
		IncludeSnippets bool    `json:"include_snippets"`
	}
	body := requestBody{
		Days:       parsePositiveQueryInt(c.Query("days"), 90),
		HourlyCost: parseFloatQuery(c.Query("hourly_cost"), 35),
	}
	if len(c.Body()) > 0 {
		_ = c.BodyParser(&body)
	}
	if body.Days <= 0 {
		body.Days = 90
	}
	if body.HourlyCost <= 0 {
		body.HourlyCost = 35
	}
	// MVP privacy rule: snippets are ignored unless explicitly implemented later.
	_ = body.IncludeSnippets

	result, err := dc.debtService.AnalyzeGitHubRepo(c.Context(), userID, repoCtx.targetUserID, repoCtx.repoID, repoCtx.owner, repoCtx.repo, body.Days, body.HourlyCost)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "debt_analysis_failed", "message": err.Error()})
	}
	return c.JSON(fiber.Map{"analysis": result})
}

func (dc *DebtController) CreateSuggestedTask(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	repoCtx, err := dc.resolveRepoContext(c, userID)
	if err != nil {
		return err
	}
	if !repoCtx.canWrite {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "developer, maintainer, or owner role required"})
	}

	suggestionID, err := primitive.ObjectIDFromHex(c.Params("suggestionId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid_parameter", "message": "suggestion id must be a valid object id"})
	}

	task, err := dc.debtService.CreateTaskFromSuggestion(c.Context(), userID, suggestionID, repoCtx.repoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create_debt_task_failed", "message": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"task": task})
}

func (dc *DebtController) CreateAllHighPriorityTasks(c *fiber.Ctx) error {
	userID, err := middleware.GetUserID(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	repoCtx, err := dc.resolveRepoContext(c, userID)
	if err != nil {
		return err
	}
	if !repoCtx.canWrite {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "developer, maintainer, or owner role required"})
	}

	tasks, err := dc.debtService.CreateHighPriorityTasks(c.Context(), userID, repoCtx.repoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "create_debt_tasks_failed", "message": err.Error()})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"tasks": tasks, "count": len(tasks)})
}

type debtRepoContext struct {
	owner        string
	repo         string
	repoID       int64
	repoName     string
	targetUserID primitive.ObjectID
	canWrite     bool
}

func (dc *DebtController) resolveRepoContext(c *fiber.Ctx, userID primitive.ObjectID) (*debtRepoContext, error) {
	owner := c.Params("owner")
	repo := c.Params("repo")
	fullName := fmt.Sprintf("%s/%s", owner, repo)

	synced, err := dc.syncedRepoRepo.FindByRepoName(c.Context(), fullName)
	if err != nil {
		return nil, c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "database_error", "message": err.Error()})
	}
	if synced != nil {
		repoCtx := &debtRepoContext{
			owner:        owner,
			repo:         repo,
			repoID:       synced.RepoID,
			repoName:     synced.RepoName,
			targetUserID: synced.UserID,
			canWrite:     synced.UserID == userID,
		}
		if synced.UserID == userID {
			return repoCtx, nil
		}
		collab, _ := dc.collaboratorRepo.FindByUserAndRepo(c.Context(), userID, synced.RepoID)
		if collab == nil {
			return nil, c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "access denied"})
		}
		repoCtx.canWrite = collab.Role == domain.RoleOwner || collab.Role == domain.RoleMaintainer || collab.Role == domain.RoleDeveloper
		return repoCtx, nil
	}

	ghRepo, err := dc.githubService.GetRepository(c.Context(), userID, owner, repo)
	if err != nil {
		return nil, c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden", "message": "repository is not synced and GitHub access could not be verified"})
	}

	return &debtRepoContext{
		owner:        owner,
		repo:         repo,
		repoID:       ghRepo.GetID(),
		repoName:     fullName,
		targetUserID: userID,
		canWrite:     true,
	}, nil
}

func parseFloatQuery(value string, fallback float64) float64 {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
