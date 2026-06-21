create table if not exists users (
	id uuid primary key,
	email text not null unique,
	created_at timestamptz not null default now()
);

create table if not exists guest_subjects (
	id uuid primary key,
	created_at timestamptz not null default now()
);

create table if not exists password_credentials (
	user_id uuid primary key references users(id),
	password_hash text not null,
	created_at timestamptz not null default now()
);

create table if not exists refresh_tokens (
	id uuid primary key,
	token_hash text not null unique,
	subject_kind text not null,
	user_id uuid references users(id),
	guest_id uuid references guest_subjects(id),
	status text not null,
	issued_at timestamptz not null default now(),
	expires_at timestamptz not null,
	consumed_at timestamptz,
	constraint refresh_tokens_subject_kind_check check (subject_kind in ('user', 'guest')),
	constraint refresh_tokens_status_check check (status in ('active', 'consumed', 'revoked')),
	constraint refresh_tokens_subject_check check (
		(subject_kind = 'user' and user_id is not null and guest_id is null)
		or
		(subject_kind = 'guest' and guest_id is not null and user_id is null)
	)
);

create index if not exists refresh_tokens_active_hash_idx
	on refresh_tokens(token_hash)
	where status = 'active';

