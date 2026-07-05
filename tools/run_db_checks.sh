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

if [[ -z "${SHARECROP_HTTP_ADDR:-}" ]]; then
  echo "SHARECROP_HTTP_ADDR is required" >&2
  exit 2
fi

go run ./cmd/sharecrop migrate up
go test -tags integration ./tests/integration
go test -tags http_e2e ./tests/http_e2e

# The shared scenario_parity script (tests/scenario_parity/scenario.ts) is
# also run against the WASM demo backend in CI (check-wasm-scenario-parity),
# but until now was never run against the real backend anywhere in CI - so a
# new assertion added there could silently diverge from real-backend
# behavior with nothing catching it. Run it here too, against a real,
# DB-backed server, so "parity" actually means both backends were checked.
#
# Output is redirected to a file, not left attached to this script's own
# stdout/stderr: `go run` spawns a child binary process that a plain `kill`
# on its own PID doesn't reliably reach, so an unredirected background
# server can outlive this script and hold its pipe open indefinitely.
scenario_parity_log="$(mktemp)"
go run ./cmd/sharecrop serve > "$scenario_parity_log" 2>&1 &
scenario_parity_server_pid=$!
trap 'pkill -P "$scenario_parity_server_pid" 2>/dev/null || true; kill "$scenario_parity_server_pid" 2>/dev/null || true' EXIT

scenario_parity_origin="http://127.0.0.1${SHARECROP_HTTP_ADDR}"
scenario_parity_ready=0
for _ in $(seq 1 50); do
  if curl -sf "${scenario_parity_origin}/healthz" > /dev/null 2>&1; then
    scenario_parity_ready=1
    break
  fi
  sleep 0.2
done
if [[ "$scenario_parity_ready" -ne 1 ]]; then
  echo "scenario parity server never became healthy; log:" >&2
  cat "$scenario_parity_log" >&2
  exit 1
fi

deno run --allow-net --allow-env --allow-run tools/run_local_real_scenario_parity.ts -- --origin "${scenario_parity_origin}"

pkill -P "$scenario_parity_server_pid" 2>/dev/null || true
kill "$scenario_parity_server_pid" 2>/dev/null || true
trap - EXIT
