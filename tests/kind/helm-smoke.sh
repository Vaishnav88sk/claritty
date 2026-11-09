#!/usr/bin/env bash
set -euo pipefail

agent_values=(
  --set-string image=claritty-agent:kind
  --set-string clusterName=kind-smoke
  --set-string hubURL=http://claritty-hub:8822
  --set-string existingSecret=agent-secrets
  --set-string llm.provider=mock
  --set 'watchNamespaces[0]=claritty'
)
hub_values=(
  --set-string image=claritty-hub:kind
  --set-string existingSecret=hub-secrets
)

helm lint infra/helm/claritty-agent "${agent_values[@]}"
helm lint infra/helm/claritty-hub "${hub_values[@]}"
helm template claritty-agent infra/helm/claritty-agent -n claritty "${agent_values[@]}" |
  yamllint -c .yamllint -
helm template claritty-hub infra/helm/claritty-hub -n claritty "${hub_values[@]}" |
  yamllint -c .yamllint -
helm template claritty-hub infra/helm/claritty-hub -n claritty "${hub_values[@]}" \
  -f infra/examples/hub-production.values.yaml | yamllint -c .yamllint -
helm template claritty-agent infra/helm/claritty-agent -n claritty "${agent_values[@]}" \
  --set leaderElection.enabled=true,replicaCount=2,monitoring.enabled=true | yamllint -c .yamllint -

kubectl create namespace claritty-db
kubectl -n claritty-db apply -f tests/kind/postgres.yaml
kubectl -n claritty-db rollout status deployment/postgres --timeout=120s

db_sql() {
  kubectl -n claritty-db exec -i deployment/postgres -- psql -XAtq -U claritty -v ON_ERROR_STOP=1 "$@"
}

# Start with a populated database from the release before the migration ledger.
cat sre-agent/hub/internal/db/migrations/0001_initial.sql \
  sre-agent/hub/internal/db/migrations/0002_evidence.sql | db_sql
db_sql <<'SQL'
INSERT INTO clusters (name, last_seen) VALUES ('kind-legacy', '2024-01-01 00:00:00');
INSERT INTO incidents (id, cluster, detected_at) VALUES ('INC-LEGACY', 'kind-legacy', '2024-01-01 00:00:00');
CREATE ROLE claritty_runtime LOGIN PASSWORD 'kind-runtime-only';
GRANT SELECT, INSERT, UPDATE ON clusters, incidents TO claritty_runtime;
ALTER DEFAULT PRIVILEGES FOR ROLE claritty IN SCHEMA public GRANT SELECT ON TABLES TO claritty_runtime;
SQL

kubectl create namespace claritty
kubectl label namespace claritty pod-security.kubernetes.io/enforce=restricted
kubectl -n claritty create secret generic hub-secrets \
  --from-literal=DATABASE_URL='postgres://claritty_runtime:kind-runtime-only@postgres.claritty-db:5432/claritty?sslmode=disable' \
  --from-literal=CLARITTY_HUB_API_KEY=kind-test-only
kubectl -n claritty create secret generic hub-migration \
  --from-literal=DATABASE_URL='postgres://claritty:kind-test-only@postgres.claritty-db:5432/claritty?sslmode=disable'
kubectl -n claritty create secret generic agent-secrets \
  --from-literal=CLARITTY_HUB_API_KEY=kind-test-only

hub_values+=(--set-string migrations.existingSecret=hub-migration)
helm install claritty-hub infra/helm/claritty-hub -n claritty --wait --timeout 120s "${hub_values[@]}"
# Re-run the hook to verify a second migration attempt is safe.
helm upgrade claritty-hub infra/helm/claritty-hub -n claritty --wait --timeout 120s "${hub_values[@]}"
[[ $(db_sql -c 'SELECT count(*) FROM claritty_schema_migrations') == 3 ]]
[[ $(db_sql -c "SELECT has_table_privilege('claritty_runtime', 'claritty_schema_migrations', 'INSERT')") == f ]]
helm install claritty-agent infra/helm/claritty-agent -n claritty --wait --timeout 120s "${agent_values[@]}"

agent_account=system:serviceaccount:claritty:claritty-agent
kubectl auth can-i --quiet --as="$agent_account" list configmaps -n claritty
kubectl auth can-i --quiet --as="$agent_account" list events -n claritty
if kubectl auth can-i --quiet --as="$agent_account" list pods -n claritty-db; then
  echo 'Agent can list pods outside its configured namespace' >&2
  exit 1
fi

kubectl -n claritty port-forward deployment/claritty-hub 18822:8822 18824:8824 &
forward_pid=$!
trap 'kill "$forward_pid" 2>/dev/null || true' EXIT

