package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/codetasker/backend/internal/database"
	"github.com/codetasker/backend/internal/debt"
	"github.com/codetasker/backend/internal/domain"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DebtRepository struct {
	runsCol    *mongo.Collection
	metricsCol *mongo.Collection
	tasksCol   *mongo.Collection
}

func NewDebtRepository(db *database.Database) *DebtRepository {
	return &DebtRepository{
		runsCol:    db.Collection("debt_analysis_runs"),
		metricsCol: db.Collection("debt_file_metrics"),
		tasksCol:   db.Collection("debt_suggested_tasks"),
	}
}

func (r *DebtRepository) SaveAnalysis(
	ctx context.Context,
	userID primitive.ObjectID,
	repoID int64,
	repoName string,
	hourlyCost float64,
	result debt.AnalysisResult,
) (debt.AnalysisResult, error) {
	run := domain.DebtAnalysisRun{
		ID:                    primitive.NewObjectID(),
		RepoID:                repoID,
		RepoName:              repoName,
		UserID:                userID,
		AnalyzedAt:            result.AnalyzedAt,
		Days:                  result.Days,
		HourlyEngineerCostUSD: hourlyCost,
		Summary:               toDomainSummary(result.Summary),
	}
	if run.AnalyzedAt.IsZero() {
		run.AnalyzedAt = time.Now().UTC()
	}

	if _, err := r.runsCol.InsertOne(ctx, run); err != nil {
		return result, fmt.Errorf("insert debt analysis run: %w", err)
	}

	now := time.Now().UTC()
	metricDocs := make([]interface{}, 0, len(result.Hotspots))
	taskDocs := []interface{}{}

	for i := range result.Hotspots {
		hotspot := &result.Hotspots[i]
		metricDocs = append(metricDocs, domain.DebtFileMetric{
			ID:                   primitive.NewObjectID(),
			RunID:                run.ID,
			RepoID:               repoID,
			RepoName:             repoName,
			FilePath:             hotspot.File,
			DebtScore:            hotspot.DebtScore,
			Level:                string(hotspot.Level),
			Metrics:              toDomainMetrics(hotspot.Metrics),
			EstimatedMonthlyCost: hotspot.EstimatedMonthlyCost,
			Reasons:              hotspot.Reasons,
			CreatedAt:            now,
		})

		for j := range hotspot.SuggestedTasks {
			task := &hotspot.SuggestedTasks[j]
			id := primitive.NewObjectID()
			task.ID = id.Hex()
			task.Status = "suggested"
			taskDocs = append(taskDocs, domain.DebtSuggestedTask{
				ID:                   id,
				RunID:                run.ID,
				RepoID:               repoID,
				RepoName:             repoName,
				FilePath:             hotspot.File,
				Title:                task.Title,
				Description:          task.Description,
				Actions:              task.Actions,
				DebtScore:            hotspot.DebtScore,
				Level:                string(hotspot.Level),
				EstimatedMonthlyCost: hotspot.EstimatedMonthlyCost,
				Status:               "suggested",
				CreatedAt:            now,
				UpdatedAt:            now,
			})
		}
	}

	if len(metricDocs) > 0 {
		if _, err := r.metricsCol.InsertMany(ctx, metricDocs); err != nil {
			return result, fmt.Errorf("insert debt file metrics: %w", err)
		}
	}
	if len(taskDocs) > 0 {
		if _, err := r.tasksCol.InsertMany(ctx, taskDocs); err != nil {
			return result, fmt.Errorf("insert debt suggested tasks: %w", err)
		}
	}

	return result, nil
}

func (r *DebtRepository) FindLatestAnalysis(ctx context.Context, repoID int64) (*debt.AnalysisResult, *domain.DebtAnalysisRun, error) {
	var run domain.DebtAnalysisRun
	err := r.runsCol.FindOne(ctx, bson.M{"repo_id": repoID}, options.FindOne().SetSort(bson.D{{Key: "analyzed_at", Value: -1}})).Decode(&run)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("find latest debt analysis: %w", err)
	}

	result, err := r.analysisForRun(ctx, run)
	if err != nil {
		return nil, nil, err
	}
	return result, &run, nil
}

