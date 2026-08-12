#!/bin/bash
# Creates the runtime role the API connects as.
#
# This role is deliberately NOT the database owner and NOT a superuser: Postgres
# lets superusers and BYPASSRLS roles ignore row-level security entirely, so
# connecting the API as ${POSTGRES_USER} would make every tenant_isolation policy
# a no-op. Migrations still run as the owner via DATABASE_MIGRATE_URL.
set -euo pipefail

if [ -z "${APP_DB_PASSWORD:-}" ]; then
  echo "init.sh: APP_DB_PASSWORD is required so the runtime role has a real password" >&2
  exit 1
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
	DO \$\$
	BEGIN
	  IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'unicore_app') THEN
	    CREATE ROLE unicore_app LOGIN;
	  END IF;
	END
	\$\$;

	ALTER ROLE unicore_app WITH PASSWORD '${APP_DB_PASSWORD}' NOSUPERUSER NOBYPASSRLS NOCREATEDB NOCREATEROLE;

	GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO unicore_app;
	GRANT USAGE ON SCHEMA public TO unicore_app;
EOSQL
