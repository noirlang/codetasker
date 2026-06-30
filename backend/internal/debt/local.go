package debt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

const maxSourceFileBytes = 1_000_000

func AnalyzeLocalRepo(ctx context.Context, opts Options) (AnalysisResult, error) {
	if opts.Repo == "" {
		opts.Repo = "."
	}
	if opts.Days <= 0 {
		opts.Days = 90
	}
	if opts.HourlyCost <= 0 {
		opts.HourlyCost = 35
	}
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}

	repoPath, err := filepath.Abs(opts.Repo)
	if err != nil {
		return AnalysisResult{}, fmt.Errorf("resolve repo path: %w", err)
	}

	commits, err := readLocalGitHistory(ctx, repoPath, opts.Days, opts.Now)
	if err != nil {
		return AnalysisResult{}, err
	}

	files, allPaths, err := readLocalSourceFiles(repoPath)
	if err != nil {
		return AnalysisResult{}, err
	}

	return AnalyzeSnapshot(repoPath, commits, files, allPaths, opts), nil
}

func readLocalGitHistory(ctx context.Context, repoPath string, days int, now time.Time) ([]CommitChange, error) {
	since := now.AddDate(0, 0, -days).Format("2006-01-02")
	args := []string{
		"-C", repoPath,
		"log",
		"--since=" + since,
		"--numstat",
		"--date=iso-strict",
		"--pretty=format:--COMMIT--%H%x09%an%x09%ae%x09%aI%x09%s",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log failed: %w: %s", err, stderr.String())
	}

	commits, err := ParseGitLogNumstat(&stdout)
	if err != nil {
		return nil, fmt.Errorf("parse git log: %w", err)
	}
	return commits, nil
}

func readLocalSourceFiles(repoPath string) ([]SourceFile, []string, error) {
	files := []SourceFile{}
	allPaths := []string{}

	err := filepath.WalkDir(repoPath, func(current string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(repoPath, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)

		if entry.IsDir() {
			if IsIgnoredPath(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if !SupportedPath(rel) || IsIgnoredPath(rel) {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxSourceFileBytes {
			return nil
		}

		allPaths = append(allPaths, rel)
		if IsTestPath(rel) {
			return nil
		}

		content, err := os.ReadFile(current)
		if err != nil {
			return err
		}
		files = append(files, SourceFile{
			Path:    rel,
			Content: string(content),
		})
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	return files, allPaths, nil
}
