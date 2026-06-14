# AI Agent Guidelines (`AGENTS.md`)

Welcome, AI Coding Assistant! If you are reading this file, you have been tasked with contributing to **Claritty**, a decentralized AI-SRE observability platform for Kubernetes.

To ensure your code aligns perfectly with our architecture and maintainability standards, please strictly adhere to the following guidelines:

## 1. Project Architecture
The repository is split into two primary domains:
*   **`clarctl-go/`**: The frontend CLI tool written in **Go 1.25.0**. It communicates with Kubernetes and formats the output into beautiful terminal dashboards.
*   **`sre-agent/`**: The backend system that runs inside the Kubernetes cluster. It interacts with LLMs (Groq, Mistral, OpenAI) to perform automated Root Cause Analysis (RCA).

## 2. Core Go Directives
*   **Version Constraints:** Always assume Go 1.25.0 unless specified otherwise. We strictly use `go vet` and `go fmt` natively rather than third-party linters to ensure compatibility.
*   **Do Not Break the Build:** Before committing Go code, always run `go vet ./...` and `gofmt -s -w .`. 
*   **Testing is Mandatory:** Every new feature in `clarctl-go` must have an accompanying unit test using the standard `testing` package. Our CI strictly enforces code coverage via Codecov.
*   **Dependency Management:** Do not add heavy external dependencies without asking. The CLI (`clarctl-go`) should be lightweight. Only use `k8s.io/client-go` when absolutely necessary, and prefer clean, abstracted interfaces over raw API calls.
*   **Dynamic Versioning:** The CLI uses dynamic linker flags for versioning. If modifying the UI banner or version logic, never hardcode the version string; ensure it relies on the `ui.Version` variable injected by the Makefile.

## 3. UI/UX Standards for the CLI
*   The CLI utilizes the **`charmbracelet`** ecosystem (`lipgloss`, `huh`, `bubbletea`).
*   Output must be beautiful and terminal-friendly. Use soft gradients, clear padding, and box-drawing characters for dashboards.
*   **Never output raw JSON** or unformatted text blocks unless the user explicitly passes an `--output=json` flag.
*   Color palettes should be accessible and support both light and dark terminal backgrounds.

## 4. Kubernetes & Agent Standards
*   The agent (`sre-agent`) must remain stateless where possible to allow for Horizontal Pod Autoscaling (HPA).
*   Any changes to RBAC permissions (`sre-agent/deploy/agent-rbac.yaml`) must follow the principle of least privilege.
*   Mock external LLM APIs when writing unit tests for the agent to prevent CI flakiness.

## 5. Submitting Pull Requests & CI
*   Keep PRs tightly scoped to a single feature or fix.
*   Ensure the `.github/workflows/ci.yaml` action passes. 
*   Note: The `KinD E2E Test` workflow (`e2e.yaml`) only triggers if files inside `sre-agent/**` are modified. This is an intentional optimization to save CI runner minutes.
*   Add clear, concise documentation for any new CLI commands or architecture changes.

**By strictly following these guidelines, you will help us maintain a robust, professional open-source project. Happy coding!**
