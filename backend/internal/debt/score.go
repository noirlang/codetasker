package debt

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func AnalyzeSnapshot(repo string, commits []CommitChange, files []SourceFile, allPaths []string, opts Options) AnalysisResult {
	if opts.Days <= 0 {
		opts.Days = 90
	}
	if opts.HourlyCost <= 0 {
		opts.HourlyCost = 35
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}

	staticMetrics := AnalyzeStatic(files, allPaths)
	historyMetrics := buildHistoryMetrics(commits)

	paths := make(map[string]struct{})
	currentSourcePaths := make(map[string]struct{}, len(allPaths))
	for _, p := range allPaths {
		if SupportedPath(p) && !IsIgnoredPath(p) && !IsTestPath(p) {
			currentSourcePaths[p] = struct{}{}
		}
	}
	for p := range staticMetrics {
		paths[p] = struct{}{}
	}
	for p := range historyMetrics {
		if _, exists := currentSourcePaths[p]; exists {
			paths[p] = struct{}{}
		}
	}

	combined := make([]Metrics, 0, len(paths))
	pathList := make([]string, 0, len(paths))
	for p := range paths {
		m := historyMetrics[p]
		sm := staticMetrics[p]
		m.LOC = sm.LOC
		m.FunctionCount = sm.FunctionCount
		m.AvgFunctionLength = sm.AvgFunctionLength
		m.MaxFunctionLength = sm.MaxFunctionLength
		m.NestingDepthEstimate = sm.NestingDepthEstimate
		m.CyclomaticComplexityEstimate = sm.CyclomaticComplexityEstimate
		m.TodoCount = sm.TodoCount
		m.DuplicateImportCount = sm.DuplicateImportCount
		m.HasTests = sm.HasTests
		m.CoverageStatus = sm.CoverageStatus
		if m.CoverageStatus == "" {
			m.CoverageStatus = "not_detected"
		}
		combined = append(combined, m)
		pathList = append(pathList, p)
		historyMetrics[p] = m
	}

	maxes := metricMaxes(combined)
	mlModel, mlErr := DefaultMLModel()
	if mlErr != nil {
		mlModel = nil
	}
	hotspots := make([]Hotspot, 0, len(pathList))
	summary := Summary{}

	for _, p := range pathList {
		metrics := historyMetrics[p]
		heuristicScore := CalculateDebtScore(metrics, maxes)
		score := heuristicScore
		var mlPrediction *MLPrediction
		if mlModel != nil {
			prediction := mlModel.Predict(metrics, heuristicScore)
			score = heuristicScore + prediction.ScoreAdjustment
			mlPrediction = &prediction
		}
		level := LevelForScore(score)
		cost := EstimateMonthlyCost(metrics, level, opts.HourlyCost)
		reasons := ExplainRisk(metrics, maxes)
		if mlPrediction != nil && mlPrediction.RiskProbability >= 0.55 {
			reasons = append(reasons, fmt.Sprintf("ML model predicts %s risk (%.0f%%)", strings.ToLower(mlPrediction.RiskLabel), mlPrediction.RiskProbability*100))
		}

		hotspot := Hotspot{
			File:                 p,
			DebtScore:            score,
			HeuristicScore:       heuristicScore,
			Level:                level,
			Metrics:              metrics,
			MLPrediction:         mlPrediction,
			EstimatedMonthlyCost: roundMoney(cost),
			Reasons:              reasons,
		}
		if level == LevelHigh || level == LevelCritical {
			hotspot.SuggestedTasks = []SuggestedTask{BuildSuggestedTask(hotspot, opts.Days)}
		} else {
			hotspot.SuggestedTasks = []SuggestedTask{}
		}

		hotspots = append(hotspots, hotspot)
		summary.EstimatedMonthlyCost += hotspot.EstimatedMonthlyCost
		switch level {
		case LevelCritical:
			summary.Critical++
		case LevelHigh:
			summary.High++
		case LevelMedium:
			summary.Medium++
		default:
			summary.Low++
		}
	}

	sort.Slice(hotspots, func(i, j int) bool {
		if hotspots[i].DebtScore == hotspots[j].DebtScore {
			return hotspots[i].File < hotspots[j].File
		}
		return hotspots[i].DebtScore > hotspots[j].DebtScore
	})

	summary.FilesAnalyzed = len(hotspots)
	summary.EstimatedMonthlyCost = roundMoney(summary.EstimatedMonthlyCost)

	return AnalysisResult{
		Repo:       repo,
		AnalyzedAt: opts.Now,
		Days:       opts.Days,
		Summary:    summary,
		Hotspots:   hotspots,
	}
}

