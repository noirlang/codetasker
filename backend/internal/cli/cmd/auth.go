package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/client"
	"github.com/codetasker/backend/internal/cli/config"
	"github.com/codetasker/backend/internal/cli/ui"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication and credentials",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in with a CodeTasker JWT or App Token",
	RunE: func(cmd *cobra.Command, args []string) error {
		tokenFlag, _ := cmd.Flags().GetString("token")
		serverFlag, _ := cmd.Flags().GetString("server")

		if serverFlag != "" {
			cfg.ServerURL = serverFlag
		}

		token := tokenFlag
		if token == "" {
			fmt.Printf("%s\n", ui.BoldStyle.Render("CodeTasker Authentication Setup"))
			fmt.Printf("Server URL [%s]: ", cfg.ServerURL)

			reader := bufio.NewReader(os.Stdin)
			serverInput, _ := reader.ReadString('\n')
			serverInput = strings.TrimSpace(serverInput)
			if serverInput != "" {
				cfg.ServerURL = serverInput
			}

			fmt.Print("Enter your CodeTasker token (or App Token): ")
			byteToken, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("read token input: %w", err)
			}
			token = strings.TrimSpace(string(byteToken))
		}

		if token == "" {
			return fmt.Errorf("token cannot be empty")
		}

		// Sanitize server URL and token against stray newlines or whitespace from terminal wraps
		cfg.ServerURL = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(cfg.ServerURL, "\n", ""), "\r", ""))
		cfg.ServerURL = strings.ReplaceAll(cfg.ServerURL, " ", "")
		cfg.Token = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(token, "\n", ""), "\r", ""))
		cfg.Token = strings.ReplaceAll(cfg.Token, " ", "")

		// Handle accidental duplicate paste of App Token (e.g. ct_app_...ct_app_...)
		if strings.HasPrefix(cfg.Token, "ct_app_") && len(cfg.Token) >= 71 {
			cfg.Token = cfg.Token[:71]
		}

		// Validate token with server
		api := client.NewClient(cfg)
		ctx := context.Background()
		user, err := api.GetMe(ctx)
		if err == nil && user != nil {
			fmt.Printf("%s Logged in as %s (@%s)\n", ui.SuccessStyle.Render("✓"), ui.BoldStyle.Render(user.Username), user.Username)
		} else {
			// If GetMe failed, check if App Token is valid via notifications
			if strings.HasPrefix(cfg.Token, "ct_app_") {
				if notifs, errNotif := api.ListNotifications(ctx, false); errNotif == nil {
					fmt.Printf("%s App Token authenticated successfully (%d notifications found).\n", ui.SuccessStyle.Render("✓"), len(notifs))
				} else {
					fmt.Printf("%s: Token validation warning: %v\n", ui.WarningStyle.Render("Note"), err)
				}
			} else {
				fmt.Printf("%s: Token validation warning: %v\n", ui.WarningStyle.Render("Note"), err)
			}
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println(ui.SuccessStyle.Render("Authentication credentials saved successfully."))
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display current authentication status",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cfg.Token == "" {
			fmt.Println(ui.WarningStyle.Render("Not logged in."))
			fmt.Println("Run 'codetasker auth login' to authenticate.")
			return nil
		}

		api := client.NewClient(cfg)
		ctx := context.Background()
		user, err := api.GetMe(ctx)
		if err != nil {
			if strings.HasPrefix(cfg.Token, "ct_app_") {
				if notifs, errNotif := api.ListNotifications(ctx, false); errNotif == nil {
					fmt.Printf("%s Connected to %s with App Token (%d notifications)\n", ui.SuccessStyle.Render("✓ Authenticated:"), cfg.ServerURL, len(notifs))
					return nil
				}
			}
			return fmt.Errorf("Connected to %s but token verification failed: %w", cfg.ServerURL, err)
		}

		fmt.Println(ui.HeaderStyle.Render("Authentication Status:"))
		fmt.Printf("  Server:    %s\n", ui.BoldStyle.Render(cfg.ServerURL))
		fmt.Printf("  User:      %s (@%s)\n", ui.SuccessStyle.Render(user.Username), user.Username)
		fmt.Printf("  GitHub ID: %d\n", user.GithubID)
		if user.Email != "" {
			fmt.Printf("  Email:     %s\n", user.Email)
		}
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored authentication token",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg.Token = ""
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println(ui.SuccessStyle.Render("Successfully logged out. Stored token cleared."))
		return nil
	},
}

func init() {
	authLoginCmd.Flags().StringP("token", "t", "", "Authentication token")
	authLoginCmd.Flags().StringP("server", "s", "", "Server URL")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
}
