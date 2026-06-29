alter table privacy_requests
	add column if not exists redacted_field_count integer not null default 0;

alter table privacy_requests
	drop constraint if exists privacy_requests_redacted_field_count_check;

alter table privacy_requests
	add constraint privacy_requests_redacted_field_count_check check (redacted_field_count >= 0);
