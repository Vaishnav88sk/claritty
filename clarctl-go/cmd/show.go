// cmd/show.go — clarctl show <incident-id>
package cmd

import (
	"fmt"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show <incident-id>",
	Short: "Show detailed view of a specific incident",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id := args[0]
		r, err := store.GetByID(id)
		if err != nil {
			return fmt.Errorf("fetch incident: %w", err)
		}
		if r == nil {
			return fmt.Errorf("incident '%s' not found", id)
		}
		ui.PrintBanner()
		ui.PrintIncidentDetail(r)
		return nil
	},
}
