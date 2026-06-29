create table if not exists notifications (
	id uuid primary key,
	recipient_user_id uuid not null references users(id),
	actor_user_id uuid not null references users(id),
	kind text not null,
	subject_kind text not null,
	subject_id text not null,
	state text not null,
	metadata_json jsonb not null default '{}'::jsonb,
	created_at timestamptz not null default now()
);

create index if not exists notifications_recipient_state_idx
	on notifications(recipient_user_id, state, created_at);

create index if not exists notifications_subject_idx
	on notifications(subject_kind, subject_id, created_at);
