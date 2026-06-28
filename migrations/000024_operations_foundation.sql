create table if not exists audit_events (
	id uuid primary key,
	actor_user_id uuid references users(id),
	action text not null,
	subject_kind text not null,
	subject_id text not null,
	metadata_json text not null default '{}',
	created_at timestamptz not null default now()
);

create index if not exists audit_events_actor_idx
	on audit_events(actor_user_id, created_at);

create index if not exists audit_events_subject_idx
	on audit_events(subject_kind, subject_id, created_at);

create table if not exists rate_limit_buckets (
	key text primary key,
	tokens double precision not null,
	updated_at timestamptz not null
);

create table if not exists mcp_http_sessions (
	id text primary key,
	subject_id text not null,
	closed_at timestamptz,
	last_seen_at timestamptz not null,
	created_at timestamptz not null default now()
);

create index if not exists mcp_http_sessions_subject_idx
	on mcp_http_sessions(subject_id, last_seen_at);
