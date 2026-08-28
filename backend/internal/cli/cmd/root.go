package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/config"
	"github.com/codetasker/backend/internal/cli/ui"
)

var (
	cfgFile string
	cfg     *config.Config
	version = "0.0.1"
)

// RootCmd is the base command for CodeTasker CLI.
var RootCmd = &cobra.Command{
	Use:   "codetasker",
	Short: "CodeTasker — Technical Debt & Code Annotation Management CLI",
	Long: `CodeTasker CLI brings complete automated technical debt management,
TODO/FIXME annotation tracking, and GitHub pull request injection directly to your terminal.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Do not print banner when json output is requested or during completion
		if cmd.Name() == "completion" || cmd.Name() == "__complete" {
			return
		}
		if jsonFlag, err := cmd.Flags().GetBool("json"); err == nil && jsonFlag {
			return
		}
		fmt.Println(ui.Banner())
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	cobra.OnInitialize(initConfig)

	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/codetasker/config.json)")
	RootCmd.PersistentFlags().StringP("server", "s", "", "CodeTasker backend server URL (e.g. http://localhost:8080)")
	RootCmd.PersistentFlags().StringP("token", "t", "", "Authentication token (JWT or App Token)")

	RootCmd.AddCommand(versionCmd)
	RootCmd.AddCommand(authCmd)
	RootCmd.AddCommand(configCmd)
	RootCmd.AddCommand(scanCmd)
	RootCmd.AddCommand(repoCmd)
	RootCmd.AddCommand(taskCmd)
	RootCmd.AddCommand(debtCmd)
	RootCmd.AddCommand(notifyCmd)
}

func initConfig() {
	var err error
	cfg, err = config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Override config with flags if provided
	if s, _ := RootCmd.PersistentFlags().GetString("server"); s != "" {
		cfg.ServerURL = s
	}
	if t, _ := RootCmd.PersistentFlags().GetString("token"); t != "" {
		cfg.Token = t
	}
}

// Execute runs the root CLI command.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CodeTasker CLI version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", ui.BoldStyle.Render("CodeTasker CLI"), ui.SuccessStyle.Render("v"+version))
	},
}
