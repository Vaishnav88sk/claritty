package cmd

import (
	"bytes"
	"testing"
)

func TestRootCmd_HasSubCommands(t *testing.T) {
	// Root command should have exactly the commands we added in init()
	expectedCmds := []string{"scan", "watch", "status", "incidents", "show", "apply", "report", "configure"}

	for _, name := range expectedCmds {
		var found bool
		for _, sub := range rootCmd.Commands() {
			if sub.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected subcommand '%s' not found on rootCmd", name)
		}
	}
}

func TestRootCmd_HelpOutput(t *testing.T) {
	// Execute the help command, which skips the PersistentPreRunE setup
	// so it won't try to connect to K8s or the DB.
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(out)
	rootCmd.SetArgs([]string{"--help"})

	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("Unexpected error executing help: %v", err)
	}

	output := out.String()
	if !bytes.Contains([]byte(output), []byte("clarctl is an AI-powered")) {
		t.Errorf("Help output did not contain expected banner/description. Output: %s", output)
	}
	if !bytes.Contains([]byte(output), []byte("Available Commands:")) {
		t.Errorf("Help output did not contain 'Available Commands:'")
	}
}
