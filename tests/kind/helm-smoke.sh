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

kubectl create namespace claritty-db
kubectl -n claritty-db apply -f tests/kind/postgres.yaml
kubectl -n claritty-db rollout status deployment/postgres --timeout=120s

kubectl create namespace claritty
kubectl label namespace claritty pod-security.kubernetes.io/enforce=restricted
kubectl -n claritty create secret generic hub-secrets \
  --from-literal=DATABASE_URL='postgres://claritty:kind-test-only@postgres.claritty-db:5432/claritty?sslmode=disable' \
  --from-literal=CLARITTY_HUB_API_KEY=kind-test-only
kubectl -n claritty create secret generic agent-secrets \
  --from-literal=CLARITTY_HUB_API_KEY=kind-test-only

helm install claritty-hub infra/helm/claritty-hub -n claritty --wait --timeout 120s "${hub_values[@]}"
helm install claritty-agent infra/helm/claritty-agent -n claritty --wait --timeout 120s "${agent_values[@]}"

agent_account=system:serviceaccount:claritty:claritty-agent
kubectl auth can-i --quiet --as="$agent_account" list configmaps -n claritty
kubectl auth can-i --quiet --as="$agent_account" list events -n claritty
if kubectl auth can-i --quiet --as="$agent_account" list pods -n claritty-db; then
  echo 'Agent can list pods outside its configured namespace' >&2
  exit 1
fi

kubectl -n claritty port-forward service/claritty-hub 18822:8822 &
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
