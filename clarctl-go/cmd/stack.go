package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

func newStackCommand(name, short, success string, composeArgs ...string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Long:  short + ".\nRequires Docker with the Compose plugin and an installed stack bundle or a Claritty checkout.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			composeFile, err := findStackComposeFile(dir)
			if err != nil {
				return err
			}
			dockerPath, err := exec.LookPath("docker")
			if err != nil {
				return fmt.Errorf("Docker with the Compose plugin is required: %w", err)
			}
			docker := exec.CommandContext(cmd.Context(), dockerPath, append([]string{"compose", "--file", composeFile}, composeArgs...)...)
			docker.Dir = filepath.Dir(composeFile)
			docker.Stdin = cmd.InOrStdin()
			docker.Stdout = cmd.OutOrStdout()
			docker.Stderr = cmd.ErrOrStderr()
			if err := docker.Run(); err != nil {
				return fmt.Errorf("docker compose %s failed: %w", name, err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), lipgloss.NewStyle().Bold(true).Render("✓ "+success))
			return nil
		},
	}
}

func findStackComposeFile(dir string) (string, error) {
	installed := filepath.Join(config.ClarittyDir(), "stack", "claritty", "docker-compose.yml")
	if _, err := os.Stat(installed); err == nil {
		return installed, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("find installed Claritty Compose file: %w", err)
	}

	for {
		// The root Compose file builds the Hub from this checkout. Use the
		// Dockerfile to distinguish it from unrelated and nested Compose files.
		if _, err := os.Stat(filepath.Join(dir, "sre-agent", "hub", "Dockerfile")); err == nil {
			composeFile := filepath.Join(dir, "docker-compose.yml")
			if _, err := os.Stat(composeFile); err != nil {
				return "", fmt.Errorf("find Claritty Compose file: %w", err)
			}
			return composeFile, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("Claritty stack files are missing. Install the stack bundle or run from a Claritty checkout")
		}
		dir = parent
	}
}
