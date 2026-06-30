package debt

import (
	"embed"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
)

//go:embed mldata/training_seed.json
var mlDataFS embed.FS

const mlModelName = "codetasker-logistic-risk-v1"

var (
	defaultMLOnce  sync.Once
	defaultMLModel *MLModel
	defaultMLErr   error
)

type MLModel struct {
	name        string
	datasetName string
	caps        mlFeatureCaps
	weights     []float64
	bias        float64
}

type mlDataset struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	FeatureCaps mlFeatureCaps `json:"feature_caps"`
	Examples    []mlExample   `json:"examples"`
}

type mlFeatureCaps struct {
	CommitCount       float64 `json:"commit_count"`
	TotalChurn        float64 `json:"total_churn"`
	Complexity        float64 `json:"complexity"`
	LOC               float64 `json:"loc"`
	BugfixCommitCount float64 `json:"bugfix_commit_count"`
	TodoCount         float64 `json:"todo_count"`
	AuthorCount       float64 `json:"author_count"`
}

type mlExample struct {
	CommitCount       int  `json:"commit_count"`
	TotalChurn        int  `json:"total_churn"`
	Complexity        int  `json:"complexity"`
	LOC               int  `json:"loc"`
	BugfixCommitCount int  `json:"bugfix_commit_count"`
	TodoCount         int  `json:"todo_count"`
	AuthorCount       int  `json:"author_count"`
	HasTests          bool `json:"has_tests"`
	DefectProne       bool `json:"defect_prone"`
}

func DefaultMLModel() (*MLModel, error) {
	defaultMLOnce.Do(func() {
		data, err := mlDataFS.ReadFile("mldata/training_seed.json")
		if err != nil {
			defaultMLErr = fmt.Errorf("read embedded ML dataset: %w", err)
			return
		}

		var dataset mlDataset
		if err := json.Unmarshal(data, &dataset); err != nil {
			defaultMLErr = fmt.Errorf("decode embedded ML dataset: %w", err)
			return
		}

		defaultMLModel, defaultMLErr = TrainMLModel(dataset)
	})
	return defaultMLModel, defaultMLErr
}

func TrainMLModel(dataset mlDataset) (*MLModel, error) {
	if len(dataset.Examples) == 0 {
		return nil, fmt.Errorf("ML dataset has no examples")
	}
	caps := dataset.FeatureCaps
	fillDefaultCaps(&caps)

	model := &MLModel{
		name:        mlModelName,
		datasetName: dataset.Name,
		caps:        caps,
		weights:     make([]float64, len(mlFeatureNames())),
	}

	const epochs = 900
	const learningRate = 0.42
	const l2 = 0.001

	for epoch := 0; epoch < epochs; epoch++ {
		for _, example := range dataset.Examples {
			x := featureVectorFromExample(example, caps)
			y := 0.0
			if example.DefectProne {
				y = 1.0
			}

			prediction := sigmoid(model.bias + dot(model.weights, x))
			err := prediction - y
			model.bias -= learningRate * err
			for i := range model.weights {
				model.weights[i] -= learningRate * (err*x[i] + l2*model.weights[i])
			}
		}
	}

	return model, nil
}

func (m *MLModel) Predict(metrics Metrics, heuristicScore int) MLPrediction {
	x := featureVectorFromMetrics(metrics, m.caps)
	probability := sigmoid(m.bias + dot(m.weights, x))
	mlScore := int(math.Round(probability * 100))
	adjusted := BlendMLScore(heuristicScore, mlScore)

	label := "NORMAL"
	switch {
	case probability >= 0.75:
		label = "HIGH"
	case probability >= 0.55:
		label = "ELEVATED"
	}

	return MLPrediction{
		Enabled:           true,
		Model:             m.name,
		Dataset:           m.datasetName,
		RiskProbability:   roundProbability(probability),
		RiskLabel:         label,
		Confidence:        roundProbability(math.Abs(probability-0.5) * 2),
		ScoreAdjustment:   adjusted - heuristicScore,
		ImportantFeatures: importantMLFeatures(m.weights, x),
	}
}

