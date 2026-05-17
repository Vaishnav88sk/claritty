// cmd/report.go — clarctl report <incident-id>
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var reportOutput string

var reportCmd = &cobra.Command{
	Use:   "report <incident-id>",
	Short: "Export a full incident report as JSON",
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

		out, err := json.MarshalIndent(r, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal report: %w", err)
		}

		if reportOutput != "" {
			if err := os.WriteFile(reportOutput, out, 0644); err != nil {
				return fmt.Errorf("write file: %w", err)
			}
			fmt.Printf("Report saved to %s\n", reportOutput)
		} else {
			fmt.Println(string(out))
		}
		return nil
	},
}

func init() {
	reportCmd.Flags().StringVarP(&reportOutput, "output", "o", "", "Output file path (default: stdout)")
}
