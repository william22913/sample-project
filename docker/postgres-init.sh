#!/bin/sh
# Applies only the Up half of the migration.
#
# sql_migrations/01_init_schema.sql is a sql-migrate file: the SQL after
# `-- +migrate Down` is live DROP statements, not comments. piped to psql whole,
# the file would create the schema and then immediately drop it, leaving the
# service to boot against an empty database - and the only symptom would be
# "relation teachers does not exist" on the first request.
#
# The application deliberately does not run migrations (see main.go), so this is
# the single point where the schema is created. It runs once, on a fresh volume:
# the postgres image executes docker-entrypoint-initdb.d only when it
# initialises a data directory.
set -e

sed -n '1,/^-- +migrate Down/p' /migrations/01_init_schema.sql \
    | psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB"
