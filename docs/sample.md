# Claritty Architecture Overview

Claritty is composed of two primary operating modes:

1. **Claritty CLI (`clarctl`)**: A local command-line tool for SREs to instantly diagnose and remediate issues in a Kubernetes cluster they have access to.
2. **Claritty SRE Agent (`sre-agent`)**: An in-cluster autonomous agent that continuously monitors namespaces, paired with a central Hub server to provide a web-based dashboard and alerting.

## Example: Diagnosing a CrashLoopBackOff

If a pod is in a `CrashLoopBackOff` state, Claritty's AI pipeline executes the following steps:

1. **Triage**: Identifies the pod state and relevant namespace.
2. **Metrics**: Fetches memory and CPU usage to check for resource starvation.
3. **Logs**: Retrieves the last 100 lines of the crashed container's logs.
4. **Infra**: Checks Kubernetes events and pod descriptions for exit codes (e.g., `OOMKilled`, or `Exit Code 1`).
5. **Runbook**: Looks up the built-in runbook for `CrashLoopBackOff`.
6. **Commander**: Synthesizes all data and generates a root cause summary along with safe, copy-pasteable remediation commands.
