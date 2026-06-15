// Package ui provides terminal rendering using lipgloss.
package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
	"github.com/charmbracelet/lipgloss"
)

// ─── Styles ───────────────────────────────────────────────────────────────────

var (
	styleBanner = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#00D7FF")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#005F87")).
			Padding(0, 2)

	styleHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#00D7FF"))
	styleGood   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))
	styleWarn   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	styleCrit   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F5F")).Bold(true)
	styleDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262"))
	styleBold   = lipgloss.NewStyle().Bold(true)
	styleBox    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#444444")).
			Padding(0, 1)
)

// Version is the CLI version. It is meant to be overridden at build time via ldflags.
// Example: go build -ldflags "-X 'github.com/Vaishnav88sk/claritty/clarctl-go/internal/ui.Version=v1.1.0'"
var Version = "dev"

// PrintBanner renders the Claritty ASCII art banner.
func PrintBanner() {
	banner := `
 ██████╗██╗      █████╗ ██████╗ ██╗████████╗████████╗██╗   ██╗
██╔════╝██║     ██╔══██╗██╔══██╗██║╚══██╔══╝╚══██╔══╝╚██╗ ██╔╝
██║     ██║     ███████║██████╔╝██║   ██║      ██║    ╚████╔╝ 
██║     ██║     ██╔══██║██╔══██╗██║   ██║      ██║     ╚██╔╝  
╚██████╗███████╗██║  ██║██║  ██║██║   ██║      ██║      ██║   
 ╚═════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝╚═╝   ╚═╝      ╚═╝      ╚═╝   `

	info := styleDim.Render(fmt.Sprintf("AI-SRE Engine  ·  %s  ·  Kubernetes Observability", Version))
	fmt.Println(styleBanner.Render(banner + "\n" + info))
	fmt.Println()
}

// PrintClusterHealth renders the cluster health dashboard panel.
func PrintClusterHealth(snap *incident.ClusterSnapshot) {
	if snap == nil {
		fmt.Println(styleBox.Render(styleDim.Render("No health data yet — run: clarctl scan")))
		return
	}

	scoreColor := styleGood
	if snap.HealthScore < 70 {
		scoreColor = styleWarn
	}
	if snap.HealthScore < 40 {
		scoreColor = styleCrit
	}

	content := fmt.Sprintf(
		"%s %s\n%s  %s  %s  %s\n%s  %s",
		styleHeader.Render("Cluster Health Score:"),
		scoreColor.Render(fmt.Sprintf("%.0f/100", snap.HealthScore)),
		styleDim.Render(fmt.Sprintf("Nodes: %d/%d Ready", snap.ReadyNodes, snap.TotalNodes)),
		styleDim.Render(fmt.Sprintf("Pods: %d Running / %d Pending / %d Failed",
			snap.RunningPods, snap.PendingPods, snap.FailedPods)),
		styleDim.Render(fmt.Sprintf("CrashLoop: %d", snap.CrashloopPods)),
		styleDim.Render(fmt.Sprintf("Open Incidents: %d", snap.OpenIncidents)),
		styleDim.Render(fmt.Sprintf("CPU: %.1f%%", snap.CPUUsagePct)),
		styleDim.Render(fmt.Sprintf("Memory: %.1f%%", snap.MemUsagePct)),
	)
	fmt.Println(styleBox.Render(styleHeader.Render("─── Cluster Health ───") + "\n" + content))
}

// PrintIncidentsTable renders a sorted table of incidents.
func PrintIncidentsTable(incidents []*incident.Report) {
	if len(incidents) == 0 {
		fmt.Println(styleGood.Render("✓ No incidents in the selected time range."))
		return
	}

	// Header
	fmt.Printf("\n%s\n", styleHeader.Render("─── Incident History ───"))
	fmt.Printf("%-14s %-6s %-10s %-10s %-55s %s\n",
		"ID", "SEV", "STATUS", "CONFIDENCE", "TITLE", "DETECTED")
	fmt.Println(strings.Repeat("─", 110))

	for _, r := range incidents {
		sevStyle := styleSeverity(r.Severity)
		fmt.Printf("%-14s %-6s %-10s %-10s %-55s %s\n",
			r.ID,
			sevStyle.Render(string(r.Severity)),
			string(r.Status),
			fmt.Sprintf("%d%%", r.ConfidenceScore),
			truncate(r.Title, 54),
			r.DetectedAt.Format("01/02 15:04"),
		)
	}
}

