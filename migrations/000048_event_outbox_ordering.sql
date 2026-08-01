-- Commit-ordered event sequencing and dispatch attempt accounting.
--
-- Ordering: cursor feed reads page with "seq > after", so events must become
-- visible in seq order. Without coordination, commit order and seq order can
-- diverge (a transaction holding a lower seq can commit after a later-seq
-- event was already served), and a cursor that advanced past the gap skips
-- the late event forever. Every transaction that inserts domain_events rows
-- now locks the single domain_event_fence row FOR UPDATE immediately before
-- its first event insert, serializing {seq allocation -> commit} so
-- visibility order equals seq order. SQLite needs no fence (single writer,
-- writes serialize); the translated SELECT simply drops FOR UPDATE there.
create table if not exists domain_event_fence (
	id integer primary key,
	constraint domain_event_fence_single check (id = 1)
);

insert into domain_event_fence (id) values (1) on conflict do nothing;

-- Attempt accounting: the dispatch sweep counts its attempts per recorded
-- row; rows that keep failing move to the terminal dispatch_failed state
-- instead of retrying forever, and stay inspectable through the store.
alter table domain_events
	add column if not exists dispatch_attempts bigint not null default 0;

alter table domain_events drop constraint if exists domain_events_dispatch_state_check;

alter table domain_events
	add constraint domain_events_dispatch_state_check check (dispatch_state in ('recorded', 'dispatched', 'dispatch_failed'));
