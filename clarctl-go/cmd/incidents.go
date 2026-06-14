// cmd/incidents.go — clarctl incidents: view incident history.
package cmd

import (
	"fmt"
	"strings"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/spf13/cobra"
)

var (
	incSeverity string
	incStatus   string
	incHours    int
	incLimit    int
)

var incidentsCmd = &cobra.Command{
	Use:     "incidents",
	Aliases: []string{"inc", "ls"},
	Short:   "View incident history with optional filters",
	Example: `  clarctl incidents
  clarctl incidents --severity SEV1
  clarctl incidents --status OPEN --hours 48`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.PrintBanner()

		list, err := store.GetIncidents(
			strings.ToUpper(incSeverity),
			strings.ToUpper(incStatus),
			incHours, incLimit,
		)
		if err != nil {
			return fmt.Errorf("fetch incidents: %w", err)
		}
		ui.PrintIncidentsTable(list)

		avg, count, _ := store.MTTRStats()
		ui.PrintMTTRStats(avg, count)
		return nil
	},
}

func init() {
	incidentsCmd.Flags().StringVar(&incSeverity, "severity", "", "Filter by severity (SEV1, SEV2, SEV3, SEV4)")
	incidentsCmd.Flags().StringVar(&incStatus, "status", "", "Filter by status (OPEN, INVESTIGATING, MITIGATED, RESOLVED)")
	incidentsCmd.Flags().IntVar(&incHours, "hours", 24, "Look back N hours")
	incidentsCmd.Flags().IntVar(&incLimit, "limit", 20, "Maximum results to show")
}
