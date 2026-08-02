-- Email verification becomes an explicit lifecycle state instead of a hidden
-- nullable-timestamp flag, and the 100-credit signup grant moves from
-- registration time to first-verification time (a sybil gate: an unverified
-- account keeps a zero balance). email_verified_at stays as the event-time
-- fact. Existing accounts that already verified are backfilled as verified;
-- everyone else starts unverified (they keep any grant they already
-- received - the grant's per-account idempotency key prevents a second one on
-- later verification). SQLite skips the ADD CONSTRAINT statement (unsupported
-- ALTER); the domain enum validates there instead.
alter table users
	add column if not exists email_verification_state text not null default 'unverified';

update users set email_verification_state = 'verified' where email_verified_at is not null;

alter table users add constraint users_email_verification_state_check check (
	email_verification_state in ('unverified', 'verified')
);
