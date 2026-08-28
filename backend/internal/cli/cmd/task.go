package cmd

import (
	"bufio"
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

func parseLocationSpec(s, defaultNote string, defaultNewFile bool) (domain.TaskLocation, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return domain.TaskLocation{}, fmt.Errorf("empty location specification")
	}

	parts := strings.SplitN(s, ":", 3)
	filePath := strings.TrimSpace(parts[0])
	lineNum := 1
	isNew := defaultNewFile
	desc := defaultNote

	if len(parts) >= 2 {
		second := strings.TrimSpace(strings.ToLower(parts[1]))
		if second == "new" || second == "+" || second == "0" {
			isNew = true
			lineNum = 1
		} else if n, err := strconv.Atoi(second); err == nil && n > 0 {
			lineNum = n
		} else if len(parts) == 2 {
			desc = parts[1]
		}
	}

	if len(parts) >= 3 {
		desc = strings.TrimSpace(parts[2])
	}

	if desc == "" {
		desc = defaultNote
	}

	return domain.TaskLocation{
		FilePath:    filePath,
		LineNumber:  lineNum,
		Description: desc,
		IsNewFile:   isNew,
	}, nil
}

var taskInjectCmd = &cobra.Command{
	Use:   "inject",
	Short: "Inject single or multi-location TODO/FIXME tasks and create a GitHub PR",
	Long: `Inject single or multi-location code tasks directly into repository files and open an automated GitHub Pull Request.

Examples:
  # Single location in existing file:
  codetasker task inject --repo "owner/repo" --file "main.go" --line 42 --note "Refactor handler"

  # Create a brand new file with a TODO:
  codetasker task inject --repo "owner/repo" --file "pkg/auth/jwt.go" --new-file --note "Implement token refresh"

  # Inject across multiple lines in the same file:
  codetasker task inject --repo "owner/repo" --file "main.go" --lines "12,25,50" --note "Review bounds check"

  # Multi-location across multiple files in a single atomic PR:
  codetasker task inject --repo "owner/repo" \
    -L "src/main.go:42:Refactor handler" \
    -L "src/auth.go:15:Validate session token" \
    -L "pkg/scaffold.go:new:Initial scaffold module"

  # Interactive wizard:
  codetasker task inject -i`,
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		interactive, _ := cmd.Flags().GetBool("interactive")
		repoTarget, _ := cmd.Flags().GetString("repo")
		filePath, _ := cmd.Flags().GetString("file")
		lineNum, _ := cmd.Flags().GetInt("line")
		linesFlag, _ := cmd.Flags().GetString("lines")
		locationFlags, _ := cmd.Flags().GetStringArray("location")
		taskType, _ := cmd.Flags().GetString("type")
		note, _ := cmd.Flags().GetString("note")
		branch, _ := cmd.Flags().GetString("branch")
		isNewFile, _ := cmd.Flags().GetBool("new-file")
		issueURL, _ := cmd.Flags().GetString("issue-url")

		if taskType == "" {
			taskType = "TODO"
		}
		taskType = strings.ToUpper(strings.TrimSpace(taskType))

		// Interactive Builder Mode
		if interactive || (repoTarget == "" && filePath == "" && len(locationFlags) == 0 && linesFlag == "") {
			fmt.Println(ui.HeaderStyle.Render("CodeTasker Multi-Location Task Injector"))
			fmt.Println(ui.SubtleStyle.Render("Create single or multi-file code annotations and open an automated GitHub PR.\n"))

			if repoTarget == "" {
				defaultR := cfg.DefaultRepo
				prompt := "Target repository (owner/repo)"
				if defaultR != "" {
					prompt += fmt.Sprintf(" [%s]", defaultR)
				}
				fmt.Print(prompt + ": ")
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				if input != "" {
					repoTarget = input
				} else {
					repoTarget = defaultR
				}
			}

			if branch == "" {
				defaultB := cfg.DefaultBranch
				if defaultB == "" {
					defaultB = "main"
				}
				fmt.Printf("Base branch [%s]: ", defaultB)
				input, _ := reader.ReadString('\n')
				input = strings.TrimSpace(input)
				if input != "" {
					branch = input
				} else {
					branch = defaultB
				}
			}

			fmt.Printf("Annotation tag type [%s]: ", taskType)
			tInput, _ := reader.ReadString('\n')
			tInput = strings.TrimSpace(tInput)
			if tInput != "" {
				taskType = strings.ToUpper(tInput)
			}

			if issueURL == "" {
				fmt.Print("Linked GitHub Issue URL (optional): ")
				iInput, _ := reader.ReadString('\n')
				issueURL = strings.TrimSpace(iInput)
			}

			var interactiveLocations []domain.TaskLocation
			for {
				fmt.Printf("\n--- Location #%d ---\n", len(interactiveLocations)+1)
				fmt.Print("File path (e.g. src/auth.go): ")
				fPath, _ := reader.ReadString('\n')
				fPath = strings.TrimSpace(fPath)
				if fPath == "" {
					if len(interactiveLocations) > 0 {
						break
					}
					fmt.Println(ui.ErrorStyle.Render("File path cannot be empty."))
					continue
				}

				fmt.Print("Create as brand new file? (y/N): ")
				newChoice, _ := reader.ReadString('\n')
				newChoice = strings.TrimSpace(strings.ToLower(newChoice))
				locIsNew := newChoice == "y" || newChoice == "yes"

				locLine := 1
				if !locIsNew {
					fmt.Print("Line number for injection [1]: ")
					lStr, _ := reader.ReadString('\n')
					lStr = strings.TrimSpace(lStr)
					if lStr != "" {
						if n, err := strconv.Atoi(lStr); err == nil && n > 0 {
							locLine = n
						}
					}
				}

				fmt.Print("Task description / note: ")
				locDesc, _ := reader.ReadString('\n')
				locDesc = strings.TrimSpace(locDesc)
				if locDesc == "" {
					locDesc = note
				}
				if locDesc == "" {
					locDesc = "Pending refactoring"
				}

				interactiveLocations = append(interactiveLocations, domain.TaskLocation{
					FilePath:    fPath,
					LineNumber:  locLine,
					Description: locDesc,
					IsNewFile:   locIsNew,
				})

				fmt.Print("Add another location to this PR? (y/N): ")
				more, _ := reader.ReadString('\n')
				more = strings.TrimSpace(strings.ToLower(more))
				if more != "y" && more != "yes" {
					break
				}
			}

			locationFlags = nil
			filePath = ""
			var builtLocations []domain.TaskLocation = interactiveLocations
			if len(builtLocations) == 0 {
				return fmt.Errorf("no locations specified")
			}

			return executeInject(repoTarget, branch, taskType, issueURL, builtLocations)
		}

		if repoTarget == "" {
			repoTarget = cfg.DefaultRepo
		}
		if repoTarget == "" {
			return fmt.Errorf("--repo (e.g. 'owner/repo') is required")
		}

		if branch == "" {
			branch = cfg.DefaultBranch
			if branch == "" {
				branch = "main"
			}
		}

		var locations []domain.TaskLocation

		if len(locationFlags) > 0 {
			for _, locStr := range locationFlags {
				loc, err := parseLocationSpec(locStr, note, isNewFile)
				if err != nil {
					return fmt.Errorf("parse --location %q: %w", locStr, err)
				}
				if loc.Description == "" {
					return fmt.Errorf("location %q requires a description or pass --note", locStr)
				}
				locations = append(locations, loc)
			}
		} else if filePath != "" {
			if note == "" {
				return fmt.Errorf("--note (task description) is required")
			}

			if linesFlag != "" {
				lineStrs := strings.Split(linesFlag, ",")
				for _, ls := range lineStrs {
					ls = strings.TrimSpace(ls)
					if ls == "" {
						continue
					}
					ln, err := strconv.Atoi(ls)
					if err != nil || ln <= 0 {
						ln = 1
					}
					locations = append(locations, domain.TaskLocation{
						FilePath:    filePath,
						LineNumber:  ln,
						Description: note,
						IsNewFile:   isNewFile,
					})
				}
			} else {
				if lineNum <= 0 {
					lineNum = 1
				}
				locations = append(locations, domain.TaskLocation{
					FilePath:    filePath,
					LineNumber:  lineNum,
					Description: note,
					IsNewFile:   isNewFile,
				})
			}
		} else {
			return fmt.Errorf("either --file or one or more -L/--location arguments are required")
		}

		return executeInject(repoTarget, branch, taskType, issueURL, locations)
	},
}

