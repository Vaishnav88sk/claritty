# Kubernetes deployment

Install the Hub once with an external PostgreSQL database. Install the Agent once in each cluster you want to monitor, giving each cluster a distinct `clusterName`.

## Secrets

Create the Secrets in the namespace of each Helm release before installing it. Set `existingSecret` to the Secret name. Both releases use the same nonempty `CLARITTY_HUB_API_KEY` for ingestion.

| Release | Secret keys |
| --- | --- |
| Hub | `DATABASE_URL`, `CLARITTY_HUB_API_KEY`, and optionally `SLACK_WEBHOOK_URL` |
| Agent | `CLARITTY_HUB_API_KEY` and the selected hosted provider's key (`GROQ_API_KEY`, `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `MISTRAL_API_KEY`) |

The Agent also supports `llm.provider=ollama` with `llm.ollamaHost` set, and `llm.provider=mock` for test environments. After rotating a Secret, restart the affected Deployment so it reloads the environment variables.

The Hub chart runs versioned schema migrations in a Job before install and upgrade. Use `migrations.existingSecret` for a separate database owner credential and grant the Hub runtime role only the privileges it needs. See the [PostgreSQL operations guide](postgres/README.md).

## Install with Helm

Kubernetes pulls the Hub and Agent container images from Docker Hub. The Helm client contacts GHCR only when you install a published chart through an `oci://ghcr.io` URL. You can also install the charts directly from this checkout using Docker Hub image references.

### Use local charts and Docker Hub images

Run from the repository root. Set `TAG` to the published Docker Hub image tag that matches the chart revision in this checkout, including the `v` prefix. Use the Hub cluster's `kubectl` context for the first command and the monitored cluster's context for the second.

```bash
HUB_IMAGE="docker.io/vaishnav88sk/claritty-hub:${TAG:?Set TAG to a matching published release tag}"
AGENT_IMAGE="docker.io/vaishnav88sk/claritty-ai-sre-agent:${TAG}"

helm upgrade --install claritty-hub ./infra/helm/claritty-hub \
  --namespace claritty --wait --timeout 7m \
  --set-string existingSecret=claritty-hub-secrets \
  --set-string image="$HUB_IMAGE"

helm upgrade --install claritty-agent ./infra/helm/claritty-agent \
  --namespace claritty --wait \
  --set-string existingSecret=claritty-agent-secrets \
  --set-string image="$AGENT_IMAGE" \
  --set-string clusterName=production-us-east \
  --set-string hubURL=http://claritty-hub.claritty.svc:8822
```

For production, replace the two image assignments with digest references. You can also point them to images built from this checkout and made available to your cluster.

### Use published Helm charts from GHCR

The release workflow publishes versioned chart packages to GHCR and pins their images to the corresponding Docker Hub digests. Use `helm registry login ghcr.io` if the chart packages are private. Set `TAG` to the published release tag and derive the chart version by removing the `v` prefix.

```bash
VERSION="${TAG:?Set TAG to a published release tag}"
VERSION="${VERSION#v}"

helm upgrade --install claritty-hub oci://ghcr.io/vaishnav88sk/claritty/charts/claritty-hub \
  --version "$VERSION" --namespace claritty --wait --timeout 7m \
  --set-string existingSecret=claritty-hub-secrets

helm upgrade --install claritty-agent oci://ghcr.io/vaishnav88sk/claritty/charts/claritty-agent \
  --version "$VERSION" --namespace claritty --wait \
  --set-string existingSecret=claritty-agent-secrets \
  --set-string clusterName=production-us-east \
  --set-string hubURL=http://claritty-hub.claritty.svc:8822
```

### Configure connectivity

Both examples use the Hub's private ClusterIP address for an Agent in the same cluster. For Agents in other clusters, set `hubURL` to the reachable HTTPS ingestion hostname from the TLS and SSO configuration below.

The Agent scans immediately at startup and waits `scanIntervalSeconds` between scans. Set `watchNamespaces` to a list of existing namespaces to limit pod, log, event, and change collection and their RBAC grants. The Agent retains cluster-wide node visibility. The HTTP scan controls are disabled by default. Keep them on a private path if you enable `controlEnabled`.

For a cluster already running the raw Agent Deployment, stop the existing Agent before installing the chart to avoid overlapping scans. Choose a distinct Helm release name if you retain the raw manifests for rollback.

