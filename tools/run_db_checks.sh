#!/usr/bin/env bash
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 2
fi

if [[ -z "${SHARECROP_MIGRATIONS_DIR:-}" ]]; then
  echo "SHARECROP_MIGRATIONS_DIR is required" >&2
  exit 2
fi

go run ./cmd/sharecrop migrate up
go test -tags integration ./tests/integration
go test -tags http_e2e ./tests/http_e2e
