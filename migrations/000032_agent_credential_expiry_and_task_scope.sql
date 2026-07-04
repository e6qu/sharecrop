alter table agent_credentials
	add column expires_at timestamptz null,
	add column task_id uuid null references tasks(id);

alter table agent_credential_scopes
	drop constraint agent_credential_scopes_scope_check;

alter table agent_credential_scopes
	add constraint agent_credential_scopes_scope_check check (
		scope in (
			'tasks_read', 'tasks_write', 'submissions_write', 'submissions_read', 'submissions_review',
			'org_read', 'org_manage',
			'collectibles_read', 'collectibles_manage',
			'notifications_read', 'notifications_manage',
			'users_read',
			'ledger_read',
			'moderation_read', 'moderation_manage',
			'privacy_read', 'privacy_manage',
			'platform_admin',
			'credentials_manage'
		)
	);

drop table if exists task_capability_tokens;
