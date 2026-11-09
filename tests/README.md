# Integration tests

The [Helm smoke test](kind/helm-smoke.sh) installs the Agent and Hub into a fresh KinD cluster with disposable PostgreSQL. It uses the mock LLM provider and checks authenticated delivery, scoped RBAC, Hub freshness metrics, migration of legacy data, Agent restart and Lease failover, and restoration into an empty database. The [Helm Deployment Tests workflow](../.github/workflows/infra.yaml) builds and loads both images before running the test.
