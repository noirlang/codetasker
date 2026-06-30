package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/codetasker/backend/internal/debt"
	"github.com/codetasker/backend/internal/domain"
	"github.com/codetasker/backend/internal/repository"
	"github.com/google/go-github/v62/github"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

type DebtService struct {
	githubService *GithubService
	debtRepo      *repository.DebtRepository
	taskRepo      *repository.TaskRepository
	userRepo      *repository.UserRepository
	log           *zap.Logger
}

func NewDebtService(
	githubService *GithubService,
	debtRepo *repository.DebtRepository,
	taskRepo *repository.TaskRepository,
	userRepo *repository.UserRepository,
	log *zap.Logger,
) *DebtService {
	return &DebtService{
		githubService: githubService,
		debtRepo:      debtRepo,
		taskRepo:      taskRepo,
		userRepo:      userRepo,
		log:           log,
	}
}

func (s *DebtService) AnalyzeGitHubRepo(ctx context.Context, actorID, targetUserID primitive.ObjectID, repoID int64, owner, repo string, days int, hourlyCost float64) (*debt.AnalysisResult, error) {
	if days <= 0 {
		days = 90
	}
	if hourlyCost <= 0 {
		hourlyCost = 35
	}

	now := time.Now().UTC()
	repository, err := s.githubService.GetRepository(ctx, targetUserID, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("get repository: %w", err)
	}
	defaultBranch := repository.GetDefaultBranch()
	if defaultBranch == "" {
		defaultBranch = "main"
	}
	if repoID == 0 {
		repoID = repository.GetID()
	}

	commits, err := s.githubService.ListCommitsSince(ctx, targetUserID, owner, repo, defaultBranch, now.AddDate(0, 0, -days))
	if err != nil {
		return nil, fmt.Errorf("list commits since: %w", err)
	}

	history := make([]debt.CommitChange, 0, len(commits))
	for _, commit := range commits {
		if commit == nil {
			continue
		}
		sha := commit.GetSHA()
		if sha == "" {
			continue
		}
		detail, err := s.githubService.GetCommitDiff(ctx, targetUserID, owner, repo, sha)
		if err != nil {
			s.log.Warn("debt analysis: failed to fetch commit diff",
				zap.String("repo", owner+"/"+repo),
				zap.String("sha", sha),
				zap.Error(err),
			)
			continue
		}
		history = append(history, toDebtCommit(commit, detail))
	}

	tree, err := s.githubService.GetTree(ctx, targetUserID, owner, repo, defaultBranch)
	if err != nil {
		return nil, fmt.Errorf("get tree: %w", err)
	}

	var sourcePaths []string
	var allPaths []string
	for _, entry := range tree.Entries {
		if entry == nil || entry.GetType() != "blob" {
			continue
		}
		filePath := entry.GetPath()
		if entry.GetSize() > 1_000_000 || !debt.SupportedPath(filePath) || debt.IsIgnoredPath(filePath) {
			continue
		}
		allPaths = append(allPaths, filePath)
		if !debt.IsTestPath(filePath) {
			sourcePaths = append(sourcePaths, filePath)
		}
	}

	files := s.fetchSourceFiles(ctx, targetUserID, owner, repo, defaultBranch, sourcePaths)
	result := debt.AnalyzeSnapshot(owner+"/"+repo, history, files, allPaths, debt.Options{
		Repo:       owner + "/" + repo,
		Days:       days,
		HourlyCost: hourlyCost,
		Now:        now,
	})

	saved, err := s.debtRepo.SaveAnalysis(ctx, actorID, repoID, owner+"/"+repo, hourlyCost, result)
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

func (s *DebtService) GetLatestAnalysis(ctx context.Context, repoID int64) (*debt.AnalysisResult, error) {
	result, _, err := s.debtRepo.FindLatestAnalysis(ctx, repoID)
	return result, err
}

func (s *DebtService) CreateTaskFromSuggestion(ctx context.Context, actorID primitive.ObjectID, suggestionID primitive.ObjectID, expectedRepoID ...int64) (*domain.Task, error) {
	suggestion, err := s.debtRepo.FindSuggestedTaskByID(ctx, suggestionID)
	if err != nil {
		return nil, err
	}
	if suggestion == nil {
		return nil, fmt.Errorf("debt suggested task not found")
	}
	if len(expectedRepoID) > 0 && expectedRepoID[0] != 0 && suggestion.RepoID != expectedRepoID[0] {
		return nil, fmt.Errorf("suggestion belongs to a different repository")
	}
	if suggestion.Status == "created" && suggestion.CreatedTaskID != nil {
		return s.taskRepo.FindByID(ctx, *suggestion.CreatedTaskID)
	}

	return s.createTaskFromSuggestion(ctx, actorID, suggestion)
}

func (s *DebtService) CreateHighPriorityTasks(ctx context.Context, actorID primitive.ObjectID, repoID int64) ([]domain.Task, error) {
	_, run, err := s.debtRepo.FindLatestAnalysis(ctx, repoID)
	if err != nil {
		return nil, err
	}
	if run == nil {
		return []domain.Task{}, nil
	}

	suggestions, err := s.debtRepo.FindSuggestedTasksByRun(ctx, run.ID, []string{string(debt.LevelHigh), string(debt.LevelCritical)}, "suggested")
	if err != nil {
		return nil, err
	}

	created := make([]domain.Task, 0, len(suggestions))
	for i := range suggestions {
		task, err := s.createTaskFromSuggestion(ctx, actorID, &suggestions[i])
		if err != nil {
			return created, err
		}
		if task != nil {
			created = append(created, *task)
		}
	}
	return created, nil
}

func (s *DebtService) createTaskFromSuggestion(ctx context.Context, actorID primitive.ObjectID, suggestion *domain.DebtSuggestedTask) (*domain.Task, error) {
	actor, _ := s.userRepo.FindByObjectID(ctx, actorID)
	actorName := ""
	actorAvatar := ""
	if actor != nil {
		actorName = actor.Username
		actorAvatar = actor.AvatarURL
	}

	content := suggestion.Title + "\n\n" + suggestion.Description
	task := &domain.Task{
		RepoID:             suggestion.RepoID,
		RepoName:           suggestion.RepoName,
		FilePath:           suggestion.FilePath,
		LineNumber:         1,
		Content:            content,
		Type:               "TODO",
		Status:             domain.TaskStatusOpen,
		CommitSHA:          "debt-analysis",
		CreatedByUsername:  actorName,
		CreatedByAvatarURL: actorAvatar,
		CreatedAt:          time.Now().UTC(),
		UpdatedAt:          time.Now().UTC(),
	}

	if err := s.taskRepo.UpsertTask(ctx, task); err != nil {
		return nil, err
	}

	created, err := s.taskRepo.FindByRepoFileContent(ctx, suggestion.RepoID, suggestion.FilePath, content, "TODO")
	if err != nil {
		return nil, err
	}
	if created == nil {
		return nil, fmt.Errorf("created debt task could not be reloaded")
	}

	if err := s.debtRepo.MarkSuggestedTaskCreated(ctx, suggestion.ID, created.ID); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *DebtService) fetchSourceFiles(ctx context.Context, userID primitive.ObjectID, owner, repo, ref string, paths []string) []debt.SourceFile {
	if len(paths) == 0 {
		return []debt.SourceFile{}
	}

	type result struct {
		file debt.SourceFile
		err  error
	}

	workers := 8
	if len(paths) < workers {
		workers = len(paths)
	}

	workCh := make(chan string, len(paths))
	resultCh := make(chan result, len(paths))
	for _, p := range paths {
		workCh <- p
	}
	close(workCh)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for p := range workCh {
				content, err := s.githubService.GetContents(ctx, userID, owner, repo, p, ref)
				resultCh <- result{file: debt.SourceFile{Path: p, Content: content}, err: err}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	files := []debt.SourceFile{}
	for res := range resultCh {
		if res.err != nil {
			s.log.Warn("debt analysis: failed to fetch source file",
				zap.String("repo", owner+"/"+repo),
				zap.String("file", res.file.Path),
				zap.Error(res.err),
			)
			continue
		}
		files = append(files, res.file)
	}
	return files
}

func toDebtCommit(commit *github.RepositoryCommit, detail *github.RepositoryCommit) debt.CommitChange {
	change := debt.CommitChange{
		SHA: commit.GetSHA(),
	}
	if commit.Commit != nil {
		change.Message = commit.Commit.GetMessage()
		if commit.Commit.Author != nil {
			change.AuthorName = commit.Commit.Author.GetName()
			change.AuthorEmail = commit.Commit.Author.GetEmail()
			if commit.Commit.Author.Date != nil {
				change.Date = commit.Commit.Author.Date.Time
			}
		}
	}
	if change.AuthorName == "" && commit.Author != nil {
		change.AuthorName = commit.Author.GetLogin()
	}

	if detail != nil {
		for _, file := range detail.Files {
			if file == nil {
				continue
			}
			filePath := strings.ReplaceAll(file.GetFilename(), "\\", "/")
			change.Files = append(change.Files, debt.FileChange{
				Path:    filePath,
				Added:   file.GetAdditions(),
				Deleted: file.GetDeletions(),
			})
		}
	}
	return change
}
