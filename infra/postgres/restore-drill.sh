#!/bin/sh
set -eu

if [ "$#" -ne 3 ] || [ -z "$1" ] || [ -z "$2" ] || [ "$1" = "$2" ]; then
  echo 'Usage: restore-drill.sh SOURCE_PGSERVICE RESTORE_PGSERVICE BACKUP_DIRECTORY' >&2
  exit 2
fi

source_service=$1
restore_service=$2
backup_directory=$3

# Fail before dumping if the target is not an empty disposable database.
empty=$(psql -XAtq --set=ON_ERROR_STOP=1 --dbname="service=$restore_service" --command="
  SELECT NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
    WHERE n.nspname !~ '^pg_' AND n.nspname <> 'information_schema'
      AND c.relkind IN ('r', 'p', 'v', 'm', 'f', 'S')
  )")
if [ "$empty" != t ]; then
  echo 'The restore service must point to an empty database' >&2
  exit 1
fi

umask 077
backup=$(mktemp "$backup_directory/claritty.dump.XXXXXX")
if ! pg_dump --dbname="service=$source_service" --format=custom --file="$backup"; then
  rm -f "$backup"
  exit 1
fi
printf 'Backup written to %s\n' "$backup"

pg_restore --dbname="service=$restore_service" --no-owner --no-privileges \
  --single-transaction --exit-on-error "$backup"

psql -X --set=ON_ERROR_STOP=1 --dbname="service=$restore_service" --command='
  SELECT (SELECT count(*) FROM claritty_schema_migrations) AS migrations,
         (SELECT count(*) FROM clusters) AS clusters,
         (SELECT count(*) FROM incidents) AS incidents;
'
