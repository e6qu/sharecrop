create table if not exists agent_credentials (
	id uuid primary key,
	user_id uuid not null references users(id),
	label text not null,
	token_hash text not null unique,
	state text not null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint agent_credentials_state_check check (state in ('active', 'revoked'))
);

create table if not exists agent_credential_scopes (
	credential_id uuid not null references agent_credentials(id),
	scope text not null,
	created_at timestamptz not null default now(),
	primary key (credential_id, scope),
	constraint agent_credential_scopes_scope_check check (
		scope in ('tasks_read', 'tasks_write', 'submissions_write', 'submissions_read', 'submissions_review')
	)
);

create index if not exists agent_credentials_user_idx
	on agent_credentials(user_id);

create index if not exists agent_credentials_active_hash_idx
	on agent_credentials(token_hash)
	where state = 'active';
