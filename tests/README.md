# Integration tests

The [Helm smoke test](kind/helm-smoke.sh) installs the Agent and Hub into a fresh KinD cluster with disposable PostgreSQL. It uses the mock LLM provider and checks authenticated delivery, namespace-scoped RBAC, and report delivery after an Agent restart. The [Helm Deployment Tests workflow](../.github/workflows/infra.yaml) builds and loads both images before running the test.
