-- When accepting a submission closes its task, every other submission still
-- in 'submitted' moves to the terminal 'superseded' state in the same
-- transaction (the competing work can no longer be reviewed). SQLite skips
-- the ADD CONSTRAINT statement (unsupported ALTER); the domain layer
-- validates the enum there instead.
alter table submissions
	drop constraint if exists submissions_state_check;

alter table submissions
	add constraint submissions_state_check check (
		state in ('submitted', 'invalid', 'accepted', 'rejected', 'changes_requested', 'superseded')
	);
