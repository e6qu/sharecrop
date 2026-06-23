-- Allow standalone (user-owned) teams alongside organization-owned teams.
-- Team ownership is a tagged union in the domain: a team is owned either by an
-- organization or directly by a user. Storage uses owner_kind to select the
-- variant and exactly one of organization_id / owner_user_id for the owner.

alter table teams
	add column if not exists owner_user_id uuid references users(id);

alter table teams
	drop constraint if exists teams_owner_kind_check;

alter table teams
	drop constraint if exists teams_organization_owner_check;

alter table teams
	add constraint teams_owner_kind_check check (owner_kind in ('organization', 'user'));

alter table teams
	add constraint teams_owner_reference_check check (
		(owner_kind = 'organization' and organization_id is not null and owner_user_id is null)
		or
		(owner_kind = 'user' and owner_user_id is not null and organization_id is null)
	);

create index if not exists teams_owner_user_idx
	on teams(owner_user_id);
