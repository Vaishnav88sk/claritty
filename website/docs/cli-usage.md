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
This command triggers the 6-stage AI pipeline locally, generating a beautifully formatted terminal dashboard with incident metadata, evidence citations, chronological timeline, and change analysis.

### Scan a Specific Namespace
Scope the diagnostic scan to a single namespace:
```bash
clarctl scan namespace production
```

### Scan a Specific Pod
Focus the deep-dive investigation on a specific pod:
```bash
clarctl scan pod payment-service-78dfb -n production
```

### Change Correlation Window (`--change-window`)
Configure the lookback window for the "What Changed?" engine to detect recent rollouts, image tag mutations, and config changes:
```bash
# Look back 30 minutes (default is 1h)
clarctl scan namespace production --change-window=30m

# Look back 2 hours
clarctl scan namespace production --change-window=2h
```

### Structured Output (`--output=json`)
Export the complete incident report, including evidence items, timeline, changes, conclusions, and remediation plan:
```bash
clarctl scan namespace production --output=json
```

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
