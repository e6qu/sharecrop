#!/usr/bin/env bash
#
# Migration rehearsal: applies this branch's NEW migrations against a scratch
# database that carries the PRE-migration schema of a base ref (default
# origin/main) populated with generated data, and prints how long each new
# migration takes. Use it before merging a migration to see its cost against
# realistically sized tables instead of an empty dev database.
#
# What it does:
#   1. Creates a scratch database on the server named by DATABASE_URL (the
#      connection is used only to issue CREATE/DROP DATABASE; the dev
#      database's contents are never touched).
#   2. Applies the base ref's migrations (git archive <ref> migrations) to
#      the scratch database through the real migration runner.
#   3. Seeds generated rows at an adjustable scale into the tables the
#      pending migrations touch or scan.
#   4. Applies each not-yet-applied migration from ./migrations one at a
#      time (psql, single transaction each — the runner applies all pending
#      files in one transaction, so per-file timing needs psql), printing
#      the duration of each.
#   5. Confirms the migration runner agrees nothing is left to apply, then
#      drops the scratch database.
#
# Environment:
#   DATABASE_URL              required; names the Postgres server (the
#                             connecting role must have CREATEDB).
#   REHEARSE_BASE_REF         base ref for the pre-migration schema
#                             (default origin/main).
#   REHEARSE_USERS            seeded users (default 20000).
#   REHEARSE_TASKS            seeded tasks (default 50000).
#   REHEARSE_LEDGER_ENTRIES   seeded ledger entries (default 100000).
#   REHEARSE_EVENTS           seeded domain events + recipients (default
#                             100000; migration 000046 rewrites this table).
#   REHEARSE_NOTIFICATIONS    seeded notifications (default 100000; 000046
#                             adds a column + partial unique index here).
#
# Measured 2026-08-01 on a local Postgres 15 (Apple silicon) at the default
# scale (20k users, 50k tasks, 100k ledger entries, 100k events, 100k
# notifications), two runs:
#   000046_event_outbox.sql            724 ms / 974 ms
#   000047_submission_superseded.sql    23 ms /  21 ms
set -euo pipefail

if [[ -z "${DATABASE_URL:-}" ]]; then
  echo "DATABASE_URL is required" >&2
  exit 2
fi

base_ref="${REHEARSE_BASE_REF:-origin/main}"
seed_users="${REHEARSE_USERS:-20000}"
seed_tasks="${REHEARSE_TASKS:-50000}"
seed_ledger="${REHEARSE_LEDGER_ENTRIES:-100000}"
seed_events="${REHEARSE_EVENTS:-100000}"
seed_notifications="${REHEARSE_NOTIFICATIONS:-100000}"

# Split DATABASE_URL into server part and query string so the scratch
# database reuses the same server, credentials, and options.
url_no_query="${DATABASE_URL%%\?*}"
url_query=""
if [[ "$DATABASE_URL" == *\?* ]]; then
  url_query="?${DATABASE_URL#*\?}"
fi
scratch_db="sharecrop_rehearsal_$(date +%s)_$$"
scratch_url="${url_no_query%/*}/${scratch_db}${url_query}"

staging="$(mktemp -d)"
cleanup() {
  psql "$DATABASE_URL" -q -c "drop database if exists ${scratch_db} with (force)" || true
  rm -rf "$staging"
}
trap cleanup EXIT

now_ms() {
  perl -MTime::HiRes=time -e 'printf "%d", time()*1000'
}

echo "== scratch database: ${scratch_db}"
psql "$DATABASE_URL" -q -v ON_ERROR_STOP=1 -c "create database ${scratch_db}"

echo "== applying base schema from ${base_ref}"
git archive "$base_ref" migrations | tar -x -C "$staging"
DATABASE_URL="$scratch_url" SHARECROP_MIGRATIONS_DIR="$staging/migrations" \
  go run ./cmd/sharecrop migrate up