For maintainers, the release workflow uses `DOCKERHUB_USERNAME` and `DOCKERHUB_TOKEN` to publish to both Docker Hub repositories. The packaged Compose stack pins the Hub image by digest.

## TLS and dashboard SSO

Install the Gateway API CRDs and a compatible controller. Use a CNI that enforces NetworkPolicy. Create a Gateway with two exact HTTPS listener hostnames and a certificate covering both. The [Gateway example](examples/gateway.yaml) assumes a TLS Secret in the same namespace. Publish both DNS names to the Gateway address. [Gateway API TLS guide](https://gateway-api.sigs.k8s.io/guides/user-guides/tls/)

Register a confidential OIDC client and configure the callback `https://claritty.example.com/oauth2/callback` for your dashboard hostname. Create the OIDC Secret with `CLIENT_ID`, `CLIENT_SECRET`, and a randomly generated `COOKIE_SECRET`. List verified email domains and, where supported by your identity provider, restrict access further with `edge.oidc.allowedGroups`. [OAuth2 Proxy configuration](https://oauth2-proxy.github.io/oauth2-proxy/configuration/overview/)

Copy the [production values](examples/hub-production.values.yaml) into your environment repository. Pin an OAuth2 Proxy image with support for `--trusted-proxy-ip`, such as the 7.15 series. Match the Gateway's actual data plane Pod namespace and labels, and set `edge.oidc.trustedProxyCIDRs` to the addresses of the trusted proxy hops. If monitoring is enabled, match the Prometheus Pod namespace and labels too. Install or upgrade the Hub with `-f hub-production.values.yaml --timeout 7m`.

The dashboard and browser API run behind the OIDC proxy. The ingestion hostname serves only `POST /api/v1/incidents` and uses the Agent's ingestion key. Set the Agent `hubURL` to the ingestion hostname. Use `kubectl get httproute` to confirm both routes are accepted. From outside the cluster, verify that unauthenticated dashboard access starts an OIDC login, an unauthenticated ingestion POST returns `401`, and a GET to the ingestion endpoint returns `405`. Then sign in with an authorized account and confirm that the Hub receives an Agent report.

## Monitoring

Install Prometheus Operator before setting `monitoring.enabled=true`. Configure `monitoring.labels` to match your Prometheus `serviceMonitorSelector` and `ruleSelector`. Allow discovery in the chart's namespace. The Agent exposes scan duration, report delivery outcomes, and leader status. The Hub exposes the last persisted report receipt time per cluster using PostgreSQL's receipt clock. Its alert rules cover stale reports, expected clusters that have never reported, and an unavailable metrics endpoint. Set `monitoring.expectedClusters` and tune `monitoring.staleAfterSeconds` for your scan interval, scan duration, and failover budget. [Prometheus Operator API reference](https://prometheus-operator.dev/docs/api-reference/api/)

## Agent high availability

For a fresh Agent release, set `leaderElection.enabled=true` and `replicaCount=2`. Keep `autoWatch=true` and the HTTP scan controls disabled. Helm creates a namespace-scoped Lease and grants the Agent permission to read and update that Lease. Only its holder starts scans. On leadership loss, the process cancels its work and exits. A standby can take over after the lease expires, normally within about 30 seconds plus the retry interval. Schedule replicas on separate nodes for node-failure coverage. [Kubernetes Leases](https://kubernetes.io/docs/concepts/architecture/leases/)

When upgrading an existing Agent without Lease election, first install the new chart with `leaderElection.enabled=true` and `replicaCount=1`. Wait for a report from the updated Agent, then increase `replicaCount`. This uses a recreate rollout for the first step so the original scanner stops before the Lease holder starts. Inspect the holder with `kubectl -n claritty get lease claritty-agent`.

## GitOps

Use the [Argo CD examples](gitops/argocd/README.md) to pin chart and image revisions. Provision PostgreSQL, DNS, certificates, Secrets, and identity separately using your platform's tooling.

## Verify an installation

```bash
kubectl -n claritty logs deployment/claritty-agent
kubectl -n claritty port-forward service/claritty-hub 8822:8822
```

For the default private install, open `http://localhost:8822` and confirm that your cluster has a report. The [KinD smoke test](../tests/kind/helm-smoke.sh) exercises report delivery, migrations against a legacy database, Lease failover, and a restore drill with disposable PostgreSQL.
