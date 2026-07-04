create table if not exists org_credentials (
	id uuid primary key,
	organization_id uuid not null references organizations(id),
	label text not null,
	token_hash text not null unique,
	state text not null,
	expires_at timestamptz null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint org_credentials_state_check check (state in ('active', 'revoked'))
);

create table if not exists org_credential_scopes (
	credential_id uuid not null references org_credentials(id),
	scope text not null,
	created_at timestamptz not null default now(),
	primary key (credential_id, scope),
	constraint org_credential_scopes_scope_check check (
		scope in (
			'tasks_read', 'tasks_write', 'submissions_write', 'submissions_read', 'submissions_review',
			'org_read', 'org_manage',
			'collectibles_read', 'collectibles_manage',
			'notifications_read', 'notifications_manage',
			'users_read',
			'ledger_read',
			'moderation_read', 'moderation_manage',
			'privacy_read', 'privacy_manage',
			'platform_admin',
			'credentials_manage'
		)
	)
);

create index if not exists org_credentials_organization_idx
	on org_credentials(organization_id);

create index if not exists org_credentials_active_hash_idx
	on org_credentials(token_hash)
	where state = 'active';
