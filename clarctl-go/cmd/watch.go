// cmd/watch.go — clarctl watch: continuous monitoring loop.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ai"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui"
)

var (
	watchInterval int
	watchApply    bool
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Continuous monitoring loop with live dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.PrintBanner()

		pipe, err := ai.New(cfg, k8sCli)
		if err != nil {
			return fmt.Errorf("init AI pipeline: %w", err)
		}

		fmt.Printf("Starting continuous watcher (interval: %ds, Ctrl+C to stop)\n\n", watchInterval)

		// Graceful shutdown on SIGINT / SIGTERM
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		scanCount := 0
		for {
			if ctx.Err() != nil {
				fmt.Println("\nWatcher stopped.")
				return nil
			}

			scanCount++
			ts := time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
			fmt.Printf("\n──── Scan #%d  ·  %s ────\n", scanCount, ts)

			scanCtx, cancel := context.WithTimeout(ctx,
				time.Duration(cfg.AgentTimeoutSeconds*6)*time.Second)

			report, err := pipe.RunScan(scanCtx)
			cancel()
			
			if err != nil {
				if ctx.Err() != nil {
					fmt.Println("\nWatcher stopped by user.")
					return nil
				}
				fmt.Printf("[error] Scan #%d failed: %v\n", scanCount, err)
			} else {
				_ = store.SaveIncident(report)

				snap, _ := k8sCli.CollectSnapshot(ctx, cfg.Namespaces)
				if snap != nil {
					openCount, _ := store.OpenIncidentCount()
					dbSnap := &incident.ClusterSnapshot{
						Timestamp:     time.Now().UTC(),
						TotalNodes:    snap.TotalNodes,
						ReadyNodes:    snap.ReadyNodes,
						TotalPods:     snap.TotalPods,
						RunningPods:   snap.RunningPods,
						PendingPods:   snap.PendingPods,
						FailedPods:    snap.FailedPods,
						CrashloopPods: snap.CrashloopPods,
						OpenIncidents: openCount,
					}
					dbSnap.ComputeHealthScore()
					_ = store.SaveSnapshot(dbSnap)
					ui.PrintClusterHealth(dbSnap)
				}

				recentInc, _ := store.GetIncidents("", "", 24, 10)
				ui.PrintIncidentsTable(recentInc)

				avg, count, _ := store.MTTRStats()
				ui.PrintMTTRStats(avg, count)

				// For SEV1/SEV2: show detail and prompt remediation
				if report.Severity == incident.SEV1 || report.Severity == incident.SEV2 {
					ui.PrintIncidentDetail(report)
					if watchApply && len(report.RemediationPlan) > 0 {
						action := ui.PromptRemediation(report)
						switch action {
						case "execute":
							cfg.DryRun = false
							applyRemediation(report, false)
						case "dry":
							applyRemediation(report, true)
						}
					}
				}
			}

			fmt.Printf("\nNext scan in %ds...\n", watchInterval)
			select {
			case <-ctx.Done():
				fmt.Println("\nWatcher stopped.")
				return nil
			case <-time.After(time.Duration(watchInterval) * time.Second):
			}
		}
	},
}

func init() {
	watchCmd.Flags().IntVar(&watchInterval, "interval", 300, "Scan interval in seconds")
	watchCmd.Flags().BoolVar(&watchApply, "apply", false, "Auto-prompt remediation after each scan")
}
