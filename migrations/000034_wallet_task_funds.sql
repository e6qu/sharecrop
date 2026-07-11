-- Replace the escrow state machines (task_escrows, task_collectible_rewards)
-- with stateless "task_funds" temp stores. A row exists iff the task currently
-- holds allocated credits or a held collectible reward; the row is deleted on
-- award or refund. The ledger entry kind task_escrow is renamed to task_fund.

-- Rename the ledger entry kind task_escrow -> task_fund.
alter table ledger_entries
	drop constraint if exists ledger_entries_kind_check;

update ledger_entries set kind = 'task_fund' where kind = 'task_escrow';

alter table ledger_entries
	add constraint ledger_entries_kind_check check (
		kind in ('signup_grant', 'task_fund', 'task_refund', 'task_payout', 'task_tip', 'manual_adjustment')
	);

-- Allocated credits currently locked to a task. A row exists iff the task holds
-- allocated credits; deleted on award/refund.
create table if not exists task_funds (
	task_id uuid primary key references tasks(id),
	funder_account_id uuid not null references credit_accounts(id),
	credit_amount bigint not null check (credit_amount > 0)
);

-- Collectibles currently held for a task's reward. A row exists iff the
-- collectible is held for that task; deleted on award/refund. A collectible can
-- be held by at most one task at a time.
create table if not exists task_fund_collectibles (
	task_id uuid not null references tasks(id),
	collectible_id uuid not null references collectibles(id),
	primary key (task_id, collectible_id),
	constraint task_fund_collectibles_collectible_unique unique (collectible_id)
);

create index if not exists task_funds_funder_account_idx
	on task_funds(funder_account_id);

drop table if exists task_escrows;
drop table if exists task_collectible_rewards;
