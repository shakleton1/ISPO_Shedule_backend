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

# Install goose if not present
if ! command -v goose &> /dev/null; then
  echo "Installing goose..."
  go install github.com/pressly/goose/v3/cmd/goose@latest
fi

# Run migrations
export GOOSE_DRIVER=postgres
export GOOSE_DBSTRING="host=postgres port=5432 user=postgres password=${ISPO_DB_PASSWORD:-postgres} dbname=ispo_schedule sslmode=disable"

goose -dir /app/db/migrations up

echo "Migrations applied successfully!"

# Start the API
echo "Starting API server..."
exec /app/api
