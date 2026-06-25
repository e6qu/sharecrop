-- Pre-baked developer task types and a typed reference URL, plus a comment
-- thread on individual tasks (mirroring series_comments) so detailed tasks such
-- as code reviews support back-and-forth.

alter table tasks
	add column if not exists task_type text not null default 'general';

alter table tasks
	add column if not exists reference_url text not null default '';

alter table tasks
	drop constraint if exists tasks_task_type_check;

alter table tasks
	add constraint tasks_task_type_check check (
		task_type in ('general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing')
	);

create index if not exists tasks_task_type_idx
	on tasks(task_type);

create table if not exists task_comments (
	id uuid primary key,
	task_id uuid not null references tasks(id),
	author_user_id uuid not null references users(id),
	body text not null,
	created_at timestamptz not null default now()
);

create index if not exists task_comments_task_idx
	on task_comments(task_id, created_at);
