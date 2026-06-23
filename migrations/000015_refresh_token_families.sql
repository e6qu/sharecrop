alter table refresh_tokens
	add column if not exists family_id uuid;

update refresh_tokens
	set family_id = id
	where family_id is null;

alter table refresh_tokens
	alter column family_id set not null;

create index if not exists refresh_tokens_family_active_idx
	on refresh_tokens(family_id)
	where status = 'active';
