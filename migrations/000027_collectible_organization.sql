alter table collectibles
	add column if not exists organization_id uuid references organizations(id) on delete restrict;

update collectibles
set organization_id = owner_user_id
where owner_kind = 'organization'
	and organization_id is null;

create index if not exists collectibles_organization_idx
	on collectibles(organization_id);