// PrintIncidentDetail renders the full detail view of an incident.
func PrintIncidentDetail(r *incident.Report) {
	sevStyle := styleSeverity(r.Severity)
	header := fmt.Sprintf("%s  %s  [%s]  Confidence: %d%%",
		sevStyle.Render("["+string(r.Severity)+"]"),
		styleBold.Render(r.Title),
		string(r.Status),
		r.ConfidenceScore,
	)
	fmt.Println(styleBox.Render(header))
	fmt.Println()

	fmt.Printf("%s %s\n", styleHeader.Render("ID:"), r.ID)
	fmt.Printf("%s %s\n", styleHeader.Render("Category:"), r.Category)
	fmt.Printf("%s %s\n", styleHeader.Render("Namespaces:"), strings.Join(r.AffectedNamespaces, ", "))
	if len(r.AffectedServices) > 0 {
		fmt.Println(styleHeader.Render("Affected Pods/Services:"))
		for _, svc := range r.AffectedServices {
			fmt.Printf("  • %s (Namespace: %s) — Impact: %s\n", styleBold.Render(svc.ServiceName), svc.Namespace, svc.ImpactLevel)
		}
	}
	fmt.Printf("%s %s\n", styleHeader.Render("Detected:"), r.DetectedAt.Format(time.RFC3339))

	if r.LLMModel != "" {
		fmt.Printf("%s %s\n", styleHeader.Render("LLM Model:"), styleDim.Render(r.LLMModel))
	}
	fmt.Println()

	if r.RootCause != "" {
		fmt.Println(styleHeader.Render("Root Cause:"))
		fmt.Println(" ", r.RootCause)
		fmt.Println()
	}

	if len(r.ContributingFactors) > 0 {
		fmt.Println(styleHeader.Render("Contributing Factors:"))
		for _, f := range r.ContributingFactors {
			fmt.Printf("  • %s\n", f)
		}
		fmt.Println()
	}

	if len(r.RemediationPlan) > 0 {
		fmt.Println(styleHeader.Render("Remediation Plan:"))
		for _, step := range r.RemediationPlan {
			destructive := ""
			if step.Destructive {
				destructive = styleCrit.Render(" [DESTRUCTIVE]")
			}
			fmt.Printf("  %s %s%s\n",
				styleBold.Render(fmt.Sprintf("Step %d:", step.StepNumber)),
				step.Description,
				destructive,
			)
			if step.Command != "" {
				fmt.Printf("    %s\n", styleDim.Render("$ "+step.Command))
			}
		}
		fmt.Println()
	}
}

// PromptRemediation asks the user to choose how to handle the remediation plan.
// Returns "execute", "dry", or "skip".
func PromptRemediation(r *incident.Report) string {
	if len(r.RemediationPlan) == 0 {
		return "skip"
	}

	fmt.Println()
	fmt.Println(styleWarn.Render("⚡ Remediation plan available. What would you like to do?"))
	fmt.Println(styleBold.Render("  [y]") + "  Execute all steps (applies real commands)")
	fmt.Println(styleBold.Render("  [dry]") + " Dry-run only (print commands, don't execute)")
	fmt.Println(styleBold.Render("  [n]") + "  Skip — I'll handle this manually")
	fmt.Print(styleHeader.Render("Your choice [y/dry/n]: "))

	var input string
	fmt.Scanln(&input)
	switch strings.ToLower(strings.TrimSpace(input)) {
	case "y", "yes":
		return "execute"
	case "dry":
		return "dry"
	default:
		return "skip"
	}
}

// PrintMTTRStats renders MTTR statistics.
func PrintMTTRStats(avg float64, count int) {
	if count == 0 {
		fmt.Println(styleDim.Render("No resolved incidents yet — MTTR stats unavailable."))
		return
	}
	fmt.Printf("%s Avg MTTR: %s (%d incidents)\n",
		styleHeader.Render("📊"),
		styleGood.Render(fmt.Sprintf("%.0fs", avg)),
		count,
	)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func styleSeverity(sev incident.Severity) lipgloss.Style {
	switch sev {
	case incident.SEV1:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true)
	case incident.SEV2:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF8C00")).Bold(true)
	case incident.SEV3:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD700"))
	default:
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// PrintConfigSuccess renders a success message after configuration.
func PrintConfigSuccess(provider, model, path string) {
	fmt.Println()
	fmt.Println(styleHeader.Render("✨ Successfully configured Claritty!"))
	fmt.Printf("%s %s\n", styleBold.Render("Provider:"), styleGood.Render(provider))
	fmt.Printf("%s %s\n", styleBold.Render("Model:   "), styleGood.Render(model))
	fmt.Printf("%s %s\n", styleBold.Render("Path:    "), styleDim.Render(path))
}
