package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/config"
	"github.com/codetasker/backend/internal/cli/ui"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify CodeTasker CLI configuration",
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configuration keys and values",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, _ := config.ConfigPath()
		fmt.Printf("%s (%s)\n\n", ui.HeaderStyle.Render("CodeTasker Configuration"), ui.SubtleStyle.Render(cfgPath))
		fmt.Printf("  server_url:          %s\n", cfg.ServerURL)
		tokenMask := "(not set)"
		if cfg.Token != "" {
			tokenMask = cfg.Token[:min(len(cfg.Token), 8)] + "..."
		}
		fmt.Printf("  token:               %s\n", tokenMask)
		fmt.Printf("  default_repo:        %s\n", cfg.DefaultRepo)
		fmt.Printf("  default_branch:      %s\n", cfg.DefaultBranch)
		fmt.Printf("  default_days:        %d\n", cfg.DefaultDays)
		fmt.Printf("  default_hourly_cost: $%.2f\n", cfg.DefaultHourlyCost)
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get the value of a configuration key",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		key := args[0]
		switch key {
		case "server_url":
			fmt.Println(cfg.ServerURL)
		case "token":
			fmt.Println(cfg.Token)
		case "default_repo":
			fmt.Println(cfg.DefaultRepo)
		case "default_branch":
			fmt.Println(cfg.DefaultBranch)
		case "default_days":
			fmt.Println(cfg.DefaultDays)
		case "default_hourly_cost":
			fmt.Println(cfg.DefaultHourlyCost)
		default:
			fmt.Println(ui.ErrorStyle.Render("Unknown configuration key: " + key))
		}
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set the value of a configuration key",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		val := args[1]

		switch key {
		case "server_url":
			cfg.ServerURL = val
		case "token":
			cfg.Token = val
		case "default_repo":
			cfg.DefaultRepo = val
		case "default_branch":
			cfg.DefaultBranch = val
		case "default_days":
			days, err := strconv.Atoi(val)
			if err != nil {
				return fmt.Errorf("invalid integer for default_days: %w", err)
			}
			cfg.DefaultDays = days
		case "default_hourly_cost":
			cost, err := strconv.ParseFloat(val, 64)
			if err != nil {
				return fmt.Errorf("invalid float for default_hourly_cost: %w", err)
			}
			cfg.DefaultHourlyCost = cost
		default:
			return fmt.Errorf("unknown configuration key: %s", key)
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("%s Set %s = %s\n", ui.SuccessStyle.Render("✓"), ui.BoldStyle.Render(key), val)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configSetCmd)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
