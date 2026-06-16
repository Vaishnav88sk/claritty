---
sidebar_position: 4
---

# CLI Usage

The `clarctl` CLI is the developer's gateway to Claritty. Run directly from your terminal, it leverages your local `kubeconfig` context to instantly diagnose issues in your clusters.

## Basic Commands

### Verify Installation
Ensure the CLI is correctly installed and dynamically versioned:
```bash
clarctl version
```

### Scan the Current Context
Run a comprehensive, real-time diagnostic scan of your entire active Kubernetes context:
```bash
clarctl scan
```
This command triggers the 6-stage AI pipeline locally, generating a beautifully formatted terminal output (using `lipgloss` and `bubbletea`) instead of raw JSON.

---

## Interactive Remediation

When `clarctl scan` detects an incident, it doesn't just stop at giving you an RCA. The **Commander Agent** will propose a step-by-step remediation plan right in your terminal.

For every command proposed by the AI, the CLI will enter interactive mode:

```bash
> AI proposes: kubectl rollout restart deployment my-app -n default
> Execute? [y/dry/n]: 
```

### Prompt Options
- `y` (Yes): Immediately executes the command against the cluster.
- `dry` (Dry Run): Simulates the command using the Kubernetes API `--dry-run=client` flag to ensure it's structurally valid without making actual mutations.
- `n` (No): Rejects the command and halts the remediation sequence.

:::warning

While Claritty runs commands through a strict allowlist to prevent destructive actions, you should always review proposed commands before typing `y`. Use `dry` if you are unsure of a command's side-effects.

:::
