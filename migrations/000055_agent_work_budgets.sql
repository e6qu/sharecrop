-- Agent work budgets. A personal agent credential carries a work policy:
-- work-seeking is disabled by default for every credential (existing ones
-- included - a deliberate breaking change for live agents), and when a human
-- enables it they must state a daily task budget; the remaining allowances
-- are optional (null = no limit configured). The advisory token budget is
-- stored for the agent to read and is never enforced by the server.
--
-- SQLite skips the ADD CONSTRAINT statements (unsupported ALTER) and strips
-- inline CHECKs; the domain work-policy types validate there instead.
alter table agent_credentials
	add column if not exists work_seeking_state text not null default 'work_seeking_disabled',
	add column if not exists work_max_tasks_per_day bigint,
	add column if not exists work_max_concurrent_reservations bigint,
	add column if not exists work_max_credits_per_day bigint,
	add column if not exists work_min_reward_credits bigint,
	add column if not exists work_token_budget_tokens bigint,
	add column if not exists work_token_budget_note text;

alter table agent_credentials add constraint agent_credentials_work_seeking_state_check check (
	work_seeking_state in ('work_seeking_disabled', 'work_seeking_enabled')
);

-- An enabled policy always states its daily task budget; a disabled policy
-- carries no allowances at all.
alter table agent_credentials add constraint agent_credentials_work_allowances_check check (
	(work_seeking_state = 'work_seeking_enabled' and work_max_tasks_per_day is not null)
	or (
		work_seeking_state = 'work_seeking_disabled'
		and work_max_tasks_per_day is null
		and work_max_concurrent_reservations is null
		and work_max_credits_per_day is null
		and work_min_reward_credits is null
		and work_token_budget_tokens is null
		and work_token_budget_note is null
	)
);

-- The optional allowed-task-type restriction of an enabled policy, one row
-- per allowed type (no rows = every type allowed), mirroring the
-- agent_credential_scopes shape.
create table if not exists agent_credential_work_task_types (
	credential_id uuid not null references agent_credentials(id),
	task_type text not null,
	created_at timestamptz not null default now(),
	primary key (credential_id, task_type),
	constraint agent_credential_work_task_types_type_check check (
		task_type in (
			'general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing',
			'document_review', 'documentation_writing', 'diagram_writing', 'planning', 'research', 'data_extraction',
			'troubleshooting', 'code_analysis', 'architecture_review', 'threat_analysis'
		)
	)
);

-- Reservations record which agent credential established them (null = the
-- owning user acted directly through a session), so the concurrent
-- reservation cap can count exactly the reservations this credential holds.
alter table task_reservations
	add column if not exists reserved_via_credential_id uuid references agent_credentials(id);

create index if not exists task_reservations_via_credential_active_idx
	on task_reservations(reserved_via_credential_id)
	where state = 'active';

-- UTC-calendar-day budget counters (daily task budget, daily credit spend,
-- peer-transfer velocity, budget-refusal day totals), modeled on
-- rate_limit_buckets but windowed by day instead of refilled continuously.
-- day is the UTC date in YYYY-MM-DD form.
create table if not exists work_day_counters (
	key text not null,
	day text not null,
	used bigint not null,
	primary key (key, day)
);
