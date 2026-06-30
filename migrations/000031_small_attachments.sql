create table if not exists task_attachments (
	task_id uuid not null references tasks(id),
	attachment_index integer not null,
	filename text not null,
	content_type text not null,
	content bytea not null,
	created_at timestamptz not null default now(),
	primary key (task_id, attachment_index),
	constraint task_attachments_index_check check (attachment_index >= 0),
	constraint task_attachments_size_check check (octet_length(content) > 0 and octet_length(content) <= 512000),
	constraint task_attachments_content_type_check check (content_type in ('image/png', 'image/jpeg', 'image/gif', 'image/webp', 'text/plain', 'application/json', 'application/pdf'))
);

create table if not exists submission_attachments (
	submission_id uuid not null references submissions(id),
	attachment_index integer not null,
	filename text not null,
	content_type text not null,
	content bytea not null,
	created_at timestamptz not null default now(),
	primary key (submission_id, attachment_index),
	constraint submission_attachments_index_check check (attachment_index >= 0),
	constraint submission_attachments_size_check check (octet_length(content) > 0 and octet_length(content) <= 512000),
	constraint submission_attachments_content_type_check check (content_type in ('image/png', 'image/jpeg', 'image/gif', 'image/webp', 'text/plain', 'application/json', 'application/pdf'))
);
