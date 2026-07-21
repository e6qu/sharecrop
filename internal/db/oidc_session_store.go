package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type OpenIDConnectSessionStore struct{ db Beginner }

func NewOpenIDConnectSessionStore(db Beginner) OpenIDConnectSessionStore {
	return OpenIDConnectSessionStore{db: db}
}

func (store OpenIDConnectSessionStore) StoreOpenIDConnectSession(ctx context.Context, hash auth.RefreshTokenHash, session auth.OpenIDConnectSession) auth.StoreOpenIDConnectSessionResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.StoreOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin OpenID Connect session transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var familyID string
	if err := tx.QueryRow(ctx, "select family_id::text from refresh_tokens where token_hash = $1", hash.String()).Scan(&familyID); err != nil {
		return auth.StoreOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "load OpenID Connect refresh-token family failed")}
	}
	_, err = tx.Exec(ctx, `
		insert into oidc_sessions (
			family_id, provider, issuer, subject, sid, username, email, role,
			raw_id_token, client_id, end_session_endpoint, post_logout_redirect_uri, expires_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`, familyID, session.Provider, session.Issuer, session.Subject, session.SID, session.Username,
		session.Email, session.Role, session.RawIDToken, session.ClientID, session.EndSessionEndpoint,
		session.PostLogoutRedirectURI, session.ExpiresAt)
	if err != nil {
		return auth.StoreOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "store OpenID Connect session failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.StoreOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit OpenID Connect session failed")}
	}
	return auth.OpenIDConnectSessionStored{}
}

func (store OpenIDConnectSessionStore) FindOpenIDConnectSession(ctx context.Context, hash auth.RefreshTokenHash) auth.FindOpenIDConnectSessionResult {
	var session auth.OpenIDConnectSession
	err := store.db.QueryRow(ctx, `
		select s.provider, s.issuer, s.subject, s.sid, s.username, s.email, s.role,
			s.raw_id_token, s.client_id, s.end_session_endpoint, s.post_logout_redirect_uri, s.expires_at
		from oidc_sessions s
		join refresh_tokens r on r.family_id = s.family_id
		where r.token_hash = $1
	`, hash.String()).Scan(&session.Provider, &session.Issuer, &session.Subject, &session.SID,
		&session.Username, &session.Email, &session.Role, &session.RawIDToken, &session.ClientID,
		&session.EndSessionEndpoint, &session.PostLogoutRedirectURI, &session.ExpiresAt)
	if errors.Is(err, ErrNoRows) {
		return auth.OpenIDConnectSessionNotFound{}
	}
	if err != nil {
		return auth.FindOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "load OpenID Connect session failed")}
	}
	return auth.OpenIDConnectSessionFound{Session: session}
}

// RotateOpenIDConnectSession confirms the invariant rather than performing a
// move: rows are addressed by refresh-token family, and refreshing continues
// the family, so the replacement token already resolves to the same session.
func (store OpenIDConnectSessionStore) RotateOpenIDConnectSession(ctx context.Context, _, next auth.RefreshTokenHash) auth.RotateOpenIDConnectSessionResult {
	switch found := store.FindOpenIDConnectSession(ctx, next).(type) {
	case auth.OpenIDConnectSessionFound:
		return auth.OpenIDConnectSessionRotated{Session: found.Session}
	case auth.OpenIDConnectSessionNotFound:
		return auth.OpenIDConnectSessionNotRotated{}
	default:
		return auth.RotateOpenIDConnectSessionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "confirm OpenID Connect session rotation failed")}
	}
}

func (store OpenIDConnectSessionStore) ApplyFrontchannelLogout(ctx context.Context, claim auth.OpenIDConnectFrontchannelLogout) auth.FrontchannelLogoutResult {
	_, err := store.db.Exec(ctx, `
		update refresh_tokens set status = 'revoked'
		where status = 'active' and family_id in (
			select family_id from oidc_sessions
			where provider = $1 and issuer = $2 and client_id = $3 and sid = $4
		)
	`, claim.Provider, claim.Issuer, claim.ClientID, claim.SID)
	if err != nil {
		return auth.FrontchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke Front-Channel Logout sessions failed")}
	}
	return auth.FrontchannelLogoutApplied{}
}

func (store OpenIDConnectSessionStore) ApplyBackchannelLogout(ctx context.Context, claim auth.OpenIDConnectLogoutClaim, now time.Time) auth.BackchannelLogoutResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return auth.BackchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin Back-Channel Logout transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "delete from oidc_logout_claims where expires_at <= $1", now); err != nil {
		return auth.BackchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "expire Back-Channel Logout replay claims failed")}
	}
	inserted, err := tx.Exec(ctx, `
		insert into oidc_logout_claims (provider, issuer, client_id, jti, expires_at)
		values ($1, $2, $3, $4, $5)
		on conflict (provider, issuer, client_id, jti) do nothing
	`, claim.Provider, claim.Issuer, claim.ClientID, claim.JWTID, claim.ExpiresAt)
	if err != nil {
		return auth.BackchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "claim Back-Channel Logout token failed")}
	}
	if inserted == 0 {
		return auth.BackchannelLogoutReplay{}
	}
	selection := "s.subject = $4"
	selected := claim.Subject
	if claim.SID != "" {
		selection = "s.sid = $4"
		selected = claim.SID
	}
	_, err = tx.Exec(ctx, `
		update refresh_tokens set status = 'revoked'
		where status = 'active' and family_id in (
			select s.family_id from oidc_sessions s
			where s.provider = $1 and s.issuer = $2 and s.client_id = $3 and `+selection+`
		)
	`, claim.Provider, claim.Issuer, claim.ClientID, selected)
	if err != nil {
		return auth.BackchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke Back-Channel Logout sessions failed")}
	}
	if err := tx.Commit(ctx); err != nil {
		return auth.BackchannelLogoutRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit Back-Channel Logout failed")}
	}
	return auth.BackchannelLogoutApplied{}
}
