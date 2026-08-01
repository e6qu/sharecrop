-- Users get a required display name, shown wherever the product names an
-- actor or counterparty. Existing accounts are backfilled from the email
-- local part (the same default derivation registration applies when no name
-- is provided). The empty-string default exists only so the column can be
-- added NOT NULL; the application always writes a real name.
alter table users
	add column if not exists display_name text not null default '';

update users
	set display_name = split_part(email, '@', 1)
	where display_name = '';
