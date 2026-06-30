package debt

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

var bugfixTerms = []string{
	"fix",
	"bug",
	"hotfix",
	"patch",
	"regression",
	"broken",
	"error",
	"issue",
}

func IsBugfixCommit(message string) bool {
	lower := strings.ToLower(message)
	for _, term := range bugfixTerms {
		if strings.Contains(lower, term) {
			return true
		}
	}
	return false
}

func ParseGitLogNumstat(r io.Reader) ([]CommitChange, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)

	var commits []CommitChange
	var current *CommitChange

	flush := func() {
		if current != nil {
			commits = append(commits, *current)
			current = nil
		}
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.HasPrefix(line, "--COMMIT--") {
			flush()
			payload := strings.TrimPrefix(line, "--COMMIT--")
			parts := strings.SplitN(payload, "\t", 5)
			current = &CommitChange{}
			if len(parts) > 0 {
				current.SHA = parts[0]
			}
			if len(parts) > 1 {
				current.AuthorName = parts[1]
			}
			if len(parts) > 2 {
				current.AuthorEmail = parts[2]
			}
			if len(parts) > 3 {
				if ts, err := time.Parse(time.RFC3339, parts[3]); err == nil {
					current.Date = ts
				}
			}
			if len(parts) > 4 {
				current.Message = parts[4]
			}
			continue
		}

		if current == nil {
			continue
		}

		fields := strings.SplitN(line, "\t", 3)
		if len(fields) != 3 {
			continue
		}

		added := parseNumstatNumber(fields[0])
		deleted := parseNumstatNumber(fields[1])
		path := normalizeNumstatPath(fields[2])
		if path == "" {
			continue
		}

		current.Files = append(current.Files, FileChange{
			Path:    path,
			Added:   added,
			Deleted: deleted,
		})
	}
	flush()

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return commits, nil
}

func parseNumstatNumber(value string) int {
	if value == "-" {
		return 0
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func normalizeNumstatPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.ReplaceAll(path, "\\", "/")

	if strings.Contains(path, " => ") {
		if open := strings.Index(path, "{"); open >= 0 {
			if close := strings.Index(path[open:], "}"); close >= 0 {
				inside := path[open+1 : open+close]
				parts := strings.SplitN(inside, " => ", 2)
				if len(parts) == 2 {
					path = path[:open] + parts[1] + path[open+close+1:]
				}
			}
		} else {
			parts := strings.SplitN(path, " => ", 2)
			if len(parts) == 2 {
				path = parts[1]
			}
		}
	}

	return strings.Trim(path, "\"")
}
