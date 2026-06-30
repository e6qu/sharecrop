create table if not exists moderation_report_triage (
	report_audit_event_id uuid primary key references audit_events(id),
	state text not null default 'open',
	resolution_note text not null default '',
	updated_by_user_id uuid references users(id),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint moderation_report_triage_state_check check (state in ('open', 'resolved', 'dismissed'))
);

insert into moderation_report_triage (report_audit_event_id, state, resolution_note, created_at, updated_at)
select id, 'open', '', created_at, created_at
from audit_events
where action = 'moderation_report_created'
on conflict (report_audit_event_id) do nothing;

create index if not exists moderation_report_triage_state_idx
	on moderation_report_triage(state, updated_at);

create table if not exists platform_admins (
	user_id uuid primary key references users(id),
	source text not null,
	state text not null default 'active',
	granted_by_user_id uuid references users(id),
	created_at timestamptz not null default now(),
	updated_at timestamptz not null default now(),
	constraint platform_admins_source_check check (source in ('granted')),
	constraint platform_admins_state_check check (state in ('active', 'revoked'))
);

create index if not exists platform_admins_state_idx
	on platform_admins(state, updated_at);

create table if not exists privacy_retention_runs (
	id uuid primary key,
	actor_user_id uuid not null references users(id),
	redacted_field_count integer not null,
	created_at timestamptz not null default now(),
	constraint privacy_retention_runs_redacted_count_check check (redacted_field_count >= 0)
);
