alter table submissions
	drop constraint if exists submissions_state_check;

alter table submissions
	add constraint submissions_state_check check (state in ('submitted', 'invalid', 'accepted', 'rejected', 'changes_requested'));

alter table submissions
	add column if not exists review_note text not null default '',
	add column if not exists reviewed_by_user_id uuid references users(id),
	add column if not exists review_recorded_at timestamptz,
	add column if not exists review_idempotency_key text;

alter table ledger_entries
	drop constraint if exists ledger_entries_kind_check;

alter table ledger_entries
	add constraint ledger_entries_kind_check check (
		kind in ('signup_grant', 'task_escrow', 'task_refund', 'task_payout', 'task_tip', 'manual_adjustment')
	);
