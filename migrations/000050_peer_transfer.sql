-- Peer credit sends: a user (or an organization, through a member holding
-- the billing permission) moves credits from their spendable balance to
-- another user or organization as a double-entry pair of peer_transfer
-- ledger rows. SQLite skips the ADD CONSTRAINT statement (unsupported
-- ALTER); the domain layer validates the enum there instead.
alter table ledger_entries drop constraint if exists ledger_entries_kind_check;

alter table ledger_entries add constraint ledger_entries_kind_check check (
	kind in ('signup_grant', 'task_fund', 'task_refund', 'task_payout', 'task_tip', 'manual_adjustment', 'peer_transfer')
);
