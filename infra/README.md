# Kubernetes deployment

Install the Hub once with an external PostgreSQL database. Install the Agent once in each cluster you want to monitor, giving each cluster a distinct `clusterName`.

## Secrets

Create the Secrets in the namespace of each Helm release before installing it. Set `existingSecret` to the Secret name. Both releases use the same nonempty `CLARITTY_HUB_API_KEY` for ingestion.

| Release | Secret keys |
| --- | --- |
| Hub | `DATABASE_URL`, `CLARITTY_HUB_API_KEY`, and optionally `SLACK_WEBHOOK_URL` |
| Agent | `CLARITTY_HUB_API_KEY` and the selected hosted provider's key (`GROQ_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `MISTRAL_API_KEY`) |

The Agent also supports `llm.provider=ollama` with `llm.ollamaHost` set, and `llm.provider=mock` for test environments. After rotating a Secret, restart the affected Deployment so it reloads the environment variables.

The Hub applies its schema migrations at startup. Its PostgreSQL role needs permission to create and alter the Claritty tables.

## Install with Helm

Set `version` to a published Helm chart version, omitting the `v` from the corresponding release tag. The charts published by the release workflow include the image digests for that release. Authenticate to GHCR with `helm registry login ghcr.io` if the packages are private. For a chart installed from this checkout, replace the OCI reference with the local chart path, omit `--version`, and pass `--set-string image=<image-reference>`.

Run the Hub install with `kubectl` configured for the Hub cluster.

```bash
helm upgrade --install claritty-hub oci://ghcr.io/vaishnav88sk/claritty/charts/claritty-hub \
  --version "$version" --namespace claritty --wait \
  --set-string existingSecret=claritty-hub-secrets
```

The Hub creates a ClusterIP Service on port `8822`. For Agents in the same cluster, set `hubURL` to `http://claritty-hub.claritty.svc:8822` when the cluster network is trusted. For cross-cluster installs, use a private HTTPS endpoint or a TLS-enabled gateway. Protect the dashboard and its API with your identity proxy. For machine access through a gateway, route `POST /api/v1/incidents` separately and require the ingestion key. Use PostgreSQL TLS and backups in production.

Run the Agent install with `kubectl` configured for the monitored cluster.

```bash
helm upgrade --install claritty-agent oci://ghcr.io/vaishnav88sk/claritty/charts/claritty-agent \
  --version "$version" --namespace claritty --wait \
  --set-string existingSecret=claritty-agent-secrets \
  --set-string clusterName=production-us-east \
  --set-string hubURL=https://claritty.example.com
```

The Agent scans immediately at startup and waits `scanIntervalSeconds` between scans. Set `watchNamespaces` to a list of existing namespaces to limit pod, log, event, and change collection and their RBAC grants. The Agent retains cluster-wide node visibility. The HTTP scan controls are disabled by default. Keep them on a private path if you enable `controlEnabled`.

For a cluster already running the raw Agent Deployment, stop the existing Agent before installing the chart to avoid overlapping scans. Choose a distinct Helm release name if you retain the raw manifests for rollback. Upgrading the Agent with Helm briefly stops scanning because the Deployment replaces the old pod before starting its replacement.

## Verify an installation

```bash
kubectl -n claritty logs deployment/claritty-agent
kubectl -n claritty port-forward service/claritty-hub 8822:8822
```

Open the dashboard at `http://localhost:8822` and confirm that your cluster has a report. The [KinD smoke test](../tests/kind/helm-smoke.sh) verifies report persistence and Agent restart with a mock provider and a disposable PostgreSQL instance.
