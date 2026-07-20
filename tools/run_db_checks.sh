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
# behavior with nothing catching it. Run it here against BOTH real-backend
# modes: the native in-process mux AND the WASI guest (now the production
# default), so "parity" covers every place the real backend runs.
#
# Both modes run from ONE directly-executed binary that embeds the guest, not
# `go run`: a backgrounded `go run` spawns a child a plain `kill` on its own PID
# doesn't reliably reach, so a stale server can outlive its run and answer the
# next run's health probe from the wrong process. A real binary's own PID is the
# listener, so teardown is reliable. `native` mode is selected by env var; the
# embedded guest is simply ignored there.
scenario_parity_port="${SHARECROP_HTTP_ADDR##*:}"
scenario_parity_binary=""

# One cleanup path covers every exit: kill whatever is still listening on the
# scenario port (a server that failed to tear down cleanly), drop the temp
# binary, and restore the committed empty guest placeholder so the working tree
# is left clean.
cleanup_scenario_parity() {
  lsof -nP -iTCP:"${scenario_parity_port}" -sTCP:LISTEN -t 2>/dev/null | xargs -r kill -9 2>/dev/null || true
  [[ -n "$scenario_parity_binary" ]] && rm -f "$scenario_parity_binary"
  : > internal/wasiguest/app-guest.wasm 2>/dev/null || true
}
trap cleanup_scenario_parity EXIT

wait_for_port_free() {
  for _ in $(seq 1 50); do
    if ! lsof -nP -iTCP:"${scenario_parity_port}" -sTCP:LISTEN > /dev/null 2>&1; then
      return 0
    fi
    sleep 0.2
  done
  echo "port ${scenario_parity_port} never freed before the next scenario server" >&2
  exit 1
}

# run_scenario_parity starts a server, waits for it to become healthy, runs the
# real-backend scenario against it, then tears it down. Output is redirected to
# a file rather than left on this script's stdout/stderr so a hung server can
# never hold the script's pipe open.
run_scenario_parity() {
  local label="$1"
  shift
  local log
  log="$(mktemp)"
  "$@" > "$log" 2>&1 &
  local pid=$!

  local origin="$SHARECROP_HTTP_ADDR"
  if [[ "$origin" == :* ]]; then
    origin="127.0.0.1${origin}"
  fi
  origin="http://${origin}"
  local ready=0
  # The WASI guest pool takes longer to warm than the native mux, so allow a
  # generous readiness budget.
  for _ in $(seq 1 150); do
    # If our server process died (e.g. it failed to bind because a stale server
    # still holds the port), stop waiting - otherwise a lingering process could
    # answer the probe and we'd silently test the wrong server.
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "${label} scenario server exited before becoming healthy; log:" >&2
      cat "$log" >&2
      exit 1
    fi
    if curl -sf "${origin}/healthz" > /dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 0.2
  done
  if [[ "$ready" -ne 1 ]]; then
    echo "${label} scenario server never became healthy; log:" >&2
    cat "$log" >&2
    exit 1
  fi

  # For the WASI run, prove the guest pool was actually used (not a silent
  # fallback to the native mux).
  if [[ "$label" == "wasi" ]] && ! grep -q "WASI guest pool" "$log"; then
    echo "wasi scenario server did not use the guest pool; log:" >&2
    cat "$log" >&2
    exit 1
  fi

  deno run --allow-net --allow-env --allow-run tools/run_local_real_scenario_parity.ts -- --origin "${origin}"

  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  wait_for_port_free
}

# Build the guest into the embed path and compile ONE binary that bakes it in -
# the same artifact production runs.
make wasi-app-guest
scenario_parity_binary="$(mktemp -u)"
go build -o "$scenario_parity_binary" ./cmd/sharecrop

# 1) Native in-process mux. 2) WASI guest (the production default).
run_scenario_parity native env SHARECROP_WASI_MODE=native "$scenario_parity_binary" serve
run_scenario_parity wasi "$scenario_parity_binary" serve
