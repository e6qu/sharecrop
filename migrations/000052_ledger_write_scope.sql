-- The ledger_write scope gates the MCP peer credit send tool
-- (sharecrop.send_credits). Re-create both credential scope CHECK
-- constraints with the full current scope enum so the new scope is mintable,
-- following migration 000042's pattern. SQLite skips these ALTER statements
-- (unsupported) and strips inline CHECKs; the domain scope enum validates
-- there instead.
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
			'ledger_read', 'ledger_write',
			'moderation_read', 'moderation_manage',
			'privacy_read', 'privacy_manage',
			'platform_admin',
			'credentials_manage',
			'webhooks_read', 'webhooks_manage'
		)
	);

alter table org_credential_scopes
	drop constraint org_credential_scopes_scope_check;

alter table org_credential_scopes
	add constraint org_credential_scopes_scope_check check (
		scope in (
			'tasks_read', 'tasks_write', 'submissions_write', 'submissions_read', 'submissions_review',
			'org_read', 'org_manage',
			'collectibles_read', 'collectibles_manage',
			'notifications_read', 'notifications_manage',
			'users_read',
			'ledger_read', 'ledger_write',
			'moderation_read', 'moderation_manage',
			'privacy_read', 'privacy_manage',
			'platform_admin',
			'credentials_manage',
			'webhooks_read', 'webhooks_manage'
		)
	);
