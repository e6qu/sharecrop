-- Make task series a first-class managed entity. A series previously existed
-- only as a grouping label created as a side effect of task creation. It now
-- carries its own description and lifecycle state, supports a comment thread,
-- and its member tasks are indexed for ordered retrieval and reordering.

alter table task_series
	add column if not exists description text not null default '';

alter table task_series
	add column if not exists state text not null default 'draft';

alter table task_series
	add column if not exists updated_at timestamptz not null default now();

alter table task_series
	drop constraint if exists task_series_state_check;

alter table task_series
	add constraint task_series_state_check check (state in ('draft', 'published', 'closed'));

-- Member tasks are listed and reordered by series; index the foreign key.
create index if not exists tasks_series_idx
	on tasks(series_id);

-- A comment thread on a series, ordered by creation time.
create table if not exists series_comments (
	id uuid primary key,
	series_id uuid not null references task_series(id),
	author_user_id uuid not null references users(id),
	body text not null,
	created_at timestamptz not null default now()
);

create index if not exists series_comments_series_idx
	on series_comments(series_id, created_at);
