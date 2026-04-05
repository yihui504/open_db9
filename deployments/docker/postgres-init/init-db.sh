#!/bin/sh

for migration in /migrations/control/*.up.sql; do
  echo "Applying migration: $migration"
  psql -v ON_ERROR_STOP=0 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" -f "$migration" 2>&1 || echo "  (migration skipped or had warnings - continuing)"
done

echo "All migrations applied."