func (r *DebtRepository) FindSuggestedTaskByID(ctx context.Context, id primitive.ObjectID) (*domain.DebtSuggestedTask, error) {
	var task domain.DebtSuggestedTask
	err := r.tasksCol.FindOne(ctx, bson.M{"_id": id}).Decode(&task)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, fmt.Errorf("find debt suggested task: %w", err)
	}
	return &task, nil
}

func (r *DebtRepository) FindSuggestedTasksByRepo(ctx context.Context, repoID int64, levels []string, status string) ([]domain.DebtSuggestedTask, error) {
	filter := bson.M{"repo_id": repoID}
	if len(levels) > 0 {
		filter["level"] = bson.M{"$in": levels}
	}
	if status != "" {
		filter["status"] = status
	}

	cursor, err := r.tasksCol.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "debt_score", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find debt suggested tasks by repo: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []domain.DebtSuggestedTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode debt suggested tasks: %w", err)
	}
	if tasks == nil {
		tasks = []domain.DebtSuggestedTask{}
	}
	return tasks, nil
}

func (r *DebtRepository) FindSuggestedTasksByRun(ctx context.Context, runID primitive.ObjectID, levels []string, status string) ([]domain.DebtSuggestedTask, error) {
	filter := bson.M{"run_id": runID}
	if len(levels) > 0 {
		filter["level"] = bson.M{"$in": levels}
	}
	if status != "" {
		filter["status"] = status
	}

	cursor, err := r.tasksCol.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "debt_score", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find debt suggested tasks by run: %w", err)
	}
	defer cursor.Close(ctx)

	var tasks []domain.DebtSuggestedTask
	if err := cursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode debt suggested tasks by run: %w", err)
	}
	if tasks == nil {
		tasks = []domain.DebtSuggestedTask{}
	}
	return tasks, nil
}

func (r *DebtRepository) MarkSuggestedTaskCreated(ctx context.Context, suggestionID, taskID primitive.ObjectID) error {
	_, err := r.tasksCol.UpdateOne(ctx,
		bson.M{"_id": suggestionID},
		bson.M{"$set": bson.M{
			"status":          "created",
			"created_task_id": taskID,
			"updated_at":      time.Now().UTC(),
		}},
	)
	if err != nil {
		return fmt.Errorf("mark debt suggested task created: %w", err)
	}
	return nil
}

