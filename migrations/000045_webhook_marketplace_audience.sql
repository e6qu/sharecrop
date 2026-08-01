-- Webhook subscriptions gain an audience. 'recipient' (the default) keeps the
-- existing behavior: deliveries expand for events whose recipient set contains
-- the subscription owner. 'marketplace' is the "agents ask for work" channel:
-- deliveries expand for every task_opened event whose task is public and open,
-- regardless of recipients, optionally narrowed by task type and a minimum
-- credit reward. The filters apply only to marketplace subscriptions. SQLite
-- skips the ADD CONSTRAINT statements (unsupported ALTER); the domain layer
-- validates there instead.
alter table webhook_subscriptions
	add column if not exists audience text not null default 'recipient',
	add column if not exists filter_task_type text,
	add column if not exists filter_min_credit_reward bigint;

alter table webhook_subscriptions
	add constraint webhook_subscriptions_audience_check check (audience in ('recipient', 'marketplace'));

alter table webhook_subscriptions
	add constraint webhook_subscriptions_filter_task_type_check check (
		filter_task_type is null
		or filter_task_type in ('general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing')
	);

alter table webhook_subscriptions
	add constraint webhook_subscriptions_min_reward_check check (
		filter_min_credit_reward is null or filter_min_credit_reward > 0
	);

alter table webhook_subscriptions
	add constraint webhook_subscriptions_recipient_filters_check check (
		audience = 'marketplace'
		or (filter_task_type is null and filter_min_credit_reward is null)
	);
