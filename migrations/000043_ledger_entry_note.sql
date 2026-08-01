-- Platform-admin manual credit grants record a required human explanation.
-- The note lives on the ledger entry itself so the reason travels with the
-- money movement. Existing entries (and non-grant kinds) carry an empty note.
alter table ledger_entries
	add column if not exists note text not null default '';
