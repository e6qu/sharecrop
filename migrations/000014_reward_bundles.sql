alter table tasks
	drop constraint if exists tasks_reward_kind_check;

alter table tasks
	add constraint tasks_reward_kind_check check (reward_kind in ('none', 'credit', 'collectible', 'bundle'));

alter table tasks
	drop constraint if exists tasks_reward_check;

alter table tasks
	add constraint tasks_reward_check check (
		(reward_kind in ('none', 'collectible') and reward_credit_amount is null)
		or
		(reward_kind in ('credit', 'bundle') and reward_credit_amount is not null and reward_credit_amount > 0)
	);
