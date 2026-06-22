alter table tasks
	add column if not exists participation_policy text not null default 'open',
	add column if not exists assignee_scope text not null default 'user',
	add column if not exists reservation_expires_after_hours integer not null default 48;

alter table tasks
	add constraint tasks_participation_policy_check check (participation_policy in ('open', 'reservation_required', 'approval_required'));

alter table tasks
	add constraint tasks_assignee_scope_check check (assignee_scope in ('user', 'organization_team'));

alter table tasks
	add constraint tasks_reservation_expires_after_hours_check check (
		reservation_expires_after_hours >= 1 and reservation_expires_after_hours <= 720
	);

create table if not exists task_reservations (
	id uuid primary key,
	task_id uuid not null references tasks(id),
	assignee_kind text not null,
	user_id uuid references users(id),
	team_id uuid references teams(id),
	organization_id uuid references organizations(id),
	state text not null,
	requested_by_user_id uuid not null references users(id),
	expires_at timestamptz not null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint task_reservations_assignee_kind_check check (assignee_kind in ('user', 'organization_team')),
	constraint task_reservations_state_check check (
		state in ('requested', 'active', 'declined', 'cancelled_by_requester', 'cancelled_by_worker', 'expired', 'submitted')
	),
	constraint task_reservations_assignee_check check (
		(assignee_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(assignee_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
	)
);

create unique index if not exists task_reservations_one_active_idx
	on task_reservations(task_id)
	where state = 'active';

create index if not exists task_reservations_task_idx
	on task_reservations(task_id, state, expires_at);

create table if not exists task_implementor_bans (
	task_id uuid not null references tasks(id),
	assignee_kind text not null,
	assignee_key text not null,
	user_id uuid references users(id),
	team_id uuid references teams(id),
	organization_id uuid references organizations(id),
	banned_by_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	primary key (task_id, assignee_kind, assignee_key),
	constraint task_implementor_bans_assignee_kind_check check (assignee_kind in ('user', 'organization_team')),
	constraint task_implementor_bans_assignee_check check (
		(assignee_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(assignee_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
	)
);
