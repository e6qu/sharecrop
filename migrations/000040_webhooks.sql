-- Outbound webhook subscriptions and deliveries. Subscriptions are owned by a
-- user or an organization and filter on domain event kinds (join table, same
-- shape as agent_credential_scopes). The secret is stored as written because
-- the dispatcher must compute an HMAC over each delivery body.
create table if not exists webhook_subscriptions (
	id uuid primary key,
	owner_kind text not null,
	owner_user_id uuid references users(id),
	owner_organization_id uuid references organizations(id),
	url text not null,
	secret text not null,
	state text not null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint webhook_subscriptions_owner_kind_check check (owner_kind in ('user', 'organization')),
	constraint webhook_subscriptions_owner_check check (
		(owner_kind = 'user' and owner_user_id is not null and owner_organization_id is null)
		or (owner_kind = 'organization' and owner_organization_id is not null and owner_user_id is null)
	),
	constraint webhook_subscriptions_state_check check (state in ('active', 'revoked'))
);

create table if not exists webhook_subscription_kinds (
	subscription_id uuid not null references webhook_subscriptions(id),
	kind text not null,
	primary key (subscription_id, kind)
);

-- One delivery row per (subscription, event). The host-side pump inserts
-- pending rows as events arrive, claims due rows with FOR UPDATE SKIP LOCKED,
-- and walks the bounded retry schedule to delivered or dead.
create table if not exists webhook_deliveries (
	id uuid primary key,
	subscription_id uuid not null references webhook_subscriptions(id),
	event_seq bigint not null references domain_events(seq),
	state text not null,
	attempt_count bigint not null default 0,
	next_attempt_at timestamptz not null,
	last_status text not null default '',
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint webhook_deliveries_state_check check (state in ('pending', 'delivered', 'dead')),
	constraint webhook_deliveries_subscription_event_unique unique (subscription_id, event_seq)
);

create index if not exists webhook_deliveries_due_idx
	on webhook_deliveries(state, next_attempt_at);

-- Single shared fan-out cursor: the pump advances last_seq under row lock so
-- each domain event is expanded into deliveries exactly once across replicas.
create table if not exists webhook_pump_cursor (
	singleton bigint primary key,
	last_seq bigint not null,
	constraint webhook_pump_cursor_singleton_check check (singleton = 1)
);

insert into webhook_pump_cursor (singleton, last_seq)
	values (1, 0)
	on conflict (singleton) do nothing;
