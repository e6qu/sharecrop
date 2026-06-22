alter table tasks
	add column if not exists reward_kind text not null default 'none',
	add column if not exists reward_credit_amount integer;

alter table tasks
	add constraint tasks_reward_kind_check check (reward_kind in ('none', 'credit'));

alter table tasks
	add constraint tasks_reward_check check (
		(reward_kind = 'none' and reward_credit_amount is null)
		or
		(reward_kind = 'credit' and reward_credit_amount is not null and reward_credit_amount > 0)
	);

alter table submissions
	add column if not exists accepted_idempotency_key text;
