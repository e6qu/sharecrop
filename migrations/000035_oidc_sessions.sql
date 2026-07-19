create table if not exists oidc_sessions (
	family_id uuid primary key references refresh_tokens(id),
	provider text not null,
	issuer text not null,
	subject text not null,
	sid text not null,
	raw_id_token text not null,
	client_id text not null,
	end_session_endpoint text not null,
	post_logout_redirect_uri text not null,
	expires_at timestamptz not null
);

create index if not exists oidc_sessions_subject_idx
	on oidc_sessions(provider, issuer, client_id, subject);

create index if not exists oidc_sessions_sid_idx
	on oidc_sessions(provider, issuer, client_id, sid);

create table if not exists oidc_logout_claims (
	provider text not null,
	issuer text not null,
	client_id text not null,
	jti text not null,
	expires_at timestamptz not null,
	primary key (provider, issuer, client_id, jti)
);

create index if not exists oidc_logout_claims_expiry_idx
	on oidc_logout_claims(expires_at);
