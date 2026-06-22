alter table credit_accounts drop constraint if exists credit_accounts_owner_kind_check;
alter table credit_accounts drop constraint if exists credit_accounts_owner_check;

alter table credit_accounts add column if not exists organization_id uuid references organizations(id);

alter table credit_accounts add constraint credit_accounts_owner_kind_check
	check (owner_kind in ('user', 'organization'));

alter table credit_accounts add constraint credit_accounts_owner_check check (
	(owner_kind = 'user' and user_id is not null and organization_id is null)
	or
	(owner_kind = 'organization' and organization_id is not null and user_id is null)
);

create unique index if not exists credit_accounts_organization_unique
	on credit_accounts(organization_id)
	where organization_id is not null;