func buildHistoryMetrics(commits []CommitChange) map[string]Metrics {
	type mutable struct {
		metrics Metrics
		authors map[string]struct{}
		seenSHA map[string]struct{}
	}

	byPath := make(map[string]*mutable)
	for _, commit := range commits {
		author := commit.AuthorEmail
		if author == "" {
			author = commit.AuthorName
		}
		bugfix := IsBugfixCommit(commit.Message)

		for _, file := range commit.Files {
			if file.Path == "" {
				continue
			}
			item := byPath[file.Path]
			if item == nil {
				item = &mutable{
					authors: make(map[string]struct{}),
					seenSHA: make(map[string]struct{}),
				}
				byPath[file.Path] = item
			}

			if _, seen := item.seenSHA[commit.SHA]; !seen {
				item.metrics.CommitCount++
				item.seenSHA[commit.SHA] = struct{}{}
				if bugfix {
					item.metrics.BugfixCommitCount++
				}
			}
			if author != "" {
				item.authors[author] = struct{}{}
			}

			item.metrics.ChurnAdded += file.Added
			item.metrics.ChurnDeleted += file.Deleted
			item.metrics.TotalChurn += file.Added + file.Deleted
			if commit.Date.After(time.Time{}) {
				if item.metrics.LastTouchedAt == nil || commit.Date.After(*item.metrics.LastTouchedAt) {
					t := commit.Date
					item.metrics.LastTouchedAt = &t
				}
			}
		}
	}

	result := make(map[string]Metrics, len(byPath))
	for p, item := range byPath {
		item.metrics.AuthorCount = len(item.authors)
		result[p] = item.metrics
	}
	return result
}

type maxMetrics struct {
	CommitCount       int
	TotalChurn        int
	Complexity        int
	LOC               int
	BugfixCommitCount int
	TodoCount         int
	AuthorCount       int
}

func metricMaxes(metrics []Metrics) maxMetrics {
	var max maxMetrics
	for _, m := range metrics {
		max.CommitCount = maxInt(max.CommitCount, m.CommitCount)
		max.TotalChurn = maxInt(max.TotalChurn, m.TotalChurn)
		max.Complexity = maxInt(max.Complexity, m.CyclomaticComplexityEstimate)
		max.LOC = maxInt(max.LOC, m.LOC)
		max.BugfixCommitCount = maxInt(max.BugfixCommitCount, m.BugfixCommitCount)
		max.TodoCount = maxInt(max.TodoCount, m.TodoCount)
		max.AuthorCount = maxInt(max.AuthorCount, m.AuthorCount)
	}
	return max
}

func CalculateDebtScore(metrics Metrics, max maxMetrics) int {
	score := 0.30*normalize(metrics.CommitCount, max.CommitCount) +
		0.20*normalize(metrics.TotalChurn, max.TotalChurn) +
		0.15*normalize(metrics.CyclomaticComplexityEstimate, max.Complexity) +
		0.10*normalize(metrics.LOC, max.LOC) +
		0.10*normalize(metrics.BugfixCommitCount, max.BugfixCommitCount) +
		0.10*normalize(metrics.TodoCount, max.TodoCount) +
		0.05*normalize(metrics.AuthorCount, max.AuthorCount)

	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return int(math.Round(score))
}

func LevelForScore(score int) Level {
	switch {
	case score >= 85:
		return LevelCritical
	case score >= 70:
		return LevelHigh
	case score >= 40:
		return LevelMedium
	default:
		return LevelLow
	}
}

func EstimateMonthlyCost(metrics Metrics, level Level, hourlyCost float64) float64 {
	if hourlyCost <= 0 {
		hourlyCost = 35
	}
	factor := 1.0
	switch level {
	case LevelMedium:
		factor = 1.4
	case LevelHigh:
		factor = 2.0
	case LevelCritical:
		factor = 3.0
	}

	monthlyHoursWasted := (float64(metrics.BugfixCommitCount) * 1.5) +
		(float64(metrics.CommitCount) * factor * 0.25) +
		(float64(metrics.TodoCount) * 0.2)
	return monthlyHoursWasted * hourlyCost
}

