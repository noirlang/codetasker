package debt

import "time"

type Level string

const (
	LevelLow      Level = "LOW"
	LevelMedium   Level = "MEDIUM"
	LevelHigh     Level = "HIGH"
	LevelCritical Level = "CRITICAL"
)

type Options struct {
	Repo       string
	Days       int
	HourlyCost float64
	Now        time.Time
}

type AnalysisResult struct {
	Repo       string    `json:"repo"`
	AnalyzedAt time.Time `json:"analyzed_at"`
	Days       int       `json:"days"`
	Summary    Summary   `json:"summary"`
	Hotspots   []Hotspot `json:"hotspots"`
}

type Summary struct {
	FilesAnalyzed        int     `json:"files_analyzed"`
	Critical             int     `json:"critical"`
	High                 int     `json:"high"`
	Medium               int     `json:"medium"`
	Low                  int     `json:"low"`
	EstimatedMonthlyCost float64 `json:"estimated_monthly_cost"`
}

type Hotspot struct {
	File                 string          `json:"file"`
	DebtScore            int             `json:"debt_score"`
	Level                Level           `json:"level"`
	Metrics              Metrics         `json:"metrics"`
	EstimatedMonthlyCost float64         `json:"estimated_monthly_cost"`
	Reasons              []string        `json:"reasons"`
	SuggestedTasks       []SuggestedTask `json:"suggested_tasks"`
}

type Metrics struct {
	CommitCount                  int        `json:"commit_count"`
	ChurnAdded                   int        `json:"churn_added"`
	ChurnDeleted                 int        `json:"churn_deleted"`
	TotalChurn                   int        `json:"total_churn"`
	AuthorCount                  int        `json:"author_count"`
	LastTouchedAt                *time.Time `json:"last_touched_at,omitempty"`
	BugfixCommitCount            int        `json:"bugfix_commit_count"`
	LOC                          int        `json:"loc"`
	FunctionCount                int        `json:"function_count"`
	AvgFunctionLength            float64    `json:"avg_function_length"`
	MaxFunctionLength            int        `json:"max_function_length"`
	NestingDepthEstimate         int        `json:"nesting_depth_estimate"`
	CyclomaticComplexityEstimate int        `json:"cyclomatic_complexity_estimate"`
	TodoCount                    int        `json:"todo_count"`
	DuplicateImportCount         int        `json:"duplicate_import_count"`
	HasTests                     bool       `json:"has_tests"`
	CoverageStatus               string     `json:"coverage_status"`
}

type SuggestedTask struct {
	ID            string   `json:"id,omitempty"`
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	Actions       []string `json:"actions"`
	Status        string   `json:"status,omitempty"`
	CreatedTaskID string   `json:"created_task_id,omitempty"`
}

type CommitChange struct {
	SHA         string
	Message     string
	AuthorName  string
	AuthorEmail string
	Date        time.Time
	Files       []FileChange
}

type FileChange struct {
	Path    string
	Added   int
	Deleted int
}

type SourceFile struct {
	Path    string
	Content string
}
