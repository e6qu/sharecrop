create table if not exists app_metadata (
	name text primary key,
	value text not null,
	recorded_at timestamptz not null default now()
);
