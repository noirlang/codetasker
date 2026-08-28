package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/ui"
	"github.com/codetasker/backend/internal/parser"
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Scan a local codebase directory for TODO, FIXME, BUG, HACK, and NOTE annotations",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetDir := "."
		if len(args) > 0 {
			targetDir = args[0]
		}

		absPath, err := filepath.Abs(targetDir)
		if err != nil {
			return fmt.Errorf("resolve directory path: %w", err)
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")
		typeFilter, _ := cmd.Flags().GetString("type")

		if !jsonOutput {
			fmt.Printf("%s %s\n\n", ui.BoldStyle.Render("Scanning Codebase:"), ui.SubtleStyle.Render(absPath))
		}

		var fileContents []parser.FileContent
		ignoredDirs := map[string]bool{
			".git": true, "node_modules": true, "dist": true, "build": true,
			"vendor": true, ".gemini": true, ".idea": true, ".vscode": true,
			"target": true, "bin": true, "obj": true,
		}

		err = filepath.Walk(absPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if info.IsDir() {
				if ignoredDirs[info.Name()] {
					return filepath.SkipDir
				}
				return nil
			}

			// Skip files larger than 2MB
			if info.Size() > 2*1024*1024 {
				return nil
			}

			// Read file content
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}

			// Skip binary files (check for null byte in first 512 bytes)
			n := len(data)
			if n > 512 {
				n = 512
			}
			if strings.Contains(string(data[:n]), "\x00") {
				return nil
			}

			relPath, err := filepath.Rel(absPath, path)
			if err != nil {
				relPath = path
			}

			fileContents = append(fileContents, parser.FileContent{
				Path:    relPath,
				Content: string(data),
			})
			return nil
		})
		if err != nil {
			return fmt.Errorf("walk directory: %w", err)
		}

		parsedTasks := parser.ParseFiles(fileContents, 0)

		// Filter by type if requested
		if typeFilter != "" {
			var filtered []parser.ParsedTask
			for _, t := range parsedTasks {
				if strings.EqualFold(t.Type, typeFilter) {
					filtered = append(filtered, t)
				}
			}
			parsedTasks = filtered
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(parsedTasks)
		}

		if len(parsedTasks) == 0 {
			fmt.Println(ui.SuccessStyle.Render("✓ No code annotations found in this directory."))
			return nil
		}

		var rows [][]string
		counts := make(map[string]int)

		for _, t := range parsedTasks {
			counts[strings.ToUpper(t.Type)]++
			rows = append(rows, []string{
				ui.TaskTypeBadge(t.Type),
				fmt.Sprintf("%s:%d", t.FilePath, t.LineNumber),
				t.Content,
			})
		}

		fmt.Println(ui.RenderTable([]string{"TYPE", "LOCATION", "ANNOTATION NOTE"}, rows))

		fmt.Println()
		fmt.Printf("%s %d annotations found across %d files\n", ui.BoldStyle.Render("Summary:"), len(parsedTasks), len(fileContents))
		summaryParts := []string{}
		for _, k := range []string{"TODO", "FIXME", "BUG", "HACK", "NOTE"} {
			if c, ok := counts[k]; ok && c > 0 {
				summaryParts = append(summaryParts, fmt.Sprintf("%s: %d", ui.TaskTypeBadge(k), c))
			}
		}
		fmt.Printf("Breakdown: %s\n", strings.Join(summaryParts, "  "))

		return nil
	},
}

func init() {
	scanCmd.Flags().Bool("json", false, "Output results in JSON format")
	scanCmd.Flags().String("type", "", "Filter annotations by type (TODO, FIXME, BUG, HACK, NOTE)")
}
