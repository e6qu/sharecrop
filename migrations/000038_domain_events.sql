-- Append-only domain event stream. Events are emitted by the domain services
-- (both REST and MCP paths), and feed notifications, webhook deliveries, and
-- the browser live-update feed. seq is the global cursor for consumers.
create table if not exists domain_events (
	seq bigserial primary key,
	id uuid not null unique,
	kind text not null,
	actor_kind text not null,
	actor_user_id uuid references users(id),
	task_id uuid,
	submission_id uuid,
	reservation_id uuid,
	series_id uuid,
	organization_id uuid,
	collectible_id uuid,
	metadata_json jsonb not null default '{}',
	occurred_at timestamptz not null default now(),
	constraint domain_events_actor_kind_check check (actor_kind in ('user', 'system')),
	constraint domain_events_actor_user_check check (
		actor_kind <> 'user' or actor_user_id is not null
	)
);

create index if not exists domain_events_task_idx
	on domain_events(task_id, seq);

create index if not exists domain_events_org_idx
	on domain_events(organization_id, seq);

-- Per-user visibility of an event, resolved at emission time by the emitting
-- service (owner, assignee, submitter, recipient...). The live feed reads
-- (user_id, event_seq) in cursor order.
create table if not exists domain_event_recipients (
	event_seq bigint not null references domain_events(seq),
	user_id uuid not null references users(id),
	primary key (user_id, event_seq)
);
