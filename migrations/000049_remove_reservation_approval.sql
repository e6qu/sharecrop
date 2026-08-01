-- Remove the reservation approval gate. Reserving a task now always yields
-- an immediately-active reservation: the approval_required participation
-- policy folds into reservation_required, and the requested/declined
-- reservation states become historical-only (stored rows keep them, no code
-- path produces them again).
update tasks set participation_policy = 'reservation_required' where participation_policy = 'approval_required';

-- Pending approval requests are settled once: per task with no active
-- reservation, the oldest requested reservation is promoted to active (the
-- worker who asked first gets the task). Every other requested reservation
-- is declined — the final historical use of that state.
update task_reservations
set state = 'active', state_recorded_at = now()
where state = 'requested'
	and not exists (
		select 1 from task_reservations as active_reservation
		where active_reservation.task_id = task_reservations.task_id
		and active_reservation.state = 'active'
	)
	and id = (
		select oldest.id from task_reservations as oldest
		where oldest.task_id = task_reservations.task_id and oldest.state = 'requested'
		order by oldest.created_at asc, oldest.id asc
		limit 1
	);

update task_reservations
set state = 'declined', state_recorded_at = now()
where state = 'requested';

-- The participation policy enum narrows to open | reservation_required.
-- SQLite skips the ADD CONSTRAINT statement (unsupported ALTER); the domain
-- layer validates the enum there instead.
alter table tasks drop constraint if exists tasks_participation_policy_check;

alter table tasks add constraint tasks_participation_policy_check check (
	participation_policy in ('open', 'reservation_required')
);
