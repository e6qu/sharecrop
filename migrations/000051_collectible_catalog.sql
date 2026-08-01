-- DB-backed collectible catalog with entry lifecycle and edition caps, plus
-- instance-level provenance (catalog slug, edition number, issuer) and the
-- withdrawn instance state.
--
-- The catalog was previously a fixed Go list (internal/assets/catalog.go);
-- the 25 default entries are seeded here so both dialects share one source.
-- max_editions is the per-entry mint cap: NULL means uncapped (badges),
-- editions are capped at 100, uniques at 1.
create table if not exists collectible_catalog_entries (
	slug text primary key,
	name text not null,
	kind text not null,
	transfer_policy text not null,
	art text not null,
	state text not null default 'available',
	max_editions bigint,
	created_at timestamptz not null default now(),
	state_recorded_at timestamptz not null default now(),
	constraint collectible_catalog_entries_kind_check check (kind in ('unique', 'edition', 'badge')),
	constraint collectible_catalog_entries_state_check check (state in ('available', 'withdrawn')),
	constraint collectible_catalog_entries_policy_check check (
		transfer_policy in ('non_transferable_except_payout', 'transferable_between_users', 'transferable_within_organization', 'issuer_controlled')
	)
);

insert into collectible_catalog_entries (slug, name, kind, transfer_policy, art, max_editions) values
	('harvest-star', 'Harvest Star', 'badge', 'transferable_between_users', 'harvest-star', null),
	('golden-sickle', 'Golden Sickle', 'badge', 'transferable_between_users', 'golden-sickle', null),
	('seedling', 'Seedling', 'badge', 'transferable_between_users', 'seedling', null),
	('sun-token', 'Sun Token', 'badge', 'transferable_between_users', 'sun-token', null),
	('rain-drop', 'Rain Drop', 'badge', 'transferable_between_users', 'rain-drop', null),
	('wheat-sheaf', 'Wheat Sheaf', 'badge', 'transferable_between_users', 'wheat-sheaf', null),
	('red-barn', 'Red Barn', 'badge', 'transferable_between_users', 'red-barn', null),
	('scarecrow', 'Scarecrow', 'badge', 'transferable_between_users', 'scarecrow', null),
	('honey-pot', 'Honey Pot', 'badge', 'transferable_between_users', 'honey-pot', null),
	('pumpkin', 'Pumpkin', 'badge', 'transferable_between_users', 'pumpkin', null),
	('apple', 'Apple', 'badge', 'transferable_between_users', 'apple', null),
	('carrot', 'Carrot', 'badge', 'transferable_between_users', 'carrot', null),
	('beehive', 'Beehive', 'badge', 'transferable_between_users', 'beehive', null),
	('windmill', 'Windmill', 'badge', 'transferable_between_users', 'windmill', null),
	('tractor', 'Tractor', 'badge', 'transferable_between_users', 'tractor', null),
	('silver-plow', 'Silver Plow', 'edition', 'transferable_between_users', 'silver-plow', 100),
	('golden-egg', 'Golden Egg', 'edition', 'transferable_between_users', 'golden-egg', 100),
	('prize-cow', 'Prize Cow', 'edition', 'transferable_between_users', 'prize-cow', 100),
	('lucky-clover', 'Lucky Clover', 'edition', 'transferable_between_users', 'lucky-clover', 100),
	('full-moon-harvest', 'Full-Moon Harvest', 'edition', 'transferable_between_users', 'full-moon-harvest', 100),
	('cornucopia', 'Cornucopia', 'unique', 'transferable_between_users', 'cornucopia', 1),
	('first-harvest-trophy', 'First-Harvest Trophy', 'unique', 'transferable_between_users', 'first-harvest-trophy', 1),
	('founders-seed', 'Founder''s Seed', 'unique', 'transferable_between_users', 'founders-seed', 1),
	('rainbow-field', 'Rainbow Field', 'unique', 'transferable_between_users', 'rainbow-field', 1),
	('golden-combine', 'Golden Combine', 'unique', 'transferable_between_users', 'golden-combine', 1)
on conflict do nothing;

-- Instance provenance. catalog_slug ties an awarded copy to its catalog
-- entry (NULL for custom mints); edition_number is the mint sequence for
-- edition-kind instances; issued_by_user_id is the acting user who minted or
-- awarded the instance (NULL for pre-migration rows). Existing instances are
-- deliberately NOT back-linked to catalog entries by name: retroactively
-- linking could violate the new uniqueness guarantees for historical copies,
-- so the guarantees apply from this migration forward.
alter table collectibles
	add column if not exists catalog_slug text,
	add column if not exists edition_number bigint,
	add column if not exists issued_by_user_id uuid;

-- Instance lifecycle gains the withdrawn state (admin removed the instance
-- from its holder; not transferable, tippable, awardable, or escrowable).
-- SQLite skips the constraint pair (unsupported ALTER); the domain layer
-- validates the enum there instead.
alter table collectibles drop constraint if exists collectibles_state_check;

alter table collectibles add constraint collectibles_state_check check (
	state in ('minted', 'escrowed', 'awarded', 'withdrawn')
);

-- True uniqueness, enforced by the engine rather than app checks alone:
-- at most one live instance per unique catalog entry; at most one live
-- instance per (issuer, name) for custom unique mints; edition numbers are
-- unique per catalog entry and per custom (issuer, name) series.
create unique index if not exists collectibles_unique_catalog_live_uidx
	on collectibles(catalog_slug)
	where kind = 'unique' and catalog_slug is not null and state <> 'withdrawn';

create unique index if not exists collectibles_unique_custom_live_uidx
	on collectibles(issued_by_user_id, name)
	where kind = 'unique' and catalog_slug is null and issued_by_user_id is not null and state <> 'withdrawn';

create unique index if not exists collectibles_catalog_edition_uidx
	on collectibles(catalog_slug, edition_number)
	where catalog_slug is not null and edition_number is not null;

create unique index if not exists collectibles_custom_edition_uidx
	on collectibles(issued_by_user_id, name, edition_number)
	where catalog_slug is null and issued_by_user_id is not null and edition_number is not null;
