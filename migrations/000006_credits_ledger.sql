create table if not exists credit_accounts (
	id uuid primary key,
	owner_kind text not null,
	user_id uuid references users(id),
	created_at timestamptz not null default now(),
	constraint credit_accounts_owner_kind_check check (owner_kind in ('user')),
	constraint credit_accounts_owner_check check (
		owner_kind = 'user' and user_id is not null
	),
	constraint credit_accounts_user_unique unique (user_id)
);

create table if not exists ledger_entries (
	id uuid primary key,
	account_id uuid not null references credit_accounts(id),
	kind text not null,
	amount bigint not null,
	task_id uuid references tasks(id),
	idempotency_key text,
	created_at timestamptz not null default now(),
	constraint ledger_entries_kind_check check (
		kind in ('signup_grant', 'task_escrow', 'task_refund', 'task_payout', 'manual_adjustment')
	),
	constraint ledger_entries_amount_nonzero_check check (amount <> 0),
	constraint ledger_entries_idempotency_key_unique unique (idempotency_key)
);

create table if not exists task_escrows (
	task_id uuid primary key references tasks(id),
	funder_account_id uuid not null references credit_accounts(id),
	amount bigint not null,
	state text not null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint task_escrows_amount_check check (amount > 0),
	constraint task_escrows_state_check check (state in ('held', 'released', 'refunded'))
);

create index if not exists ledger_entries_account_idx
	on ledger_entries(account_id, created_at);

create index if not exists ledger_entries_task_idx
	on ledger_entries(task_id)
	where task_id is not null;

create unique index if not exists submissions_single_accepted_idx
	on submissions(task_id)
	where state = 'accepted';
