package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/codetasker/backend/internal/cli/client"
	"github.com/codetasker/backend/internal/cli/ui"
)

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "View and manage notifications",
}

var notifyListCmd = &cobra.Command{
	Use:   "list",
	Short: "List notifications",
	RunE: func(cmd *cobra.Command, args []string) error {
		unreadOnly, _ := cmd.Flags().GetBool("unread")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		api := client.NewClient(cfg)
		ctx := context.Background()

		notifications, err := api.ListNotifications(ctx, unreadOnly)
		if err != nil {
			return fmt.Errorf("list notifications: %w", err)
		}

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(notifications)
		}

		if len(notifications) == 0 {
			fmt.Println(ui.SuccessStyle.Render("✓ No notifications found."))
			return nil
		}

		fmt.Printf("%s (%d items)\n\n", ui.HeaderStyle.Render("Notifications"), len(notifications))

		var rows [][]string
		for _, n := range notifications {
			statusBadge := ui.SubtleStyle.Render("READ")
			if !n.Read {
				statusBadge = ui.SuccessStyle.Render("NEW")
			}

			idStr := n.ID.Hex()
			if len(idStr) > 8 {
				idStr = idStr[:8]
			}

			rows = append(rows, []string{
				idStr,
				statusBadge,
				ui.BoldStyle.Render(n.Title),
				n.Message,
				n.CreatedAt.Format("2006-01-02 15:04"),
			})
		}

		fmt.Println(ui.RenderTable([]string{"ID", "STATUS", "TITLE", "MESSAGE", "DATE"}, rows))
		return nil
	},
}

var notifyReadCmd = &cobra.Command{
	Use:   "read <notification-id>",
	Short: "Mark a notification as read",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		api := client.NewClient(cfg)
		ctx := context.Background()

		if err := api.MarkNotificationRead(ctx, id); err != nil {
			return fmt.Errorf("mark notification read: %w", err)
		}

		fmt.Printf("%s Notification marked as read.\n", ui.SuccessStyle.Render("✓"))
		return nil
	},
}

var notifyReadAllCmd = &cobra.Command{
	Use:   "read-all",
	Short: "Mark all notifications as read",
	RunE: func(cmd *cobra.Command, args []string) error {
		api := client.NewClient(cfg)
		ctx := context.Background()

		if err := api.MarkAllNotificationsRead(ctx); err != nil {
			return fmt.Errorf("mark all read: %w", err)
		}

		fmt.Println(ui.SuccessStyle.Render("✓ All notifications marked as read."))
		return nil
	},
}

func init() {
	notifyListCmd.Flags().BoolP("unread", "u", false, "Show unread notifications only")
	notifyListCmd.Flags().Bool("json", false, "Output results in JSON format")

	notifyCmd.AddCommand(notifyListCmd)
	notifyCmd.AddCommand(notifyReadCmd)
	notifyCmd.AddCommand(notifyReadAllCmd)
}
