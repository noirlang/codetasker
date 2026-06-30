package debt

import (
	"strings"
	"testing"
	"time"
)

func TestParseGitLogNumstat(t *testing.T) {
	input := strings.NewReader(`--COMMIT--abc123	Alice	alice@example.com	2026-06-01T10:00:00Z	fix checkout regression
10	2	src/payment/checkout.ts
4	0	src/{old => new}/worker.go

--COMMIT--def456	Bob	bob@example.com	2026-06-02T11:00:00Z	add reporting
-	-	assets/logo.png
1	3	app/main.py
`)

	commits, err := ParseGitLogNumstat(input)
	if err != nil {
		t.Fatalf("ParseGitLogNumstat returned error: %v", err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	if commits[0].SHA != "abc123" || commits[0].AuthorEmail != "alice@example.com" {
		t.Fatalf("unexpected first commit metadata: %+v", commits[0])
	}
	if !commits[0].Date.Equal(time.Date(2026, 6, 1, 10, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected parsed date: %s", commits[0].Date)
	}
	if commits[0].Files[0].Path != "src/payment/checkout.ts" || commits[0].Files[0].Added != 10 || commits[0].Files[0].Deleted != 2 {
		t.Fatalf("unexpected first file change: %+v", commits[0].Files[0])
	}
	if commits[0].Files[1].Path != "src/new/worker.go" {
		t.Fatalf("rename path was not normalized: %q", commits[0].Files[1].Path)
	}
	if commits[1].Files[0].Added != 0 || commits[1].Files[0].Deleted != 0 {
		t.Fatalf("binary numstat should be zeroed: %+v", commits[1].Files[0])
	}
}

func TestIsBugfixCommit(t *testing.T) {
	cases := map[string]bool{
		"fix checkout regression":     true,
		"Hotfix broken deploy":        true,
		"patch issue with API errors": true,
		"add dashboard filters":       false,
	}
	for message, expected := range cases {
		if got := IsBugfixCommit(message); got != expected {
			t.Fatalf("IsBugfixCommit(%q) = %v, want %v", message, got, expected)
		}
	}
}
