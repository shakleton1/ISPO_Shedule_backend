#!/bin/sh
set -e

echo "Starting ISPO Schedule API..."

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until pg_isready -h postgres -p 5432 -U postgres; do
  echo "PostgreSQL is unavailable - sleeping"
  sleep 2
done

echo "PostgreSQL is up - applying migrations..."

# Run migrations (goose is already installed in the image)
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="host=postgres port=5432 user=postgres password=${ISPO_DB_PASSWORD:-postgres} dbname=ispo_schedule sslmode=disable"

# Try to run migrations, but don't fail if they already exist
# This allows the app to start even if migrations were applied before
if goose -dir /app/db/migrations up; then
  echo "Migrations applied successfully!"
else
  echo "Warning: Migration failed or already applied, continuing..."
fi

# Start the API
echo "Starting API server..."
exec /app/api
