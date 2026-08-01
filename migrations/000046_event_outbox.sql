-- In-transaction event outbox. Domain events are now inserted inside the same
-- transaction as the mutation they describe, in dispatch_state 'recorded'.
-- The dispatch step (inbox fan-out, webhook delivery expansion) runs after
-- commit — inline for latency, and again from the lifecycle runner's recovery
-- sweep after a crash — and moves the row to 'dispatched'. dispatched_at
-- records the dispatch instant as a fact; the lifecycle state itself lives in
-- the dispatch_state enum, never in a nullable timestamp.
alter table domain_events
	add column if not exists dispatch_state text not null default 'recorded',
	add column if not exists dispatched_at timestamptz;

alter table domain_events
	add constraint domain_events_dispatch_state_check check (dispatch_state in ('recorded', 'dispatched'));

-- Events recorded before this migration were fanned out by the old
-- post-commit recorder path; mark them dispatched so the recovery sweep never
-- replays history.
update domain_events set dispatch_state = 'dispatched';

-- The recovery sweep scans only stale recorded rows.
create index if not exists domain_events_recorded_idx
	on domain_events(occurred_at) where dispatch_state = 'recorded';

-- Inbox rows are keyed by the event that produced them, so the inline
-- dispatch and the recovery sweep can race without duplicating a
-- notification. Rows created before the outbox carry no event id.
alter table notifications
	add column if not exists event_id uuid;

create unique index if not exists notifications_event_recipient_idx
	on notifications(event_id, recipient_user_id) where event_id is not null;

-- Webhook delivery expansion moved into the per-event dispatch step; the
-- shared pump cursor is gone.
drop table if exists webhook_pump_cursor;
