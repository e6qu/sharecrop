alter table users
	add column if not exists display_name text not null default '',
	add column if not exists status text not null default 'active',
	add column if not exists email_verified_at timestamptz;

alter table users
	add constraint users_status_check check (status in ('active', 'deactivated'));

create table if not exists account_tokens (
	id uuid primary key,
	user_id uuid not null references users(id),
	token_hash text not null unique,
	kind text not null,
	status text not null,
	issued_at timestamptz not null default now(),
	expires_at timestamptz not null,
	consumed_at timestamptz,
	constraint account_tokens_kind_check check (kind in ('email_verification', 'password_reset')),
	constraint account_tokens_status_check check (status in ('active', 'consumed', 'revoked'))
);

create index if not exists account_tokens_active_hash_idx
	on account_tokens(token_hash)
	where status = 'active';

create index if not exists account_tokens_user_kind_active_idx
	on account_tokens(user_id, kind)
	where status = 'active';