# The new migrations are the .sql files present locally but absent from the
# base ref's migrations directory.
new_migrations=()
for path in migrations/*.sql; do
  name="$(basename "$path")"
  if [[ ! -f "$staging/migrations/$name" ]]; then
    new_migrations+=("$name")
  fi
done
if [[ "${#new_migrations[@]}" -eq 0 ]]; then
  echo "no new migrations relative to ${base_ref}; nothing to rehearse"
  exit 0
fi
echo "== new migrations: ${new_migrations[*]}"

echo "== seeding ${seed_users} users, ${seed_tasks} tasks, ${seed_ledger} ledger entries, ${seed_events} events, ${seed_notifications} notifications"
psql "$scratch_url" -q -v ON_ERROR_STOP=1 \
  -v users="$seed_users" -v tasks="$seed_tasks" -v ledger="$seed_ledger" \
  -v events="$seed_events" -v notifications="$seed_notifications" <<'SQL'
-- Deterministic ids: one fixed prefix nibble per entity, the counter in the
-- low 12 hex digits.
insert into users (id, email, display_name)
select ('10000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       'seed-' || g || '@rehearsal.invalid',
       'Seed User ' || g
from generate_series(1, :users) as g;

insert into credit_accounts (id, owner_kind, user_id)
select ('30000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       'user',
       ('10000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid
from generate_series(1, :users) as g;

insert into tasks (id, owner_kind, user_id, title, description, reward_kind,
                   reward_credit_amount, state, response_schema_json,
                   data_payload_kind, created_by_user_id)
select ('20000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       'user', seed.uid, 'Seed task ' || g, 'Rehearsal seed task ' || g,
       'credit', 25,
       case when g % 2 = 0 then 'open' else 'draft' end,
       '{}'::jsonb, 'none', seed.uid
from (
  select g, ('10000000-0000-4000-8000-' || lpad(to_hex((g % :users) + 1), 12, '0'))::uuid as uid
  from generate_series(1, :tasks) as g
) as seed;

insert into ledger_entries (id, account_id, kind, amount, idempotency_key)
select ('40000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       ('30000000-0000-4000-8000-' || lpad(to_hex((g % :users) + 1), 12, '0'))::uuid,
       'manual_adjustment', 5, 'seed-' || g
from generate_series(1, :ledger) as g;

insert into domain_events (id, kind, actor_kind, actor_user_id, task_id, occurred_at)
select ('50000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       'task_opened', 'user',
       ('10000000-0000-4000-8000-' || lpad(to_hex((g % :users) + 1), 12, '0'))::uuid,
       ('20000000-0000-4000-8000-' || lpad(to_hex((g % :tasks) + 1), 12, '0'))::uuid,
       now() - make_interval(secs => g % 86400)
from generate_series(1, :events) as g;

insert into domain_event_recipients (event_seq, user_id)
select seq, actor_user_id from domain_events where actor_user_id is not null;

insert into notifications (id, recipient_user_id, actor_user_id, kind,
                           subject_kind, subject_id, state)
select ('60000000-0000-4000-8000-' || lpad(to_hex(g), 12, '0'))::uuid,
       ('10000000-0000-4000-8000-' || lpad(to_hex((g % :users) + 1), 12, '0'))::uuid,
       ('10000000-0000-4000-8000-' || lpad(to_hex(((g + 1) % :users) + 1), 12, '0'))::uuid,
       'submission_created', 'task',
       '20000000-0000-4000-8000-' || lpad(to_hex((g % :tasks) + 1), 12, '0'),
       case when g % 2 = 0 then 'unread' else 'read' end
from generate_series(1, :notifications) as g;

vacuum analyze users, credit_accounts, tasks, ledger_entries, domain_events,
  domain_event_recipients, notifications;
SQL

echo "== rehearsing"
for name in "${new_migrations[@]}"; do
  started="$(now_ms)"
  psql "$scratch_url" -q -v ON_ERROR_STOP=1 -1 -f "migrations/$name"
  finished="$(now_ms)"
  psql "$scratch_url" -q -v ON_ERROR_STOP=1 \
    -c "insert into schema_migrations (name) values ('${name}')"
  printf '%-45s %6d ms\n' "$name" "$((finished - started))"
done

# The runner must agree the rehearsed database is fully migrated.
DATABASE_URL="$scratch_url" SHARECROP_MIGRATIONS_DIR="$PWD/migrations" \
  go run ./cmd/sharecrop migrate up > /dev/null
echo "== migration runner confirms the schema is current; dropping ${scratch_db}"
