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
	cfg          *config.Config
	store        *db.DB
	k8sCli       *k8s.Client
	logger       *zap.Logger
	debugFlag    bool
	modelFlag    string
	providerFlag string
)

var rootCmd = &cobra.Command{
	Use:   "clarctl",
	Short: "Claritty AI-SRE — Production-grade Kubernetes observability engine",
	Long: `clarctl is an AI-powered Site Reliability Engineering CLI for Kubernetes clusters.

It continuously monitors your cluster, performs root-cause analysis, and 
suggests (or applies) automated remediation plans.

Examples:
  clarctl up                  Start the local Hub, web dashboard, and database
  clarctl down                Stop the local Hub and database
  clarctl status              Show current cluster health
  clarctl scan                Run a single AI-SRE scan
  clarctl scan --apply        Scan and interactively apply remediation
  clarctl scan -m claude-3-7-sonnet-latest -p anthropic
  clarctl watch               Continuous monitoring loop
  clarctl incidents           View incident history
  clarctl show INC-ABCD1234   Show incident detail
  clarctl apply INC-ABCD1234  Apply remediation for an incident`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch cmd.Name() {
		case "help", "__complete", "configure", "up", "down":
			return nil
		default:
			return setup()
		}
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
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "", "Override LLM model name (e.g. claude-3-7-sonnet-latest, gpt-4o)")
	rootCmd.PersistentFlags().StringVarP(&providerFlag, "provider", "p", "", "Override LLM provider (groq, anthropic, openai, mistral, ollama)")
	rootCmd.AddCommand(scanCmd, watchCmd, statusCmd, incidentsCmd, showCmd, applyCmd, reportCmd, configureCmd)
	rootCmd.AddCommand(
		newStackCommand("up", "Start the local Hub, web dashboard, and PostgreSQL", "Dashboard ready at http://localhost:8822", "up", "--build", "--wait"),
		newStackCommand("down", "Stop the local stack and keep PostgreSQL data", "Local stack stopped. PostgreSQL data preserved.", "down"),
	)
}

// setup initialises all shared dependencies.
func setup() error {
	// ── Config ───────────────────────────────────────────────────────────
	var err error
	cfg, err = config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if providerFlag != "" {
		cfg.LLMProvider = providerFlag
		if modelFlag == "" {
			cfg.LLMModel = config.DefaultModelForProvider(providerFlag)
		}
	}
	if modelFlag != "" {
		cfg.LLMModel = modelFlag
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
