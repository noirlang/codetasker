package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/client"
	"github.com/codetasker/backend/internal/cli/ui"
	"github.com/codetasker/backend/internal/domain"
)

var repoCmd = &cobra.Command{
	Use:   "repo",
	Short: "Manage and sync repositories",
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all synchronized repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		api := client.NewClient(cfg)
		ctx := context.Background()

		repos, err := api.ListRepos(ctx)
		if err != nil {
			return err
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(repos)
		}

		if len(repos) == 0 {
			fmt.Println(ui.WarningStyle.Render("No repositories found."))
			fmt.Println("Use 'codetasker repo sync <owner/repo>' to sync your first repository.")
			return nil
		}

		fmt.Printf("%s\n\n", ui.HeaderStyle.Render("Synchronized Repositories"))

		var rows [][]string
		for _, r := range repos {
			updated := "-"
			if !r.UpdatedAt.IsZero() {
				updated = r.UpdatedAt.Format("2006-01-02 15:04:05")
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", r.ID),
				ui.BoldStyle.Render(r.FullName),
				r.DefaultBranch,
				updated,
			})
		}

		fmt.Println(ui.RenderTable([]string{"ID", "REPOSITORY", "DEFAULT BRANCH", "UPDATED AT"}, rows))
		return nil
	},
}

var repoSyncCmd = &cobra.Command{
	Use:   "sync <owner/repo>",
	Short: "Trigger a full codebase task scan and synchronization for a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format, expected 'owner/repo' (got %s)", target)
		}
		owner, repo := parts[0], parts[1]

		fmt.Printf("%s Initiating synchronization for %s...\n", ui.InfoStyle.Render("⟳"), ui.BoldStyle.Render(target))

		api := client.NewClient(cfg)
		ctx := context.Background()

		resp, err := api.SyncRepo(ctx, owner, repo)
		if err != nil {
			return fmt.Errorf("sync repository: %w", err)
		}

		fmt.Println()
		fmt.Println(ui.SuccessStyle.Render("✓ Codebase successfully synchronized!"))
		fmt.Printf("  Scanned Files: %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d", resp.ScannedFiles)))
		fmt.Printf("  Total Tasks:   %s\n", ui.BoldStyle.Render(fmt.Sprintf("%d", resp.TotalTasks)))
		fmt.Printf("  New Tasks:     %s\n", ui.SuccessStyle.Render(fmt.Sprintf("%d", resp.NewTasks)))
		fmt.Printf("  Updated Tasks: %s\n", ui.InfoStyle.Render(fmt.Sprintf("%d", resp.UpdatedTasks)))
		fmt.Printf("  Removed Tasks: %s\n", ui.WarningStyle.Render(fmt.Sprintf("%d", resp.RemovedTasks)))
		return nil
	},
}

var repoTreeCmd = &cobra.Command{
	Use:   "tree <owner/repo>",
	Short: "Display the file tree structure of a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format, expected 'owner/repo' (got %s)", target)
		}
		owner, repo := parts[0], parts[1]
		branch, _ := cmd.Flags().GetString("branch")

		api := client.NewClient(cfg)
		ctx := context.Background()

		tree, err := api.GetRepoTree(ctx, owner, repo, branch)
		if err != nil {
			return fmt.Errorf("get repository tree: %w", err)
		}

		fmt.Printf("%s %s\n\n", ui.HeaderStyle.Render("Repository Tree:"), ui.BoldStyle.Render(target))
		for _, node := range tree {
			prefix := "📄 "
			if node.Type == "tree" {
				prefix = "📁 "
			}
			fmt.Printf("  %s%s\n", prefix, node.Path)
		}
		return nil
	},
}

var repoCollabCmd = &cobra.Command{
	Use:   "collab <owner/repo>",
	Short: "List collaborators and access roles for a repository",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		parts := strings.Split(target, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid repository format, expected 'owner/repo' (got %s)", target)
		}
		owner, repo := parts[0], parts[1]

		api := client.NewClient(cfg)
		ctx := context.Background()

		collabs, err := api.GetRepoCollaborators(ctx, owner, repo)
		if err != nil {
			return fmt.Errorf("get collaborators: %w", err)
		}

		if len(collabs) == 0 {
			fmt.Println(ui.WarningStyle.Render("No collaborators found."))
			return nil
		}

		fmt.Printf("%s %s\n\n", ui.HeaderStyle.Render("Collaborators for:"), ui.BoldStyle.Render(target))

		var rows [][]string
		for _, c := range collabs {
			roleBadge := string(c.Role)
			switch c.Role {
			case domain.RoleOwner:
				roleBadge = ui.SuccessStyle.Render("OWNER")
			case domain.RoleMaintainer:
				roleBadge = ui.InfoStyle.Render("MAINTAINER")
			case domain.RoleDeveloper:
				roleBadge = ui.WarningStyle.Render("DEVELOPER")
			case domain.RoleViewer:
				roleBadge = ui.SubtleStyle.Render("VIEWER")
			}
			rows = append(rows, []string{
				c.Username,
				roleBadge,
				c.CreatedAt.Format("2006-01-02 15:04"),
			})
		}

		fmt.Println(ui.RenderTable([]string{"USERNAME", "ROLE", "ADDED AT"}, rows))
		return nil
	},
}

func init() {
	repoListCmd.Flags().Bool("json", false, "Output results in JSON format")
	repoTreeCmd.Flags().StringP("branch", "b", "", "Branch name")

	repoCmd.AddCommand(repoListCmd)
	repoCmd.AddCommand(repoSyncCmd)
	repoCmd.AddCommand(repoTreeCmd)
	repoCmd.AddCommand(repoCollabCmd)
}
