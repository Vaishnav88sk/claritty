# PostgreSQL operations

## Schema changes and credentials

Use PostgreSQL with TLS and hostname verification in production. Put the database URLs in Kubernetes Secrets or in your secret manager. The Hub chart's pre-install and pre-upgrade Job runs `claritty-hub --migrate`. Migrations use a PostgreSQL transaction and advisory lock, so retries skip committed versions and a failed migration rolls back with its ledger entry. Schedule upgrades in a maintenance window for large incident tables. The receipt timestamp migration backfills existing incidents and builds a new index.

Use a schema owner account for the migration Job and a separate login for the Hub. Bootstrap the schema with `claritty-hub --migrate` using the owner account, then grant the runtime role the privileges below. The runtime role also needs `CONNECT` on the target database and `USAGE` on the schema.

```sql
GRANT SELECT ON TABLE claritty_schema_migrations TO claritty_runtime;
GRANT SELECT, INSERT, UPDATE ON TABLE clusters, incidents TO claritty_runtime;
```

Point `migrations.existingSecret` to the owner's Secret and `existingSecret` to the runtime Secret. The migration Job uses the runtime Secret by default for compatibility with existing installs. Set `migrations.enabled=false` when your deployment system runs migrations separately. The Hub checks the required schema version before serving. Local Compose and direct runs still migrate on startup unless `HUB_AUTO_MIGRATE=false`.

Plan rollback against a compatible schema. The migrations here are additive. Restoring an earlier database state requires a separate recovery operation.

## Backups and recovery drills

Enable automated physical backups and WAL archiving or your PostgreSQL service's point-in-time recovery. Set retention from your recovery point objective and test recovery to a chosen timestamp against your recovery time objective. Store backups encrypted with access separate from runtime credentials. Periodically restore into an isolated database and verify the dashboard, migration ledger, and representative incident IDs before recording the measured recovery time. [PostgreSQL point-in-time recovery](https://www.postgresql.org/docs/16/continuous-archiving.html)

For a portable logical restore drill, configure distinct libpq service names for the source and a disposable, empty target. Use `PGSERVICEFILE` and a protected password file or your credential provider. Run the script with PostgreSQL client tools at least as new as the source server. Pass a writable directory on encrypted storage.

```bash
sh infra/postgres/restore-drill.sh claritty-source claritty-restore /secure/backups
```

The script makes a custom-format dump, restores it in one transaction, and queries the restored migration, cluster, and incident tables. It refuses a nonempty target. The dump stays in the requested directory for inspection. Recreate production ownership and grants explicitly when promoting a recovered database, since the drill intentionally restores without source ownership or grants. [PostgreSQL restore options](https://www.postgresql.org/docs/16/app-pgrestore.html)
