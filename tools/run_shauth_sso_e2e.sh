#!/bin/sh
# SPDX-License-Identifier: AGPL-3.0-or-later
set -eu

unset CDPATH
root=$(CDPATH='' cd -- "$(dirname -- "$0")/.." && pwd)
cd "$root"

shauth_source=${SHAUTH_SOURCE_DIR:?SHAUTH_SOURCE_DIR must point to the Shauth checkout under test}
expected_shauth_commit=${SHAUTH_EXPECTED_COMMIT:-470f7890ce6f0391bca3e4f6ce4ef8a17f1c7933}
actual_shauth_commit=$(git -C "$shauth_source" rev-parse HEAD)
if [ "$actual_shauth_commit" != "$expected_shauth_commit" ]; then
	printf 'Shauth checkout is %s; expected %s\n' "$actual_shauth_commit" "$expected_shauth_commit" >&2
	exit 1
fi
compose_project=sharecrop-shauth-e2e
compose_file=$shauth_source/compose.yaml
compose_override=$root/tests/shauth-compose.override.yaml
application=http://localhost:29382
issuer=http://localhost:58080
compose() {
	docker compose --project-directory "$shauth_source" -f "$compose_file" -f "$compose_override" -p "$compose_project" "$@"
}
temporary=$(mktemp -d)
trap 'rm -rf "$temporary"' EXIT INT TERM
docker_arch=$(docker version --format '{{.Server.Arch}}')
case "$docker_arch" in
	amd64 | x86_64) docker_arch=amd64 ;;
	arm64 | aarch64) docker_arch=arm64 ;;
	*) printf 'unsupported Docker server architecture %s\n' "$docker_arch" >&2; exit 1 ;;
esac
SHARECROP_BACKCHANNEL_FORWARDER=$temporary/backchannel-forwarder
export SHARECROP_BACKCHANNEL_FORWARDER
CGO_ENABLED=0 GOOS=linux GOARCH=$docker_arch go build -o "$SHARECROP_BACKCHANNEL_FORWARDER" ./tests/backchannel-forwarder
random_secret() {
	openssl rand -base64 48 | tr -d '\n'
}

POSTGRES_PASSWORD=$(openssl rand -hex 32)
export POSTGRES_PASSWORD
HYDRA_SYSTEM_SECRET=$(random_secret)
export HYDRA_SYSTEM_SECRET
export HYDRA_DSN="postgres://shauth:${POSTGRES_PASSWORD}@postgres:5432/hydra?sslmode=disable"
export HYDRA_PUBLIC_URL=$issuer
export SHAUTH_PUBLIC_URL=$issuer
export SHAUTH_DATABASE_URL="postgres://shauth:${POSTGRES_PASSWORD}@postgres:5432/shauth?sslmode=disable"
export GITHUB_CLIENT_ID=sharecrop-contract-test
export GITHUB_CLIENT_SECRET=sharecrop-contract-test-secret
SHAUTH_BOOTSTRAP_ADMIN_PASSWORD=$(random_secret)
export SHAUTH_BOOTSTRAP_ADMIN_PASSWORD
sharecrop_client_secret=$(random_secret)
export SHAUTH_BOOTSTRAP_APPS_JSON="[{\"slug\":\"sharecrop-e2e\",\"name\":\"Sharecrop\",\"description\":\"Sharecrop relying-party contract.\",\"launch_url\":\"${application}/\",\"oidc_client_id\":\"sharecrop-e2e\",\"oidc_client_secret\":\"${sharecrop_client_secret}\",\"redirect_uris\":[\"${application}/api/auth/shauth/callback\"],\"post_logout_redirect_uris\":[\"${application}/api/auth/signed-out\"],\"frontchannel_logout_uri\":\"${application}/api/auth/shauth/frontchannel-logout\",\"backchannel_logout_uri\":\"${application}/api/auth/shauth/backchannel-logout\",\"health_url\":\"${application}/healthz\",\"monitoring_url\":\"\"}]"

