create table if not exists task_series (
	id uuid primary key,
	owner_kind text not null,
	user_id uuid references users(id),
	team_id uuid references teams(id),
	organization_id uuid references organizations(id),
	title text not null,
	created_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	constraint task_series_owner_kind_check check (owner_kind in ('user', 'team', 'organization', 'organization_team')),
	constraint task_series_owner_check check (
		(owner_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(owner_kind = 'team' and user_id is null and team_id is not null and organization_id is null)
		or
		(owner_kind = 'organization' and user_id is null and team_id is null and organization_id is not null)
		or
		(owner_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
	)
);

create table if not exists tasks (
	id uuid primary key,
	series_id uuid references task_series(id),
	series_position integer,
	owner_kind text not null,
	user_id uuid references users(id),
	team_id uuid references teams(id),
	organization_id uuid references organizations(id),
	title text not null,
	description text not null,
	reward_kind text not null default 'none',
	reward_credit_amount integer,
	state text not null,
	response_schema_json jsonb not null,
	data_payload_kind text not null,
	data_payload_json jsonb,
	created_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint tasks_state_check check (state in ('draft', 'open', 'closed', 'cancelled', 'expired')),
	constraint tasks_data_payload_kind_check check (data_payload_kind in ('none', 'json')),
	constraint tasks_payload_check check (
		(data_payload_kind = 'none' and data_payload_json is null)
		or
		(data_payload_kind = 'json' and data_payload_json is not null)
	),
	constraint tasks_owner_kind_check check (owner_kind in ('user', 'team', 'organization', 'organization_team')),
	constraint tasks_owner_check check (
		(owner_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(owner_kind = 'team' and user_id is null and team_id is not null and organization_id is null)
		or
		(owner_kind = 'organization' and user_id is null and team_id is null and organization_id is not null)
		or
		(owner_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
	),
	constraint tasks_series_position_check check (
		(series_id is null and series_position is null)
		or
		(series_id is not null and series_position is not null and series_position > 0)
	)
);

create table if not exists task_visibility_scopes (
	task_id uuid not null references tasks(id),
	visibility_kind text not null,
	scope_key text not null,
	user_id uuid references users(id),
	team_id uuid references teams(id),
	organization_id uuid references organizations(id),
	created_at timestamptz not null default now(),
	primary key (task_id, visibility_kind, scope_key),
	constraint task_visibility_kind_check check (visibility_kind in ('public', 'user', 'team', 'organization', 'organization_team')),
	constraint task_visibility_scope_check check (
		(visibility_kind = 'public' and user_id is null and team_id is null and organization_id is null)
		or
		(visibility_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(visibility_kind = 'team' and user_id is null and team_id is not null and organization_id is null)
		or
		(visibility_kind = 'organization' and user_id is null and team_id is null and organization_id is not null)
		or
		(visibility_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
	)
);

create table if not exists task_capability_tokens (
	id uuid primary key,
	task_id uuid not null references tasks(id),
	token_hash text not null unique,
	state text not null,
	created_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint task_capability_tokens_state_check check (state in ('active', 'revoked'))
);

create index if not exists tasks_owner_idx
	on tasks(owner_kind, user_id, team_id, organization_id);

create index if not exists task_visibility_public_idx
	on task_visibility_scopes(visibility_kind)
	where visibility_kind = 'public';

create index if not exists task_visibility_user_idx
	on task_visibility_scopes(user_id)
	where visibility_kind = 'user';

create index if not exists task_visibility_organization_idx
	on task_visibility_scopes(organization_id)
	where visibility_kind = 'organization';

create index if not exists task_capability_tokens_active_hash_idx
	on task_capability_tokens(token_hash)
	where state = 'active';