func BlendMLScore(heuristicScore, mlScore int) int {
	blended := int(math.Round((float64(heuristicScore) * 0.80) + (float64(mlScore) * 0.20)))
	if blended < 0 {
		return 0
	}
	if blended > 100 {
		return 100
	}
	return blended
}

func mlFeatureNames() []string {
	return []string{
		"commit frequency",
		"recent churn",
		"complexity",
		"file size",
		"bugfix touches",
		"TODO density",
		"contributor spread",
		"missing tests",
	}
}

func featureVectorFromMetrics(metrics Metrics, caps mlFeatureCaps) []float64 {
	fillDefaultCaps(&caps)
	missingTests := 0.0
	if !metrics.HasTests {
		missingTests = 1.0
	}
	return []float64{
		capRatio(float64(metrics.CommitCount), caps.CommitCount),
		capRatio(float64(metrics.TotalChurn), caps.TotalChurn),
		capRatio(float64(metrics.CyclomaticComplexityEstimate), caps.Complexity),
		capRatio(float64(metrics.LOC), caps.LOC),
		capRatio(float64(metrics.BugfixCommitCount), caps.BugfixCommitCount),
		capRatio(float64(metrics.TodoCount), caps.TodoCount),
		capRatio(float64(metrics.AuthorCount), caps.AuthorCount),
		missingTests,
	}
}

func featureVectorFromExample(example mlExample, caps mlFeatureCaps) []float64 {
	return featureVectorFromMetrics(Metrics{
		CommitCount:                  example.CommitCount,
		TotalChurn:                   example.TotalChurn,
		CyclomaticComplexityEstimate: example.Complexity,
		LOC:                          example.LOC,
		BugfixCommitCount:            example.BugfixCommitCount,
		TodoCount:                    example.TodoCount,
		AuthorCount:                  example.AuthorCount,
		HasTests:                     example.HasTests,
	}, caps)
}

func importantMLFeatures(weights, features []float64) []string {
	names := mlFeatureNames()
	type item struct {
		name  string
		score float64
	}

	items := make([]item, 0, len(features))
	for i := range features {
		if i >= len(weights) || i >= len(names) {
			continue
		}
		contribution := weights[i] * features[i]
		if contribution <= 0.01 {
			continue
		}
		items = append(items, item{name: names[i], score: contribution})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	limit := 3
	if len(items) < limit {
		limit = len(items)
	}
	result := make([]string, 0, limit)
	for i := 0; i < limit; i++ {
		result = append(result, items[i].name)
	}
	return result
}

func fillDefaultCaps(caps *mlFeatureCaps) {
	if caps.CommitCount <= 0 {
		caps.CommitCount = 30
	}
	if caps.TotalChurn <= 0 {
		caps.TotalChurn = 2500
	}
	if caps.Complexity <= 0 {
		caps.Complexity = 220
	}
	if caps.LOC <= 0 {
		caps.LOC = 1600
	}
	if caps.BugfixCommitCount <= 0 {
		caps.BugfixCommitCount = 12
	}
	if caps.TodoCount <= 0 {
		caps.TodoCount = 20
	}
	if caps.AuthorCount <= 0 {
		caps.AuthorCount = 10
	}
}

func capRatio(value, cap float64) float64 {
	if value <= 0 || cap <= 0 {
		return 0
	}
	return math.Min(1, value/cap)
}

func dot(weights, x []float64) float64 {
	total := 0.0
	for i := range weights {
		if i >= len(x) {
			break
		}
		total += weights[i] * x[i]
	}
	return total
}

func sigmoid(value float64) float64 {
	if value < -40 {
		return 0
	}
	if value > 40 {
		return 1
	}
	return 1 / (1 + math.Exp(-value))
}

func roundProbability(value float64) float64 {
	return math.Round(value*1000) / 1000
}
