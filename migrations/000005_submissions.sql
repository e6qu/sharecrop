create table if not exists submissions (
	id uuid primary key,
	task_id uuid not null references tasks(id),
	submitter_kind text not null,
	user_id uuid references users(id),
	wallet_address text,
	state text not null,
	response_json jsonb not null,
	accepted_idempotency_key text,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint submissions_submitter_kind_check check (submitter_kind in ('authenticated', 'anonymous')),
	constraint submissions_state_check check (state in ('submitted', 'invalid', 'accepted', 'rejected')),
	constraint submissions_submitter_check check (
		(submitter_kind = 'authenticated' and user_id is not null and wallet_address is null)
		or
		(submitter_kind = 'anonymous' and user_id is null and wallet_address is not null)
	)
);

create table if not exists submission_receipt_tokens (
	id uuid primary key,
	submission_id uuid not null references submissions(id),
	token_hash text not null unique,
	created_at timestamptz not null default now()
);

create table if not exists submission_validation_errors (
	submission_id uuid not null references submissions(id),
	error_index integer not null,
	path text not null,
	message text not null,
	created_at timestamptz not null default now(),
	primary key (submission_id, error_index),
	constraint submission_validation_errors_index_check check (error_index >= 0)
);

create table if not exists submission_sensitive_fields (
	submission_id uuid not null references submissions(id),
	field_index integer not null,
	path text not null,
	category text not null,
	retention text not null,
	redaction text not null,
	created_at timestamptz not null default now(),
	primary key (submission_id, field_index),
	constraint submission_sensitive_fields_index_check check (field_index >= 0),
	constraint submission_sensitive_fields_category_check check (category in ('pii', 'secret')),
	constraint submission_sensitive_fields_retention_check check (retention in ('standard', 'delete_on_request')),
	constraint submission_sensitive_fields_redaction_check check (redaction in ('replace', 'remove'))
);

create index if not exists submissions_task_idx
	on submissions(task_id, created_at);

create index if not exists submissions_user_idx
	on submissions(user_id)
	where submitter_kind = 'authenticated';

create index if not exists submission_receipt_tokens_hash_idx
	on submission_receipt_tokens(token_hash);
