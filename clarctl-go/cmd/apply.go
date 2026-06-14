// cmd/apply.go — clarctl apply <incident-id>
package cmd

import (
	"fmt"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/spf13/cobra"
)

var applyDryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply <incident-id>",
	Short: "Apply the remediation plan for a specific incident",
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

		ui.PrintIncidentDetail(r)

		if len(r.RemediationPlan) == 0 {
			fmt.Println("No remediation steps available for this incident.")
			return nil
		}

		cfg.DryRun = applyDryRun
		action := ui.PromptRemediation(r)
		switch action {
		case "execute":
			cfg.DryRun = false
			applyRemediation(r, false)
		case "dry":
			applyRemediation(r, true)
		default:
			fmt.Println("Skipped — no changes made.")
		}
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", true, "Dry run mode (default: enabled)")
}
