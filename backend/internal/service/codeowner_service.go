// Package service implements the business logic layer of CodeTasker.
// codeowner_service.go resolves the responsible maintainer for a given file path
// by parsing the repository's .github/CODEOWNERS file via the GitHub API.
package service

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/codetasker/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"
)

// CodeOwnerService fetches and evaluates CODEOWNERS rules for a repository.
type CodeOwnerService struct {
	githubService *GithubService
	userRepo      *repository.UserRepository
	log           *zap.Logger
}

// NewCodeOwnerService constructs a CodeOwnerService.
func NewCodeOwnerService(githubService *GithubService, userRepo *repository.UserRepository, log *zap.Logger) *CodeOwnerService {
	return &CodeOwnerService{
		githubService: githubService,
		userRepo:      userRepo,
		log:           log,
	}
}

// codeOwnerEntry represents a single line in a CODEOWNERS file.
type codeOwnerEntry struct {
	pattern string
	owners  []string
}

// ResolveMaintainer fetches the .github/CODEOWNERS file for a repository and
// returns the username and email of the first registered CodeTasker user that
// matches the given file path. Returns empty strings if no match is found.
func (s *CodeOwnerService) ResolveMaintainer(ctx context.Context, userID primitive.ObjectID, owner, repo, filePath string) (username, email string) {
	// Try multiple CODEOWNERS locations in priority order.
	paths := []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}
	var rawContent string
	for _, p := range paths {
		content, err := s.githubService.GetContents(ctx, userID, owner, repo, p, "")
		if err == nil && content != "" {
			rawContent = content
			break
		}
	}

	if rawContent == "" {
		s.log.Debug("CODEOWNERS not found", zap.String("repo", owner+"/"+repo))
		return "", ""
	}

	entries := parseCODEOWNERS(rawContent)
	matchedOwners := resolveOwners(filePath, entries)

	// Walk through matched owners and return the first one registered in CodeTasker.
	for _, ownerLogin := range matchedOwners {
		user, err := s.userRepo.FindByUsername(ctx, ownerLogin)
		if err != nil || user == nil {
			s.log.Debug("CODEOWNERS owner not in CodeTasker", zap.String("owner", ownerLogin))
			continue
		}
		s.log.Info("maintainer resolved via CODEOWNERS",
			zap.String("file", filePath),
			zap.String("maintainer", user.Username),
		)
		return user.Username, user.Email
	}

	return "", ""
}

// parseCODEOWNERS parses the content of a CODEOWNERS file into a slice of entries.
// Lines starting with '#' are comments and are skipped.
// Each non-blank line has the format: <pattern> <@owner1> <@owner2> ...
func parseCODEOWNERS(content string) []codeOwnerEntry {
	var entries []codeOwnerEntry
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pattern := fields[0]
		var owners []string
		for _, f := range fields[1:] {
			// Strip the leading @ if present.
			ownerLogin := strings.TrimPrefix(f, "@")
			if ownerLogin != "" {
				owners = append(owners, ownerLogin)
			}
		}
		if len(owners) > 0 {
			entries = append(entries, codeOwnerEntry{pattern: pattern, owners: owners})
		}
	}
	return entries
}

// resolveOwners evaluates the CODEOWNERS entries for a given file path and
// returns the owners from the LAST matching rule (GitHub's tie-breaking rule).
func resolveOwners(filePath string, entries []codeOwnerEntry) []string {
	var matched []string
	for _, entry := range entries {
		if matchesPattern(filePath, entry.pattern) {
			matched = entry.owners // overwrite — last match wins
		}
	}
	return matched
}

// matchesPattern checks whether a file path matches a CODEOWNERS glob pattern.
// Supports:
//   - Exact file match: "README.md"
//   - Directory prefix with trailing slash: "backend/" matches "backend/foo.go"
//   - Leading star wildcard: "*.go" matches any .go file at any level
//   - Catch-all: "*" matches everything
func matchesPattern(filePath, pattern string) bool {
	// "*" catches everything.
	if pattern == "*" {
		return true
	}

	// Trailing slash means "match this directory and everything beneath it".
	if strings.HasSuffix(pattern, "/") {
		dir := strings.TrimSuffix(pattern, "/")
		// Remove leading slash if present.
		dir = strings.TrimPrefix(dir, "/")
		return strings.HasPrefix(filePath, dir+"/") || filePath == dir
	}

	// Remove leading slash for matching.
	cleanPattern := strings.TrimPrefix(pattern, "/")

	// Try exact match first.
	if filePath == cleanPattern {
		return true
	}

	// Try filepath.Match for glob patterns.
	matched, err := filepath.Match(cleanPattern, filePath)
	if err == nil && matched {
		return true
	}

	// Try matching just the filename part.
	fileName := filepath.Base(filePath)
	matched, err = filepath.Match(cleanPattern, fileName)
	if err == nil && matched {
		return true
	}

	// Try matching against each directory level sub-path.
	parts := strings.Split(filePath, "/")
	for i := range parts {
		subPath := strings.Join(parts[i:], "/")
		matched, err = filepath.Match(cleanPattern, subPath)
		if err == nil && matched {
			return true
		}
	}

	return false
}