func (r *DebtRepository) analysisForRun(ctx context.Context, run domain.DebtAnalysisRun) (*debt.AnalysisResult, error) {
	cursor, err := r.metricsCol.Find(ctx, bson.M{"run_id": run.ID}, options.Find().SetSort(bson.D{{Key: "debt_score", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("find debt file metrics: %w", err)
	}
	defer cursor.Close(ctx)

	var metrics []domain.DebtFileMetric
	if err := cursor.All(ctx, &metrics); err != nil {
		return nil, fmt.Errorf("decode debt file metrics: %w", err)
	}

	taskCursor, err := r.tasksCol.Find(ctx, bson.M{"run_id": run.ID})
	if err != nil {
		return nil, fmt.Errorf("find debt suggested tasks: %w", err)
	}
	defer taskCursor.Close(ctx)

	var tasks []domain.DebtSuggestedTask
	if err := taskCursor.All(ctx, &tasks); err != nil {
		return nil, fmt.Errorf("decode debt suggested tasks: %w", err)
	}

	tasksByFile := make(map[string][]debt.SuggestedTask)
	for _, task := range tasks {
		apiTask := debt.SuggestedTask{
			ID:          task.ID.Hex(),
			Title:       task.Title,
			Description: task.Description,
			Actions:     task.Actions,
			Status:      task.Status,
		}
		if task.CreatedTaskID != nil {
			apiTask.CreatedTaskID = task.CreatedTaskID.Hex()
		}
		tasksByFile[task.FilePath] = append(tasksByFile[task.FilePath], apiTask)
	}

	hotspots := make([]debt.Hotspot, 0, len(metrics))
	for _, metric := range metrics {
		suggested := tasksByFile[metric.FilePath]
		if suggested == nil {
			suggested = []debt.SuggestedTask{}
		}
		hotspots = append(hotspots, debt.Hotspot{
			File:                 metric.FilePath,
			DebtScore:            metric.DebtScore,
			Level:                debt.Level(metric.Level),
			Metrics:              toDebtMetrics(metric.Metrics),
			EstimatedMonthlyCost: metric.EstimatedMonthlyCost,
			Reasons:              metric.Reasons,
			SuggestedTasks:       suggested,
		})
	}

	result := &debt.AnalysisResult{
		Repo:       run.RepoName,
		AnalyzedAt: run.AnalyzedAt,
		Days:       run.Days,
		Summary:    toDebtSummary(run.Summary),
		Hotspots:   hotspots,
	}
	return result, nil
}

func toDomainSummary(summary debt.Summary) domain.DebtSummary {
	return domain.DebtSummary{
		FilesAnalyzed:        summary.FilesAnalyzed,
		Critical:             summary.Critical,
		High:                 summary.High,
		Medium:               summary.Medium,
		Low:                  summary.Low,
		EstimatedMonthlyCost: summary.EstimatedMonthlyCost,
	}
}

func toDebtSummary(summary domain.DebtSummary) debt.Summary {
	return debt.Summary{
		FilesAnalyzed:        summary.FilesAnalyzed,
		Critical:             summary.Critical,
		High:                 summary.High,
		Medium:               summary.Medium,
		Low:                  summary.Low,
		EstimatedMonthlyCost: summary.EstimatedMonthlyCost,
	}
}

func toDomainMetrics(metrics debt.Metrics) domain.DebtMetrics {
	return domain.DebtMetrics{
		CommitCount:                  metrics.CommitCount,
		ChurnAdded:                   metrics.ChurnAdded,
		ChurnDeleted:                 metrics.ChurnDeleted,
		TotalChurn:                   metrics.TotalChurn,
		AuthorCount:                  metrics.AuthorCount,
		LastTouchedAt:                metrics.LastTouchedAt,
		BugfixCommitCount:            metrics.BugfixCommitCount,
		LOC:                          metrics.LOC,
		FunctionCount:                metrics.FunctionCount,
		AvgFunctionLength:            metrics.AvgFunctionLength,
		MaxFunctionLength:            metrics.MaxFunctionLength,
		NestingDepthEstimate:         metrics.NestingDepthEstimate,
		CyclomaticComplexityEstimate: metrics.CyclomaticComplexityEstimate,
		TodoCount:                    metrics.TodoCount,
		DuplicateImportCount:         metrics.DuplicateImportCount,
		HasTests:                     metrics.HasTests,
		CoverageStatus:               metrics.CoverageStatus,
	}
}

func toDebtMetrics(metrics domain.DebtMetrics) debt.Metrics {
	return debt.Metrics{
		CommitCount:                  metrics.CommitCount,
		ChurnAdded:                   metrics.ChurnAdded,
		ChurnDeleted:                 metrics.ChurnDeleted,
		TotalChurn:                   metrics.TotalChurn,
		AuthorCount:                  metrics.AuthorCount,
		LastTouchedAt:                metrics.LastTouchedAt,
		BugfixCommitCount:            metrics.BugfixCommitCount,
		LOC:                          metrics.LOC,
		FunctionCount:                metrics.FunctionCount,
		AvgFunctionLength:            metrics.AvgFunctionLength,
		MaxFunctionLength:            metrics.MaxFunctionLength,
		NestingDepthEstimate:         metrics.NestingDepthEstimate,
		CyclomaticComplexityEstimate: metrics.CyclomaticComplexityEstimate,
		TodoCount:                    metrics.TodoCount,
		DuplicateImportCount:         metrics.DuplicateImportCount,
		HasTests:                     metrics.HasTests,
		CoverageStatus:               metrics.CoverageStatus,
	}
}
