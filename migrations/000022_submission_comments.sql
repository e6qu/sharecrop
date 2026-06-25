-- A private comment thread on an individual submission, mirroring task_comments,
-- so the submission's author and the owner of the submission's task can exchange
-- clarifying messages while a submission is under review.

create table if not exists submission_comments (
	id uuid primary key,
	submission_id uuid not null references submissions(id),
	author_user_id uuid not null references users(id),
	body text not null,
	created_at timestamptz not null default now()
);

create index if not exists submission_comments_submission_idx
	on submission_comments(submission_id, created_at);
