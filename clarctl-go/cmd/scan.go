// cmd/scan.go — clarctl scan command.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ai"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/runbooks"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
	"github.com/spf13/cobra"
)

var (
	scanApply  bool
	scanDryRun bool
)

var scanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Run a single AI-SRE scan across the cluster",
	Long: `Runs a full 6-stage AI-SRE scan across your Kubernetes cluster:
  1. Triage      — cluster-wide pod/event/node health assessment
  2. Metrics     — CPU, memory, restart rate analysis
  3. Logs        — error pattern mining from affected pods
  4. Infra       — K8s resource quota, PVC, HPA diagnosis
  5. Runbook     — remediation plan selection
  6. Commander   — final structured incident report`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.PrintBanner()

		cfg.DryRun = scanDryRun

		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		scanCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.AgentTimeoutSeconds*6)*time.Second)
		defer cancel()

		fmt.Printf("Scanning namespaces: %v | LLM: %s | Dry Run: %v\n\n",
			cfg.Namespaces, cfg.LLMModel, cfg.DryRun)
		fmt.Println("Starting AI-SRE scan... (this may take 1-3 minutes)")
		fmt.Println()

		pipe, err := ai.New(cfg, k8sCli)
		if err != nil {
			return fmt.Errorf("init AI pipeline: %w", err)
		}

		report, err := pipe.RunScan(scanCtx)
		if err != nil {
			if ctx.Err() != nil {
				fmt.Println("\nScan canceled by user.")
				return nil
			}
			return fmt.Errorf("scan failed: %w", err)
		}

		if err := handleReport(ctx, report, scanApply); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanApply, "apply", false, "Prompt to apply remediation after scan")
	scanCmd.Flags().BoolVar(&scanDryRun, "dry-run", true, "Dry run mode — print commands but don't execute")
}

// handleReport persists, renders and optionally remediates an incident.
func handleReport(ctx context.Context, r *incident.Report, apply bool) error {
	// Persist to DB
	if err := store.SaveIncident(r); err != nil {
		logger.Sugar().Warnf("failed to save incident: %v", err)
	}

	// Render detail
	ui.PrintIncidentDetail(r)

	// Prompt for remediation
	if apply && r.HasIssue && len(r.RemediationPlan) > 0 {
		action := ui.PromptRemediation(r)
		switch action {
		case "execute":
			cfg.DryRun = false
			applyRemediation(r, false)
		case "dry":
			applyRemediation(r, true)
		}
	} else if !r.HasIssue {
		fmt.Println("\n✓ Cluster is healthy. No remediation needed.")
	} else if apply {
		fmt.Println("\n[dim]No remediation plan available.[/dim]")
	}

	return nil
}

// applyRemediation executes or dry-runs each step in the remediation plan.
func applyRemediation(r *incident.Report, dryRun bool) {
	fmt.Println()
	loader := runbooks.NewLoader(cfg.RunbooksDir)

	for i := range r.RemediationPlan {
		step := &r.RemediationPlan[i]
		fmt.Printf("  Step %d: %s\n", step.StepNumber, step.Description)

		if step.Command == "" {
			fmt.Println("    (No command — manual action required)")
			step.Status = "SKIPPED"
			continue
		}

		result := runbooks.Execute(step.Command, dryRun)
		if result.DryRun {
			fmt.Printf("    [DRY RUN] %s\n", result.Command)
			step.Status = "SKIPPED"
		} else if result.Success {
			fmt.Printf("    ✓ Done (%s)\n", result.Duration.Round(time.Millisecond))
			step.Status = "APPLIED"
		} else {
			fmt.Printf("    ✗ Failed: %s\n", result.Error)
			step.Status = "FAILED"
		}
	}

	_ = loader // used for BestMatch in future expansion

	// Update DB with applied steps
	applied := 0
	for _, s := range r.RemediationPlan {
		if s.Status == "APPLIED" {
			applied++
		}
	}
	if applied > 0 {
		now := time.Now().UTC()
		r.Status = incident.StatusMitigated
		r.MitigatedAt = &now
		r.ComputeMTTR()
		_ = store.SaveIncident(r)
		_ = store.UpdateStatus(r.ID, incident.StatusMitigated)
		fmt.Printf("\n✓ Incident %s marked as MITIGATED (%d/%d steps applied)\n",
			r.ID, applied, len(r.RemediationPlan))
	}
}
