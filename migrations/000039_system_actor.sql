-- Durable system actor for background sweeps (privacy retention, reservation
-- and task expiry). The row satisfies actor foreign keys on audit/privacy/
-- domain-event tables. No password_credentials row is ever created for it and
-- registration rejects the reserved address, so this identity can never
-- authenticate. The UUID is the fixed constant exposed as core.SystemUserID().
insert into users (id, email)
	values ('00000000-0000-7000-8000-000000000001', 'system@sharecrop.invalid')
	on conflict (id) do nothing;
