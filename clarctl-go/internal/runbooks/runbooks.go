// Package runbooks loads, parses and executes YAML runbook definitions.
package runbooks

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
)

// RunbookFS holds the embedded runbook YAML files baked into the binary.
// The actual //go:embed directive is in the cmd package to keep this package pure.
var RunbookFS embed.FS

// Runbook is the parsed structure of a runbook YAML file.
type Runbook struct {
	Name        string              `yaml:"name"`
	Description string              `yaml:"description"`
	Severity    string              `yaml:"severity"`
	Category    string              `yaml:"category"`
	Triggers    []string            `yaml:"triggers"`
	Diagnosis   []string            `yaml:"diagnosis"`
	Steps       []RunbookStep       `yaml:"steps"`
	Notes       []string            `yaml:"notes"`
}

// RunbookStep mirrors the YAML step structure.
type RunbookStep struct {
	StepNumber  int    `yaml:"step_number"`
	Description string `yaml:"description"`
	Command     string `yaml:"command"`
	Destructive bool   `yaml:"is_destructive"`
	Automated   bool   `yaml:"is_automated"`
}

// ToRemediationSteps converts runbook steps to incident model steps.
func (r *Runbook) ToRemediationSteps() []incident.RemediationStep {
	out := make([]incident.RemediationStep, 0, len(r.Steps))
	for _, s := range r.Steps {
		out = append(out, incident.RemediationStep{
			StepNumber:  s.StepNumber,
			Description: s.Description,
			Command:     s.Command,
			Destructive: s.Destructive,
			Automated:   s.Automated,
			Status:      "PENDING",
		})
	}
	return out
}

// ─── Loader ───────────────────────────────────────────────────────────────────

// Loader handles discovering and parsing runbook files.
type Loader struct {
	dir string // external override dir (empty = use embedded FS)
}

// NewLoader creates a Loader. If dir is empty, embedded runbooks are used.
func NewLoader(dir string) *Loader {
	return &Loader{dir: dir}
}

// List returns the names of all available runbooks.
func (l *Loader) List() ([]string, error) {
	entries, err := l.readDir()
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e, ".yaml") || strings.HasSuffix(e, ".yml") {
			names = append(names, e)
		}
	}
	return names, nil
}

// Load reads and parses a specific runbook by filename.
func (l *Loader) Load(name string) (*Runbook, error) {
	data, err := l.readFile(name)
	if err != nil {
		return nil, fmt.Errorf("read runbook %s: %w", name, err)
	}
	var rb Runbook
	if err := yaml.Unmarshal(data, &rb); err != nil {
		return nil, fmt.Errorf("parse runbook %s: %w", name, err)
	}
	return &rb, nil
}

// BestMatch attempts to find the most relevant runbook for a given category.
func (l *Loader) BestMatch(category string) (*Runbook, string, error) {
	names, err := l.List()
	if err != nil {
		return nil, "", err
	}
	// Try exact category name match first (e.g. "crash_loop.yaml" for "crashloop")
	normalized := strings.ReplaceAll(strings.ToLower(category), "_", "")
	for _, name := range names {
		base := strings.TrimSuffix(strings.ToLower(name), ".yaml")
		base = strings.ReplaceAll(base, "_", "")
		if base == normalized {
			rb, err := l.Load(name)
			return rb, name, err
		}
	}
	// Substring match
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), normalized) {
			rb, err := l.Load(name)
			return rb, name, err
		}
	}
	return nil, "", nil
}

// ─── Executor ────────────────────────────────────────────────────────────────

// SafeKubectlPatterns is the allowlist of permitted command prefixes.
var SafeKubectlPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^kubectl get `),
	regexp.MustCompile(`^kubectl describe `),
	regexp.MustCompile(`^kubectl logs `),
	regexp.MustCompile(`^kubectl rollout restart `),
	regexp.MustCompile(`^kubectl rollout history `),
	regexp.MustCompile(`^kubectl rollout undo `),
	regexp.MustCompile(`^kubectl scale `),
	regexp.MustCompile(`^kubectl delete pod `),
	regexp.MustCompile(`^kubectl cordon `),
	regexp.MustCompile(`^kubectl uncordon `),
	regexp.MustCompile(`^kubectl top `),
}

// ExecResult holds the outcome of a command execution.
type ExecResult struct {
	Command  string
	DryRun   bool
	Success  bool
	Stdout   string
	Stderr   string
	Error    string
	Duration time.Duration
}

// Execute runs a kubectl command with safety validation.
func Execute(command string, dryRun bool) ExecResult {
	res := ExecResult{Command: command, DryRun: dryRun}

	if !isSafeCommand(command) {
		res.Error = fmt.Sprintf("command blocked by safety allowlist: %s", command)
		return res
	}

	if dryRun {
		res.Success = true
		return res
	}

	start := time.Now()
	parts := strings.Fields(command)
	cmd := exec.Command(parts[0], parts[1:]...) //nolint:gosec
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	res.Duration = time.Since(start)
	res.Stdout = stdout.String()
	res.Stderr = stderr.String()
	if err != nil {
		res.Error = err.Error()
	} else {
		res.Success = true
	}
	return res
}

func isSafeCommand(cmd string) bool {
	for _, pattern := range SafeKubectlPatterns {
		if pattern.MatchString(cmd) {
			return true
		}
	}
	return false
}

// ─── FS helpers ───────────────────────────────────────────────────────────────

func (l *Loader) readDir() ([]string, error) {
	if l.dir != "" {
		entries, err := os.ReadDir(l.dir)
		if err != nil {
			return nil, err
		}
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return names, nil
	}
	// Embedded FS
	entries, err := fs.ReadDir(RunbookFS, "runbooks")
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names, nil
}

func (l *Loader) readFile(name string) ([]byte, error) {
	if l.dir != "" {
		return os.ReadFile(filepath.Join(l.dir, name))
	}
	return RunbookFS.ReadFile(filepath.Join("runbooks", name))
}
