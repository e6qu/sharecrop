-- Collectibles can be owned by a user, a team, or an organization (so an admin
-- can award a default collectible to any of them). The owner_user_id column now
-- holds the owning entity's id of any kind, disambiguated by owner_kind, so the
-- foreign key to users is dropped.

alter table collectibles
	add column if not exists owner_kind text not null default 'user';

alter table collectibles
	drop constraint if exists collectibles_owner_user_id_fkey;

alter table collectibles
	add constraint collectibles_owner_kind_check check (owner_kind in ('user', 'team', 'organization'));
