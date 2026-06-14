// cmd/status.go — clarctl status: shows cluster health snapshot.
package cmd

import (
	"fmt"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current cluster health snapshot",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.PrintBanner()

		snap, err := store.LatestSnapshot()
		if err != nil {
			return fmt.Errorf("fetch snapshot: %w", err)
		}
		ui.PrintClusterHealth(snap)

		openInc, err := store.GetIncidents("", "OPEN", 24, 5)
		if err != nil {
			return fmt.Errorf("fetch incidents: %w", err)
		}
		if len(openInc) > 0 {
			fmt.Println("\n⚠  Open Incidents:")
			ui.PrintIncidentsTable(openInc)
		} else {
			fmt.Println("\n✓ No open incidents.")
		}
		return nil
	},
}