func executeInject(repoTarget, branch, taskType, issueURL string, locations []domain.TaskLocation) error {
	parts := strings.Split(repoTarget, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid --repo format %q, expected 'owner/repo'", repoTarget)
	}
	owner, repo := parts[0], parts[1]

	firstLoc := locations[0]
	req := domain.InjectTaskRequest{
		RepoOwner:   owner,
		RepoName:    repo,
		Branch:      branch,
		FilePath:    firstLoc.FilePath,
		LineNumber:  firstLoc.LineNumber,
		Type:        taskType,
		Description: firstLoc.Description,
		IssueURL:    issueURL,
		Locations:   locations,
	}

	fmt.Printf("%s Injecting %s across %d location(s) into %s (%s)...\n",
		ui.InfoStyle.Render("⟳"),
		ui.TaskTypeBadge(taskType),
		len(locations),
		ui.BoldStyle.Render(repoTarget),
		branch,
	)

	for _, loc := range locations {
		if loc.IsNewFile {
			fmt.Printf("  • %s [NEW FILE]: %s\n", ui.BoldStyle.Render(loc.FilePath), loc.Description)
		} else {
			fmt.Printf("  • %s:%d: %s\n", ui.BoldStyle.Render(loc.FilePath), loc.LineNumber, loc.Description)
		}
	}

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
	fmt.Printf("  Locations: %d\n", len(locations))
	if resp.TaskID != "" {
		fmt.Printf("  Task ID:   %s\n", resp.TaskID)
	}
	return nil
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
	taskListCmd.Flags().String("status", "", "Filter by status (open, in_progress, resolved)")
	taskListCmd.Flags().String("type", "", "Filter by type (TODO, FIXME, BUG, HACK, NOTE)")
	taskListCmd.Flags().Bool("json", false, "Output results in JSON format")

	taskInjectCmd.Flags().StringP("repo", "r", "", "Target repository (owner/repo)")
	taskInjectCmd.Flags().StringP("file", "f", "", "Target file path (e.g. src/auth.go)")
	taskInjectCmd.Flags().IntP("line", "l", 1, "Line number for single-line injection")
	taskInjectCmd.Flags().String("lines", "", "Comma-separated line numbers (e.g. '12,25,40')")
	taskInjectCmd.Flags().StringArrayP("location", "L", nil, "Multi-location target: 'file:line:note' or 'file:new:note' (can be repeated)")
	taskInjectCmd.Flags().String("type", "TODO", "Annotation type (TODO, FIXME, BUG, HACK, NOTE)")
	taskInjectCmd.Flags().StringP("note", "n", "", "Task description / note text")
	taskInjectCmd.Flags().StringP("branch", "b", "", "Git branch to create PR against")
	taskInjectCmd.Flags().Bool("new-file", false, "Create target file as a brand new file")
	taskInjectCmd.Flags().String("issue-url", "", "Link to an existing GitHub Issue URL")
	taskInjectCmd.Flags().BoolP("interactive", "i", false, "Interactive multi-location task builder wizard")

	taskUpdateCmd.Flags().String("status", "", "New task status (open, in_progress, resolved)")
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
