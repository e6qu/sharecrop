-- Scope ledger idempotency keys per funding account instead of globally.
-- Two unrelated actors reusing the same client-chosen key (for example
-- "retry-1") must not collide with each other; uniqueness is a per-account
-- replay guarantee, not a platform-wide one. The inline global constraint was
-- removed from 000006 for fresh builds; this drop covers databases migrated
-- before this file existed. SQLite skips the drop (unsupported ALTER) and
-- relies on the edited 000006.
alter table ledger_entries
	drop constraint if exists ledger_entries_idempotency_key_unique;

create unique index if not exists ledger_entries_account_idempotency_uidx
	on ledger_entries(account_id, idempotency_key)
	where idempotency_key is not null;

-- Request-changes gains its own idempotency key. It cannot share
-- review_idempotency_key: a submission can be changes-requested and later
-- rejected, and the two commands must replay independently.
alter table submissions
	add column if not exists changes_idempotency_key text;
