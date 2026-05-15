// cmd/root.go — root command, global flags, dependency wiring.
package cmd

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/db"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/k8s"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/runbooks"
)

// RunbookFiles is populated by the embed directive in the build (see embed.go).
var RunbookFiles embed.FS

// Global shared state wired by root PersistentPreRunE.
var (
	cfg    *config.Config
	store  *db.DB
	k8sCli *k8s.Client
	logger *zap.Logger
)

var debugFlag bool

var rootCmd = &cobra.Command{
	Use:   "clarctl",
	Short: "Claritty AI-SRE — Production-grade Kubernetes observability engine",
	Long: `clarctl is an AI-powered Site Reliability Engineering CLI for Kubernetes clusters.

It continuously monitors your cluster, performs root-cause analysis, and 
suggests (or applies) automated remediation plans.

Examples:
  clarctl status              Show current cluster health
  clarctl scan                Run a single AI-SRE scan
  clarctl scan --apply        Scan and interactively apply remediation
  clarctl watch               Continuous monitoring loop
  clarctl incidents           View incident history
  clarctl show INC-ABCD1234   Show incident detail
  clarctl apply INC-ABCD1234  Apply remediation for an incident`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip setup for help/completion commands
		if cmd.Name() == "help" || cmd.Name() == "__complete" {
			return nil
		}
		return setup()
	},
}

// Execute is the entrypoint called from main.go.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	rootCmd.AddCommand(scanCmd, watchCmd, statusCmd, incidentsCmd, showCmd, applyCmd, reportCmd)
}

// setup initialises all shared dependencies.
func setup() error {
	// ── Config ───────────────────────────────────────────────────────────
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	// ── Logger ───────────────────────────────────────────────────────────
	logLevel := zap.WarnLevel
	if debugFlag {
		logLevel = zap.DebugLevel
	}
	logCfg := zap.NewProductionConfig()
	logCfg.Level = zap.NewAtomicLevelAt(logLevel)
	logCfg.OutputPaths = []string{cfg.LogPath, "stderr"}
	logger, _ = logCfg.Build()

	// ── Database ──────────────────────────────────────────────────────────
	store, err = db.Open(cfg.DBPath)
	if err != nil {
		return fmt.Errorf("open database at %s: %w", cfg.DBPath, err)
	}

	// ── Kubernetes ────────────────────────────────────────────────────────
	k8sCli, err = k8s.New(cfg.KubeconfigPath)
	if err != nil {
		return fmt.Errorf("connect to Kubernetes: %w", err)
	}

	// ── Runbooks ─────────────────────────────────────────────────────────
	runbooks.RunbookFS = RunbookFiles

	return nil
}
