create table if not exists collectibles (
	id uuid primary key,
	name text not null,
	kind text not null,
	state text not null,
	transfer_policy text not null,
	owner_user_id uuid not null references users(id),
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint collectibles_kind_check check (kind in ('unique', 'edition', 'badge')),
	constraint collectibles_state_check check (state in ('minted', 'escrowed', 'awarded')),
	constraint collectibles_transfer_policy_check check (
		transfer_policy in ('non_transferable_except_payout', 'transferable_between_users', 'transferable_within_organization', 'issuer_controlled')
	)
);

create table if not exists task_collectible_rewards (
	task_id uuid primary key references tasks(id),
	collectible_id uuid not null references collectibles(id),
	funder_user_id uuid not null references users(id),
	state text not null,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint task_collectible_rewards_state_check check (state in ('held', 'released', 'refunded'))
);

create index if not exists collectibles_owner_idx
	on collectibles(owner_user_id);
