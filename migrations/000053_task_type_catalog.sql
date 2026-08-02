-- The task-type catalog widens from six developer-review types to sixteen
-- knowledge-work types (document review, documentation, diagrams, planning,
-- research, data extraction, troubleshooting, code analysis, architecture
-- review, threat analysis). Both task-type CHECK constraints are re-created
-- with the full enum. SQLite skips the ADD CONSTRAINT statements (unsupported
-- ALTER); the domain task-type enum validates there instead.
alter table tasks drop constraint if exists tasks_task_type_check;

alter table tasks add constraint tasks_task_type_check check (
	task_type in (
		'general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing',
		'document_review', 'documentation_writing', 'diagram_writing', 'planning', 'research', 'data_extraction',
		'troubleshooting', 'code_analysis', 'architecture_review', 'threat_analysis'
	)
);

alter table webhook_subscriptions drop constraint if exists webhook_subscriptions_filter_task_type_check;

alter table webhook_subscriptions add constraint webhook_subscriptions_filter_task_type_check check (
	filter_task_type is null
	or filter_task_type in (
		'general', 'code_review', 'security_review', 'product_review', 'ui_ux_review', 'qa_testing',
		'document_review', 'documentation_writing', 'diagram_writing', 'planning', 'research', 'data_extraction',
		'troubleshooting', 'code_analysis', 'architecture_review', 'threat_analysis'
	)
);