incidents_url='http://127.0.0.1:18822/api/v1/incidents'

await_report() {
  local previous_id=$1 report_id deadline=$((SECONDS + 120))
  while (( SECONDS < deadline )); do
    if report_id=$(curl --fail --silent --max-time 3 "${incidents_url}?cluster=kind-smoke" |
      jq -er '.[0] | select(.title == "Mock scan completed" and .severity == "SEV4") | .id') &&
      [[ -n $report_id && $report_id != "$previous_id" ]]; then
      printf '%s\n' "$report_id"
      return 0
    fi
    sleep 2
  done
  echo 'No new Agent report was persisted by the Hub' >&2
  return 1
}

first_report=$(await_report '')
printf 'Initial report persisted: %s\n' "$first_report"
metrics=$(curl --fail --silent --max-time 5 http://127.0.0.1:18824/metrics)
grep -Eq '^claritty_hub_cluster_last_report_received_timestamp_seconds\{cluster="kind-smoke"\} [1-9]' <<< "$metrics"
grep -q '^claritty_hub_cluster_last_report_received_timestamp_seconds{cluster="kind-legacy"}' <<< "$metrics"

http_status=$(curl --silent --max-time 3 --output /dev/null --write-out '%{http_code}' \
  --request POST --data '{}' "$incidents_url")
[[ $http_status == 401 ]]

old_agent=$(kubectl -n claritty get pod -l app.kubernetes.io/instance=claritty-agent -o name)
[[ -n $old_agent ]]
kubectl -n claritty scale deployment/claritty-agent --replicas=0
kubectl -n claritty wait --for=delete --timeout=60s "$old_agent"
last_report=$(await_report '')
kubectl -n claritty scale deployment/claritty-agent --replicas=1
kubectl -n claritty rollout status deployment/claritty-agent --timeout=120s

next_report=$(await_report "$last_report")
printf 'Report persisted after Agent restart: %s\n' "$next_report"

# Enable Lease election on the singleton before changing the rollout strategy.
agent_values+=(--set leaderElection.enabled=true)
helm upgrade claritty-agent infra/helm/claritty-agent -n claritty --wait --timeout 120s "${agent_values[@]}"
await_report "$next_report"
helm upgrade claritty-agent infra/helm/claritty-agent -n claritty --wait --timeout 120s \
  "${agent_values[@]}" --set replicaCount=2,scanIntervalSeconds=30

kubectl auth can-i --quiet --as="$agent_account" update lease/claritty-agent -n claritty
if kubectl auth can-i --quiet --as="$agent_account" update lease/other -n claritty; then
  echo 'Agent can update a Lease outside its release' >&2
  exit 1
fi

await_holder() {
  local previous=$1 holder deadline=$((SECONDS + 120))
  while (( SECONDS < deadline )); do
    holder=$(kubectl -n claritty get lease claritty-agent -o jsonpath='{.spec.holderIdentity}')
    if [[ -n $holder && $holder != "$previous" ]] &&
      kubectl -n claritty wait --for=condition=Ready "pod/$holder" --timeout=2s >/dev/null 2>&1; then
      printf '%s\n' "$holder"
      return 0
    fi
    sleep 2
  done
  echo 'No new ready Lease holder was elected' >&2
  return 1
}

holder=$(await_holder '')
agent_metrics=$(kubectl -n claritty exec "$holder" -- wget -qO- http://127.0.0.1:9090/metrics)
grep -qx 'claritty_agent_leader 1' <<< "$agent_metrics"
kubectl -n claritty delete pod "$holder" --wait=true
last_held_report=$(await_report '')
new_holder=$(await_holder "$holder")
failover_report=$(await_report "$last_held_report")
printf 'Report persisted after Lease failover to %s: %s\n' "$new_holder" "$failover_report"

db_sql -c 'CREATE DATABASE claritty_restore'
kubectl -n claritty-db exec -i deployment/postgres -- sh -c '
  set -eu
  scratch=$(mktemp -d)
  export PGSERVICEFILE="$scratch/pg_service.conf"
  cat > "$PGSERVICEFILE" <<EOF
[source]
dbname=claritty
user=claritty
[restore]
dbname=claritty_restore
user=claritty
EOF
  sh -s -- source restore "$scratch"
' < infra/postgres/restore-drill.sh
[[ $(db_sql -d claritty_restore -c 'SELECT count(*) FROM claritty_schema_migrations') == 3 ]]
[[ $(db_sql -d claritty_restore -c "SELECT count(*) FROM incidents WHERE id IN ('INC-LEGACY', '$failover_report')") == 2 ]]
