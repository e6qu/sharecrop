create table if not exists external_identities (
	issuer text not null,
	subject text not null,
	user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	primary key (issuer, subject)
);

create index if not exists external_identities_user_idx on external_identities(user_id);
