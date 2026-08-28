package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/client"
	"github.com/codetasker/backend/internal/cli/ui"
	"github.com/codetasker/backend/internal/domain"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage, view, and inject code tasks & annotations",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks for a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		api := client.NewClient(cfg)
		ctx := context.Background()

		repoID, _ := cmd.Flags().GetInt64("repo-id")
		repoName, _ := cmd.Flags().GetString("repo")
		statusFilter, _ := cmd.Flags().GetString("status")
		typeFilter, _ := cmd.Flags().GetString("type")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		// If repoID not provided, try to resolve via repo name or default repo
		if repoID == 0 {
			if repoName == "" {
				repoName = cfg.DefaultRepo
			}
			if repoName != "" {
				repos, err := api.ListRepos(ctx)
				if err == nil {
					for _, r := range repos {
						if strings.EqualFold(r.FullName, repoName) {
							repoID = r.ID
							break
						}
					}
				}
			}
		}

		if repoID == 0 {
			// If still 0, try to list repositories and take the first one or ask
			repos, err := api.ListRepos(ctx)
			if err != nil {
				return fmt.Errorf("list repos: %w", err)
			}
			if len(repos) == 0 {
				fmt.Println(ui.WarningStyle.Render("No repositories found. Sync a repository first: 'codetasker repo sync <owner/repo>'"))
				return nil
			}
			repoID = repos[0].ID
			repoName = repos[0].FullName
		}

		tasks, err := api.ListTasks(ctx, repoID, statusFilter, typeFilter)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(tasks)
		}

		if len(tasks) == 0 {
			fmt.Printf("%s No tasks found for repository %s.\n", ui.InfoStyle.Render("ℹ"), ui.BoldStyle.Render(repoName))
			return nil
		}

		fmt.Printf("%s %s (%d tasks)\n\n", ui.HeaderStyle.Render("Code Tasks:"), ui.BoldStyle.Render(repoName), len(tasks))

		var rows [][]string
		for _, t := range tasks {
			assignee := "-"
			if t.AssigneeUsername != "" {
				assignee = "@" + t.AssigneeUsername
			}

			idStr := t.ID.Hex()
			if len(idStr) > 8 {
				idStr = idStr[:8]
			}

			rows = append(rows, []string{
				idStr,
				ui.TaskTypeBadge(t.Type),
				ui.StatusBadge(string(t.Status)),
				fmt.Sprintf("%s:%d", t.FilePath, t.LineNumber),
				assignee,
				t.Content,
			})
		}

		fmt.Println(ui.RenderTable([]string{"ID", "TYPE", "STATUS", "LOCATION", "ASSIGNEE", "CONTENT"}, rows))
		return nil
	},
}

var taskInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject a TODO/FIXME task into repository files and create a GitHub PR",
	RunE: func(cmd *cobra.Command, args []string) error {
		repoTarget, _ := cmd.Flags().GetString("repo")
		if repoTarget == "" {
			repoTarget = cfg.DefaultRepo
		}
		if repoTarget == "" {
			return fmt.Errorf("--repo (e.g. 'owner/repo') is required")
		}

		parts := strings.Split(repoTarget, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid --repo format, expected 'owner/repo'")
		}
		owner, repo := parts[0], parts[1]

		filePath, _ := cmd.Flags().GetString("file")
		if filePath == "" {
			return fmt.Errorf("--file path is required")
		}

		lineNum, _ := cmd.Flags().GetInt("line")
		if lineNum <= 0 {
			lineNum = 1
		}

		taskType, _ := cmd.Flags().GetString("type")
		if taskType == "" {
			taskType = "TODO"
		}

		note, _ := cmd.Flags().GetString("note")
		if note == "" {
			return fmt.Errorf("--note (task description) is required")
		}

		branch, _ := cmd.Flags().GetString("branch")
		if branch == "" {
			branch = cfg.DefaultBranch
			if branch == "" {
				branch = "main"
			}
		}

		isNewFile, _ := cmd.Flags().GetBool("new-file")
		issueURL, _ := cmd.Flags().GetString("issue-url")

		req := domain.InjectTaskRequest{
			RepoOwner:   owner,
			RepoName:    repo,
			Branch:      branch,
			FilePath:    filePath,
			LineNumber:  lineNum,
			Type:        taskType,
			Description: note,
			IssueURL:    issueURL,
			Locations: []domain.TaskLocation{
				{
					FilePath:    filePath,
					LineNumber:  lineNum,
					Description: note,
					IsNewFile:   isNewFile,
				},
			},
		}

		fmt.Printf("%s Injecting %s into %s:%d...\n", ui.InfoStyle.Render("⟳"), ui.TaskTypeBadge(taskType), filePath, lineNum)

		api := client.NewClient(cfg)
		ctx := context.Background()

		resp, err := api.InjectTask(ctx, req)
		if err != nil {
			return fmt.Errorf("inject task failed: %w", err)
		}

		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("✓ Task successfully injected and Pull Request opened!"))
		fmt.Printf("  PR URL:    %s\n", ui.BoldStyle.Render(resp.PRURL))
		fmt.Printf("  Commit:    %s\n", resp.CommitSHA)
		fmt.Printf("  Branch:    %s\n", resp.Branch)
		fmt.Printf("  Task ID:   %s\n", resp.TaskID)
		return nil
	},
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-id>",
	Short: "Update task status or assignee",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		statusStr, _ := cmd.Flags().GetString("status")
		assignee, _ := cmd.Flags().GetString("assign")

		api := client.NewClient(cfg)
		ctx := context.Background()

		if statusStr != "" {
			validStatuses := map[string]domain.TaskStatus{
				"open":        domain.TaskStatusOpen,
				"in_progress": domain.TaskStatusInProgress,
				"resolved":    domain.TaskStatusResolved,
			}
			st, ok := validStatuses[strings.ToLower(statusStr)]
			if !ok {
				return fmt.Errorf("invalid status '%s'. Allowed: open, in_progress, resolved", statusStr)
			}
			if err := api.UpdateTaskStatus(ctx, taskID, st); err != nil {
				return fmt.Errorf("update status: %w", err)
			}
			fmt.Printf("%s Task status updated to %s\n", ui.SuccessStyle.Render("✓"), ui.StatusBadge(string(st)))
		}

		if cmd.Flags().Changed("assign") {
			if err := api.UpdateTaskAssignee(ctx, taskID, assignee); err != nil {
				return fmt.Errorf("update assignee: %w", err)
			}
			if assignee == "" {
				fmt.Println(ui.SuccessStyle.Render("✓ Assignee cleared."))
			} else {
				fmt.Printf("%s Task assigned to @%s\n", ui.SuccessStyle.Render("✓"), assignee)
			}
		}

		return nil
	},
}

var taskCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Add or list comments on a task",
}

var taskCommentAddCmd = &cobra.Command{
	Use:   "add <task-id> <message>",
	Short: "Add a comment to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]
		content := args[1]

		api := client.NewClient(cfg)
		ctx := context.Background()

		comment, err := api.AddComment(ctx, taskID, content)
		if err != nil {
			return fmt.Errorf("add comment: %w", err)
		}

		fmt.Printf("%s Comment posted by @%s\n", ui.SuccessStyle.Render("✓"), comment.Username)
		return nil
	},
}

var taskCommentListCmd = &cobra.Command{
	Use:   "list <task-id>",
	Short: "List all comments on a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID := args[0]

		api := client.NewClient(cfg)
		ctx := context.Background()

		comments, err := api.ListComments(ctx, taskID)
		if err != nil {
			return fmt.Errorf("list comments: %w", err)
		}

		if len(comments) == 0 {
			fmt.Println(ui.InfoStyle.Render("No comments on this task yet."))
			return nil
		}

		fmt.Printf("%s (%d comments)\n\n", ui.HeaderStyle.Render("Task Comments"), len(comments))
		for _, c := range comments {
			fmt.Printf("%s %s:\n  %s\n\n", ui.BoldStyle.Render("@"+c.Username), ui.SubtleStyle.Render(c.CreatedAt.Format("2006-01-02 15:04")), c.Content)
		}
		return nil
	},
}

func init() {
	taskListCmd.Flags().Int64("repo-id", 0, "GitHub repository ID")
	taskListCmd.Flags().StringP("repo", "r", "", "Repository name (owner/repo)")
	taskListCmd.Flags().StringP("status", "s", "", "Filter by status (open, in_progress, resolved)")
	taskListCmd.Flags().StringP("type", "t", "", "Filter by type (TODO, FIXME, BUG, HACK, NOTE)")
	taskListCmd.Flags().Bool("json", false, "Output results in JSON format")

	taskInjectCmd.Flags().StringP("repo", "r", "", "Target repository (owner/repo)")
	taskInjectCmd.Flags().StringP("file", "f", "", "Target file path (e.g. src/auth.go)")
	taskInjectCmd.Flags().IntP("line", "l", 1, "Line number for injection")
	taskInjectCmd.Flags().StringP("type", "t", "TODO", "Annotation type (TODO, FIXME, BUG, HACK, NOTE)")
	taskInjectCmd.Flags().StringP("note", "n", "", "Task description / note text")
	taskInjectCmd.Flags().StringP("branch", "b", "", "Git branch to create PR against")
	taskInjectCmd.Flags().Bool("new-file", false, "Create as a brand new file")
	taskInjectCmd.Flags().String("issue-url", "", "Link to an existing GitHub Issue URL")

	taskUpdateCmd.Flags().StringP("status", "s", "", "New task status (open, in_progress, resolved)")
	taskUpdateCmd.Flags().StringP("assign", "a", "", "Assign to GitHub username (empty to unassign)")

	taskCommentCmd.AddCommand(taskCommentAddCmd)
	taskCommentCmd.AddCommand(taskCommentListCmd)

	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskInjectCmd)
	taskCmd.AddCommand(taskUpdateCmd)
	taskCmd.AddCommand(taskCommentCmd)
}

func parseIntOrDefault(s string, def int) int {
	val, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return val
}
