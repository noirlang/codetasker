package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/codetasker/backend/internal/domain"
)

func TestFlexibleTime_Unmarshal(t *testing.T) {
	tests := []struct {
		name     string
		jsonStr  string
		expected time.Time
	}{
		{
			name:     "RFC3339",
			jsonStr:  `{"due_date": "2026-11-20T15:04:05Z"}`,
			expected: time.Date(2026, 11, 20, 15, 4, 5, 0, time.UTC),
		},
		{
			name:     "YYYY-MM-DD",
			jsonStr:  `{"due_date": "2026-11-20"}`,
			expected: time.Date(2026, 11, 20, 23, 59, 59, 0, time.UTC),
		},
		{
			name:     "YYYY-MM-DD HH:MM:SS",
			jsonStr:  `{"due_date": "2026-11-20 10:00:00"}`,
			expected: time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req domain.InjectTaskRequest
			if err := json.Unmarshal([]byte(tt.jsonStr), &req); err != nil {
				t.Fatalf("unexpected unmarshal error: %v", err)
			}
			if req.DueDate == nil {
				t.Fatalf("expected non-nil DueDate")
			}
			got := req.DueDate.ToTime()
			if got.Year() != tt.expected.Year() || got.Month() != tt.expected.Month() || got.Day() != tt.expected.Day() {
				t.Errorf("expected date %v, got %v", tt.expected, got)
			}
		})
	}
}
