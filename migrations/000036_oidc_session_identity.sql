alter table oidc_sessions add column if not exists username text not null default '';
alter table oidc_sessions add column if not exists email text not null default '';
alter table oidc_sessions add column if not exists role text not null default '';
