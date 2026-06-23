-- Allow a task to escrow more than one collectible reward.
-- The original table used task_id as the primary key, which enforced at most
-- one collectible reward per task. Replace that with a surrogate primary key
-- and a (task_id, collectible_id) uniqueness constraint so a task can hold
-- several distinct collectibles while still preventing the same collectible
-- from being escrowed twice on the same task.

alter table task_collectible_rewards
	add column if not exists id uuid;

update task_collectible_rewards
	set id = task_id
	where id is null;

alter table task_collectible_rewards
	alter column id set not null;

alter table task_collectible_rewards
	drop constraint if exists task_collectible_rewards_pkey;

alter table task_collectible_rewards
	add primary key (id);

alter table task_collectible_rewards
	add constraint task_collectible_rewards_unique_collectible unique (task_id, collectible_id);

create index if not exists task_collectible_rewards_task_idx
	on task_collectible_rewards(task_id);
