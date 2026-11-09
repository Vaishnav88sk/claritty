# Contributing to Claritty

Claritty welcomes bug fixes, features, tests, documentation, and runbooks. Please follow the [Code of Conduct](CODE_OF_CONDUCT.md) when taking part in the project.

## Report an issue or propose a change

- Search the [existing issues](https://github.com/Vaishnav88sk/claritty/issues) before opening a new one. For a bug, include the Claritty version or image tag, affected component, reproduction steps, and expected and actual behavior. Include your Kubernetes version for cluster-related issues.
- Use [GitHub Discussions](https://github.com/Vaishnav88sk/claritty/discussions) for usage questions and early ideas. Discuss changes to agent permissions, incident schemas, or deployment defaults before implementing them.
- Redact credentials, kubeconfig data, and sensitive logs before posting. Report suspected vulnerabilities through the [security policy](SECURITY.md).

## Develop and test

Use Go 1.25, which matches the CI toolchain. The CLI is in `clarctl-go/`, the in-cluster Agent is in `sre-agent/agent/`, and the Hub API and database code are in `sre-agent/hub/`. See [ARCHITECTURE.md](ARCHITECTURE.md) for how they interact.

Run these checks from each Go module you change.

```bash
gofmt -s -w .
go vet ./...
go test ./...
```

- Add standard Go `testing` coverage for new CLI features. Mock LLM calls in Agent tests and keep test data free of live credentials.
- Keep Agent RBAC scoped to the permissions the feature needs. Preserve validation and operator confirmation for CLI remediation commands.
- For the Svelte dashboard, use Node.js 22 and run `npm ci` and `npm run build` in `sre-agent/hub/dashboard/`. For the documentation site, use Node.js 20 and run the same commands in `website/`.

## Submit a change

Open a focused pull request targeting `master`. Explain what changed, how you tested it, and any impact on configuration, RBAC, or upgrades. Use Conventional Commit messages such as `fix(clarctl): handle missing stack files` or `docs: explain agent configuration`. Update the relevant usage guide when changing commands or deployment behavior.