sharecrop_pid=
sharecrop_log=$temporary/sharecrop.log
cleanup() {
	status=$?
	if [ "$status" -ne 0 ]; then
		printf '%s\n' 'Sharecrop log:' >&2
		tail -n 160 "$sharecrop_log" >&2 || true
		compose logs --no-color --tail=160 shauth hydra postgres >&2 || true
	fi
	if [ -n "$sharecrop_pid" ]; then
		kill "$sharecrop_pid" 2>/dev/null || true
		wait "$sharecrop_pid" 2>/dev/null || true
	fi
	compose down --volumes --remove-orphans >/dev/null 2>&1 || true
	: > internal/wasiguest/app-guest.wasm
	rm -rf "$temporary"
	exit "$status"
}
trap cleanup EXIT INT TERM

compose down --volumes --remove-orphans >/dev/null 2>&1 || true
compose up --build --detach

attempt=0
while [ "$attempt" -lt 180 ]; do
	if curl --fail --silent "$issuer/healthz" >/dev/null 2>&1 && curl --fail --silent http://localhost:58444/health/ready >/dev/null 2>&1; then
		break
	fi
	attempt=$((attempt + 1))
	sleep 1
done
if [ "$attempt" -eq 180 ]; then
	printf '%s\n' 'Shauth did not become healthy' >&2
	exit 1
fi

compose exec -T postgres \
	psql -v ON_ERROR_STOP=1 -U shauth -d postgres -c 'CREATE DATABASE sharecrop_e2e'

export DATABASE_URL="postgres://shauth:${POSTGRES_PASSWORD}@127.0.0.1:56432/sharecrop_e2e?sslmode=disable"
export SHARECROP_HTTP_ADDR=0.0.0.0:29382
export SHARECROP_ACCESS_TOKEN_SECRET
SHARECROP_ACCESS_TOKEN_SECRET=$(random_secret)
export SHARECROP_MIGRATIONS_DIR="$root/migrations"
export SHARECROP_INSECURE_COOKIES=true
export SHARECROP_SHAUTH_ISSUER=$issuer
export SHARECROP_SHAUTH_CLIENT_ID=sharecrop-e2e
export SHARECROP_SHAUTH_CLIENT_SECRET="$sharecrop_client_secret"
export SHARECROP_PUBLIC_URL=$application

make build
./bin/sharecrop migrate up
./bin/sharecrop serve >"$sharecrop_log" 2>&1 &
sharecrop_pid=$!

attempt=0
while [ "$attempt" -lt 60 ] && ! curl --fail --silent "$application/healthz" >/dev/null 2>&1; do
	attempt=$((attempt + 1))
	sleep 1
done
if [ "$attempt" -eq 60 ]; then
	printf '%s\n' 'Sharecrop did not become healthy' >&2
	exit 1
fi

database_name=$(compose exec -T postgres \
	psql -U shauth -d sharecrop_e2e -Atc 'SELECT current_database()')
if [ "$database_name" != sharecrop_e2e ]; then
	printf 'Sharecrop database isolation check returned %s\n' "$database_name" >&2
	exit 1
fi

SHARECROP_SHAUTH_E2E_ISSUER=$issuer \
SHARECROP_SHAUTH_E2E_APPLICATION=$application \
deno run --allow-env --allow-net --allow-run --allow-read --allow-sys --allow-write \
	tests/playwright/shauth_sso_e2e.mjs

attempt=0
backchannel_claims=0
while [ "$attempt" -lt 50 ]; do
	backchannel_claims=$(compose exec -T postgres \
		psql -U shauth -d sharecrop_e2e -Atc 'SELECT count(*) FROM oidc_logout_claims')
	if [ "$backchannel_claims" -gt 0 ]; then
		break
	fi
	attempt=$((attempt + 1))
	sleep 0.1
done
if [ "$backchannel_claims" -lt 1 ]; then
	printf '%s\n' 'Shauth did not deliver a valid expiring Back-Channel Logout token' >&2
	exit 1
fi
