package debt

import "testing"

func TestDefaultMLModelPredictsHigherRiskForHotspotMetrics(t *testing.T) {
	model, err := DefaultMLModel()
	if err != nil {
		t.Fatalf("DefaultMLModel returned error: %v", err)
	}

	low := model.Predict(Metrics{
		CommitCount:                  1,
		TotalChurn:                   40,
		CyclomaticComplexityEstimate: 5,
		LOC:                          90,
		BugfixCommitCount:            0,
		TodoCount:                    0,
		AuthorCount:                  1,
		HasTests:                     true,
	}, 15)

	high := model.Predict(Metrics{
		CommitCount:                  22,
		TotalChurn:                   2100,
		CyclomaticComplexityEstimate: 180,
		LOC:                          1400,
		BugfixCommitCount:            8,
		TodoCount:                    12,
		AuthorCount:                  7,
		HasTests:                     false,
	}, 80)

	if high.RiskProbability <= low.RiskProbability {
		t.Fatalf("high risk probability %.3f should be greater than low %.3f", high.RiskProbability, low.RiskProbability)
	}
	if high.RiskLabel != "HIGH" {
		t.Fatalf("high risk label = %s, want HIGH", high.RiskLabel)
	}
	if len(high.ImportantFeatures) == 0 {
		t.Fatalf("expected important ML features")
	}
}

func TestBlendMLScore(t *testing.T) {
	if got := BlendMLScore(80, 100); got != 84 {
		t.Fatalf("BlendMLScore(80, 100) = %d, want 84", got)
	}
	if got := BlendMLScore(10, 0); got != 8 {
		t.Fatalf("BlendMLScore(10, 0) = %d, want 8", got)
	}
}
