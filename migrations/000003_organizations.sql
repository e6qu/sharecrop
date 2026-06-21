create table if not exists organizations (
	id uuid primary key,
	name text not null,
	created_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now()
);

create table if not exists organization_memberships (
	id uuid primary key,
	organization_id uuid not null references organizations(id),
	user_id uuid not null references users(id),
	status text not null,
	created_at timestamptz not null default now(),
	status_recorded_at timestamptz not null default now(),
	constraint organization_memberships_status_check check (status in ('active', 'deactivated', 'removed')),
	constraint organization_memberships_unique_user unique (organization_id, user_id)
);

create table if not exists organization_membership_roles (
	membership_id uuid not null references organization_memberships(id),
	role text not null,
	created_at timestamptz not null default now(),
	primary key (membership_id, role),
	constraint organization_membership_roles_role_check check (
		role in ('owner', 'admin', 'member', 'billing', 'reviewer', 'public_publisher')
	)
);

create table if not exists teams (
	id uuid primary key,
	name text not null,
	owner_kind text not null,
	organization_id uuid references organizations(id),
	created_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	constraint teams_owner_kind_check check (owner_kind in ('organization')),
	constraint teams_organization_owner_check check (owner_kind = 'organization' and organization_id is not null)
);

create table if not exists team_members (
	team_id uuid not null references teams(id),
	user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	primary key (team_id, user_id)
);

create index if not exists organization_memberships_user_idx
	on organization_memberships(user_id, status);

create index if not exists teams_organization_idx
	on teams(organization_id);

