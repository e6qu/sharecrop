-- Task expiration instant. Null means the task has no expiration policy; the
-- domain models this as the ExpirationPolicy sum (NoExpiration | ExpiresAt),
-- so the nullable column is a boundary shape only. The lifecycle runner
-- transitions open tasks past this instant to the 'expired' state (already in
-- the tasks state check constraint) and refunds their escrow.
alter table tasks
	add column if not exists expires_at timestamptz;

create index if not exists tasks_open_expiry_idx
	on tasks(expires_at)
	where state = 'open' and expires_at is not null;