func ExplainRisk(metrics Metrics, max maxMetrics) []string {
	reasons := []string{}
	if metrics.CommitCount >= 5 || normalize(metrics.CommitCount, max.CommitCount) >= 60 {
		reasons = append(reasons, "Frequently changed file")
	}
	if metrics.TotalChurn >= 300 || normalize(metrics.TotalChurn, max.TotalChurn) >= 60 {
		reasons = append(reasons, "High churn in recent commits")
	}
	if metrics.CyclomaticComplexityEstimate >= 20 || normalize(metrics.CyclomaticComplexityEstimate, max.Complexity) >= 60 {
		reasons = append(reasons, "High complexity")
	}
	if metrics.BugfixCommitCount >= 2 || normalize(metrics.BugfixCommitCount, max.BugfixCommitCount) >= 60 {
		reasons = append(reasons, "Often touched in bugfix commits")
	}
	if metrics.TodoCount >= 3 || normalize(metrics.TodoCount, max.TodoCount) >= 60 {
		reasons = append(reasons, "Many TODO/FIXME comments")
	}
	if metrics.AuthorCount >= 3 {
		reasons = append(reasons, "Multiple contributors touched this file")
	}
	if metrics.LOC >= 500 {
		reasons = append(reasons, "Large file")
	}
	if !metrics.HasTests && metrics.CyclomaticComplexityEstimate >= 10 {
		reasons = append(reasons, "No matching test file detected")
	}
	if metrics.DuplicateImportCount > 0 {
		reasons = append(reasons, "Duplicate imports detected")
	}
	if len(reasons) == 0 {
		reasons = append(reasons, "Low recent change and complexity signals")
	}
	return reasons
}

func BuildSuggestedTask(h Hotspot, daysOverride ...int) SuggestedTask {
	days := 90
	if len(daysOverride) > 0 && daysOverride[0] > 0 {
		days = daysOverride[0]
	}
	actions := suggestedActions(h)
	mlDetails := "ML calibration: disabled"
	if h.MLPrediction != nil && h.MLPrediction.Enabled {
		features := "none"
		if len(h.MLPrediction.ImportantFeatures) > 0 {
			features = strings.Join(h.MLPrediction.ImportantFeatures, ", ")
		}
		mlDetails = fmt.Sprintf("Heuristic score: %d\nML risk probability: %.0f%%\nML risk label: %s\nML score adjustment: %+d\nML important features: %s",
			h.HeuristicScore,
			h.MLPrediction.RiskProbability*100,
			h.MLPrediction.RiskLabel,
			h.MLPrediction.ScoreAdjustment,
			features,
		)
	}
	description := fmt.Sprintf(`Debt score: %d
Risk level: %s
File: %s

%s

Why this is risky:
- %s

Metrics:
- Last %d day commit count: %d
- Churn: +%d / -%d (%d total)
- Complexity estimate: %d
- TODO/FIXME/HACK/XXX count: %d
- Bugfix commit count: %d
- Estimated monthly cost: $%.2f

Suggested actions:
- %s`,
		h.DebtScore,
		h.Level,
		h.File,
		mlDetails,
		strings.Join(h.Reasons, "\n- "),
		days,
		h.Metrics.CommitCount,
		h.Metrics.ChurnAdded,
		h.Metrics.ChurnDeleted,
		h.Metrics.TotalChurn,
		h.Metrics.CyclomaticComplexityEstimate,
		h.Metrics.TodoCount,
		h.Metrics.BugfixCommitCount,
		h.EstimatedMonthlyCost,
		strings.Join(actions, "\n- "),
	)

	return SuggestedTask{
		Title:       "Refactor hotspot: " + h.File,
		Description: description,
		Actions:     actions,
	}
}

func suggestedActions(h Hotspot) []string {
	actions := []string{
		"Split large functions into smaller units",
		"Add focused test coverage around the riskiest behavior",
		"Convert TODO/FIXME comments into tracked issues",
	}
	if h.Metrics.CommitCount >= 5 {
		actions = append(actions, "Move frequently changed logic into a dedicated module")
	}
	if h.Metrics.CyclomaticComplexityEstimate >= 20 {
		actions = append(actions, "Simplify branching and error handling")
	}
	if h.Metrics.DuplicateImportCount > 0 || h.Metrics.TotalChurn >= 300 {
		actions = append(actions, "Remove duplicate or repeated code paths")
	}
	return actions
}

func normalize(value, max int) float64 {
	if value <= 0 || max <= 0 {
		return 0
	}
	return math.Min(100, (float64(value)/float64(max))*100)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func roundMoney(value float64) float64 {
	return math.Round(value*100) / 100
}
