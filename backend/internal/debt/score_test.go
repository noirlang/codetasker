package debt

import (
	"strings"
	"testing"
	"time"
)

func TestCalculateDebtScoreAndLevel(t *testing.T) {
	max := maxMetrics{
		CommitCount:       20,
		TotalChurn:        1000,
		Complexity:        50,
		LOC:               800,
		BugfixCommitCount: 5,
		TodoCount:         10,
		AuthorCount:       5,
	}
	metrics := Metrics{
		CommitCount:                  20,
		TotalChurn:                   1000,
		CyclomaticComplexityEstimate: 50,
		LOC:                          800,
		BugfixCommitCount:            5,
		TodoCount:                    10,
		AuthorCount:                  5,
	}

	score := CalculateDebtScore(metrics, max)
	if score != 100 {
		t.Fatalf("score = %d, want 100", score)
	}
	if level := LevelForScore(score); level != LevelCritical {
		t.Fatalf("level = %s, want %s", level, LevelCritical)
	}
}

func TestAnalyzeSnapshotProducesSortedHotspotWithWhy(t *testing.T) {
	touchedAt := time.Date(2026, 6, 10, 12, 0, 0, 0, time.UTC)
	commits := []CommitChange{
		{
			SHA:         "a",
			Message:     "fix broken checkout",
			AuthorEmail: "a@example.com",
			Date:        touchedAt,
			Files: []FileChange{
				{Path: "src/payment/checkout.ts", Added: 200, Deleted: 50},
			},
		},
		{
			SHA:         "b",
			Message:     "patch checkout error",
			AuthorEmail: "b@example.com",
			Date:        touchedAt.Add(time.Hour),
			Files: []FileChange{
				{Path: "src/payment/checkout.ts", Added: 100, Deleted: 25},
			},
		},
	}
	files := []SourceFile{
		{
			Path: "src/payment/checkout.ts",
			Content: strings.Repeat("if (x) {\n", 20) +
				"// TODO: simplify\n// FIXME: handle retry\n// HACK: temp\n" +
				strings.Repeat("}\n", 20),
		},
		{Path: "src/payment/checkout.test.ts", Content: "test('checkout', () => {})"},
	}

	result := AnalyzeSnapshot("repo", commits, files, []string{"src/payment/checkout.ts", "src/payment/checkout.test.ts"}, Options{
		Days:       90,
		HourlyCost: 35,
		Now:        touchedAt,
	})

	if result.Summary.FilesAnalyzed != 1 {
		t.Fatalf("files analyzed = %d, want 1", result.Summary.FilesAnalyzed)
	}
	if len(result.Hotspots) != 1 {
		t.Fatalf("hotspots = %d, want 1", len(result.Hotspots))
	}
	hotspot := result.Hotspots[0]
	if hotspot.File != "src/payment/checkout.ts" {
		t.Fatalf("unexpected hotspot file: %s", hotspot.File)
	}
	if hotspot.DebtScore < 70 {
		t.Fatalf("debt score = %d, want high risk", hotspot.DebtScore)
	}
	if len(hotspot.Reasons) == 0 {
		t.Fatalf("expected why reasons")
	}
	if len(hotspot.SuggestedTasks) != 1 {
		t.Fatalf("expected suggested task for high/critical hotspot")
	}
}

func TestBuildSuggestedTaskIncludesRequiredDetails(t *testing.T) {
	hotspot := Hotspot{
		File:                 "src/payment/checkout.ts",
		DebtScore:            88,
		Level:                LevelCritical,
		EstimatedMonthlyCost: 640,
		Reasons:              []string{"Frequently changed file", "High complexity"},
		Metrics: Metrics{
			CommitCount:                  24,
			ChurnAdded:                   800,
			ChurnDeleted:                 400,
			TotalChurn:                   1200,
			CyclomaticComplexityEstimate: 42,
			TodoCount:                    7,
			BugfixCommitCount:            6,
		},
	}

	task := BuildSuggestedTask(hotspot)
	if task.Title != "Refactor hotspot: src/payment/checkout.ts" {
		t.Fatalf("unexpected task title: %s", task.Title)
	}
	required := []string{
		"Debt score: 88",
		"Risk level: CRITICAL",
		"Last 90 day commit count: 24",
		"Churn: +800 / -400 (1200 total)",
		"Estimated monthly cost: $640.00",
	}
	for _, needle := range required {
		if !strings.Contains(task.Description, needle) {
			t.Fatalf("description missing %q:\n%s", needle, task.Description)
		}
	}
	if len(task.Actions) == 0 {
		t.Fatalf("expected suggested actions")
	}
}
