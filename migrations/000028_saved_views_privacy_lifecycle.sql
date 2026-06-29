create table if not exists saved_queue_views (
	id uuid primary key,
	user_id uuid not null references users(id),
	scope text not null,
	name text not null,
	query_text text not null default '',
	state_filter text not null default '',
	type_filter text not null default '',
	sort_order text not null default 'newest',
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint saved_queue_views_scope_check check (scope in ('team_work', 'organization_tasks')),
	constraint saved_queue_views_name_check check (length(trim(name)) > 0)
);

create unique index if not exists saved_queue_views_owner_scope_name_idx
	on saved_queue_views(user_id, scope, name);

create table if not exists privacy_requests (
	id uuid primary key,
	requested_by_user_id uuid not null references users(id),
	kind text not null,
	state text not null,
	export_json text not null default '',
	resolution_note text not null default '',
	created_at timestamptz not null default now(),
	resolved_at timestamptz,
	constraint privacy_requests_kind_check check (kind in ('data_export', 'sensitive_field_deletion')),
	constraint privacy_requests_state_check check (state in ('queued', 'resolved')),
	constraint privacy_requests_resolved_check check (
		(state = 'queued' and resolved_at is null)
		or
		(state = 'resolved' and resolved_at is not null)
	)
);

create index if not exists privacy_requests_requester_state_idx
	on privacy_requests(requested_by_user_id, state, created_at);

alter table submission_sensitive_fields
	add column if not exists state text not null default 'active',
	add column if not exists redacted_at timestamptz;

alter table submission_sensitive_fields
	drop constraint if exists submission_sensitive_fields_state_check;

alter table submission_sensitive_fields
	add constraint submission_sensitive_fields_state_check check (state in ('active', 'redacted'));

create table if not exists submission_sensitive_field_events (
	id uuid primary key,
	submission_id uuid not null references submissions(id),
	actor_user_id uuid not null references users(id),
	action text not null,
	field_path text not null,
	created_at timestamptz not null default now(),
	constraint submission_sensitive_field_events_action_check check (action in ('sensitive_field_accessed', 'sensitive_field_redacted'))
);

create index if not exists submission_sensitive_field_events_submission_idx
	on submission_sensitive_field_events(submission_id, created_at);

alter table tasks
	drop constraint if exists tasks_assignee_scope_check;

alter table tasks
	add constraint tasks_assignee_scope_check check (assignee_scope in ('user', 'organization_team', 'team'));

alter table task_reservations
	drop constraint if exists task_reservations_assignee_kind_check,
	drop constraint if exists task_reservations_assignee_check;

alter table task_reservations
	add constraint task_reservations_assignee_kind_check check (assignee_kind in ('user', 'organization_team', 'team'));

alter table task_reservations
	add constraint task_reservations_assignee_check check (
		(assignee_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(assignee_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
		or
		(assignee_kind = 'team' and user_id is null and team_id is not null and organization_id is null)
	);

alter table task_implementor_bans
	drop constraint if exists task_implementor_bans_assignee_kind_check,
	drop constraint if exists task_implementor_bans_assignee_check;

alter table task_implementor_bans
	add constraint task_implementor_bans_assignee_kind_check check (assignee_kind in ('user', 'organization_team', 'team'));

alter table task_implementor_bans
	add constraint task_implementor_bans_assignee_check check (
		(assignee_kind = 'user' and user_id is not null and team_id is null and organization_id is null)
		or
		(assignee_kind = 'organization_team' and user_id is null and team_id is not null and organization_id is not null)
		or
		(assignee_kind = 'team' and user_id is null and team_id is not null and organization_id is null)
	);
