delete from submission_sensitive_fields
	where submission_id in (select id from submissions where submitter_kind = 'anonymous');

delete from submission_validation_errors
	where submission_id in (select id from submissions where submitter_kind = 'anonymous');

delete from submission_receipt_tokens
	where submission_id in (select id from submissions where submitter_kind = 'anonymous');

delete from submissions where submitter_kind = 'anonymous';

drop index if exists submissions_user_idx;

alter table submissions drop constraint if exists submissions_submitter_check;
alter table submissions drop constraint if exists submissions_submitter_kind_check;
alter table submissions drop column if exists submitter_kind;
alter table submissions drop column if exists wallet_address;
alter table submissions alter column user_id set not null;

create index if not exists submissions_user_idx
	on submissions(user_id);
