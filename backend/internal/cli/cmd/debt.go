package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/ui"
	"github.com/codetasker/backend/internal/debt"
)

var debtCmd = &cobra.Command{
	Use:   "debt",
	Short: "Analyze technical debt and calculate refactoring costs",
}

var debtAnalyzeCmd = &cobra.Command{
	Use:   "analyze [repo-path]",
	Short: "Run technical debt and code churn analysis on a local git repository",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		repoPath := "."
		if len(args) > 0 {
			repoPath = args[0]
		}

		days, _ := cmd.Flags().GetInt("days")
		if days <= 0 {
			days = cfg.DefaultDays
			if days <= 0 {
				days = 90
			}
		}

		hourlyCost, _ := cmd.Flags().GetFloat64("cost")
		if hourlyCost <= 0 {
			hourlyCost = cfg.DefaultHourlyCost
			if hourlyCost <= 0 {
				hourlyCost = 35.0
			}
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")

		if !jsonOutput {
			fmt.Printf("%s Analyzing technical debt for %s (%d days of history, $%.2f/hr)\n\n",
				ui.InfoStyle.Render("⟳"), ui.BoldStyle.Render(repoPath), days, hourlyCost)
		}

		ctx := context.Background()
		result, err := debt.AnalyzeLocalRepo(ctx, debt.Options{
			Repo:       repoPath,
			Days:       days,
			HourlyCost: hourlyCost,
			Now:        time.Now().UTC(),
		})
		if err != nil {
			return fmt.Errorf("debt analysis failed: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(result)
		}

		// Summary Banner
		fmt.Println(ui.HeaderStyle.Render("── Technical Debt Summary ────────────────────────────"))
		fmt.Printf("  Files Analyzed:      %d\n", result.Summary.FilesAnalyzed)
		fmt.Printf("  Estimated Monthly Cost: %s\n", ui.WarningStyle.Render(ui.FormatCost(result.Summary.EstimatedMonthlyCost)))
		fmt.Printf("  Risk Breakdown:      Critical: %s | High: %s | Medium: %s | Low: %s\n",
			ui.ErrorStyle.Render(fmt.Sprintf("%d", result.Summary.Critical)),
			ui.WarningStyle.Render(fmt.Sprintf("%d", result.Summary.High)),
			ui.InfoStyle.Render(fmt.Sprintf("%d", result.Summary.Medium)),
			ui.SuccessStyle.Render(fmt.Sprintf("%d", result.Summary.Low)),
		)
		fmt.Println()

		// Hotspot Table
		if len(result.Hotspots) > 0 {
			fmt.Println(ui.HeaderStyle.Render("── Top Technical Debt Hotspots ───────────────────────"))

			limit := 10
			if len(result.Hotspots) < limit {
				limit = len(result.Hotspots)
			}

			var rows [][]string
			for i := 0; i < limit; i++ {
				f := result.Hotspots[i]
				scoreStr := fmt.Sprintf("%d", f.DebtScore)
				if f.DebtScore > 50 {
					scoreStr = ui.ErrorStyle.Render(scoreStr)
				} else if f.DebtScore > 25 {
					scoreStr = ui.WarningStyle.Render(scoreStr)
				} else {
					scoreStr = ui.SuccessStyle.Render(scoreStr)
				}

				levelStr := string(f.Level)
				switch f.Level {
				case debt.LevelCritical:
					levelStr = ui.ErrorStyle.Render(levelStr)
				case debt.LevelHigh:
					levelStr = ui.WarningStyle.Render(levelStr)
				case debt.LevelMedium:
					levelStr = ui.InfoStyle.Render(levelStr)
				case debt.LevelLow:
					levelStr = ui.SuccessStyle.Render(levelStr)
				}

				rows = append(rows, []string{
					f.File,
					scoreStr,
					levelStr,
					ui.FormatCost(f.EstimatedMonthlyCost),
					fmt.Sprintf("%d", f.Metrics.TotalChurn),
					fmt.Sprintf("%d", f.Metrics.TodoCount),
					fmt.Sprintf("%d", f.Metrics.CyclomaticComplexityEstimate),
				})
			}

			fmt.Println(ui.RenderTable([]string{"FILE PATH", "DEBT SCORE", "LEVEL", "EST. COST", "CHURN", "TODOs", "COMPLEXITY"}, rows))
		}

		return nil
	},
}

func init() {
	debtAnalyzeCmd.Flags().IntP("days", "d", 90, "Number of days of git history to analyze")
	debtAnalyzeCmd.Flags().Float64P("cost", "c", 35.0, "Hourly engineer cost in USD")
	debtAnalyzeCmd.Flags().Bool("json", false, "Output results in JSON format")

	debtCmd.AddCommand(debtAnalyzeCmd)
}
