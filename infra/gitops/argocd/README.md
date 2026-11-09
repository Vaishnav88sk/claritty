# Argo CD deployment

Copy these Applications to your environment repository. Pin `targetRevision` to a tag or commit containing these charts, set the image digests, and override the production values for your platform. Select an Argo CD Project and destination that can manage the release namespace and the Agent's scoped cluster RBAC. Use a distinct Application name and `clusterName` for each monitored cluster. If you rename the Agent Application, update the matching Lease name in `ignoreDifferences`.

Provision namespaces, Secrets, the Gateway and its certificate, and the Gateway API and Prometheus Operator CRDs before syncing the Hub. Argo CD runs the Hub migration hook in `PreSync`, so the database and migration Secret must already be available. Sync the Hub first and verify its HTTPS routes, then sync the Agent Applications. For an existing singleton Agent, follow the two-step migration in the [operations guide](../../README.md).

Argo CD owns the Lease resource. The Agent owns the Lease's `spec` during operation. The ignore rule and `RespectIgnoreDifferences` let Argo CD reconcile the chart without disturbing a live election. Use a full sync when upgrading the Hub so its migration hook runs.
