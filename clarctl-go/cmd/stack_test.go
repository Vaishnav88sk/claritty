package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestStackCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake Docker executable uses a POSIX shell")
	}

	checkout := t.TempDir()
	hubDir := filepath.Join(checkout, "sre-agent", "hub")
	if err := os.MkdirAll(hubDir, 0755); err != nil {
		t.Fatal(err)
	}
	composeFile := filepath.Join(checkout, "docker-compose.yml")
	for _, path := range []string{composeFile, filepath.Join(checkout, "sre-agent", "docker-compose.yml"), filepath.Join(hubDir, "Dockerfile")} {
		if err := os.WriteFile(path, nil, 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(hubDir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("LLM_PROVIDER", "groq")
	t.Setenv("GROQ_API_KEY", "")

	binDir := t.TempDir()
	t.Setenv("PATH", binDir)
	script := "#!/bin/sh\nprintf '%s\\n' \"$PWD\" \"$@\"\nexit \"$DOCKER_TEST_EXIT_CODE\"\n"
	if err := os.WriteFile(filepath.Join(binDir, "docker"), []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name, command, dockerExit, arguments, success string
	}{
		{"up waits for the dashboard", "up", "0", "up\n--build\n--wait\n", "Dashboard ready at http://localhost:8822"},
		{"down preserves the volume", "down", "0", "down\n", "PostgreSQL data preserved"},
		{"up reports a Docker failure", "up", "42", "up\n--build\n--wait\n", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DOCKER_TEST_EXIT_CODE", tc.dockerExit)
			var stdout, stderr bytes.Buffer
			rootCmd.SetArgs([]string{tc.command})
			rootCmd.SetOut(&stdout)
			rootCmd.SetErr(&stderr)
			t.Cleanup(func() {
				rootCmd.SetArgs(nil)
				rootCmd.SetOut(nil)
				rootCmd.SetErr(nil)
			})

			err := rootCmd.Execute()
			if tc.dockerExit == "0" && err != nil {
				t.Fatalf("clarctl %s: %v", tc.command, err)
			}
			if tc.dockerExit != "0" && (err == nil || !strings.Contains(err.Error(), "exit status 42")) {
				t.Fatalf("expected Docker exit status 42, got %v", err)
			}
			want := strings.Join([]string{checkout, "compose", "--file", composeFile, tc.arguments}, "\n")
			output := stdout.String()
			if !strings.HasPrefix(output, want) {
				t.Fatalf("expected Docker invocation %q, got %q", want, output)
			}
			if tc.success == "" && strings.Contains(output, "Dashboard ready") {
				t.Fatalf("unexpected success after Docker failure: %q", output)
			}
			if tc.success != "" && !strings.Contains(output, tc.success) {
				t.Fatalf("expected success message %q, got %q", tc.success, output)
			}
		})
	}
}

func TestFindStackComposeFileIgnoresUnrelatedCompose(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), nil, 0644); err != nil {
		t.Fatal(err)
	}
	if path, err := findStackComposeFile(dir); err == nil {
		t.Fatalf("unexpected Compose file %q outside a Claritty checkout", path)
	}
}
