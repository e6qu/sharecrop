package db

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
)

func TestOpenIDConnectSessionStorePersistsAtomicLogoutReplay(t *testing.T) {
	ctx := context.Background()
	handle := NewSQLite(openSQLiteWithSchema(t))
	userID := newUserIDForTest(t)
	if _, err := handle.Exec(ctx, "insert into users (id, email) values ($1, $2)", userID.String(), "oidc-session@example.test"); err != nil {
		t.Fatal(err)
	}

	firstToken := auth.NewRefreshToken(time.Now()).(auth.RefreshTokenCreated).Value
	seedOIDCRefreshToken(t, handle, userID.String(), firstToken)
	firstStore := NewOpenIDConnectSessionStore(handle)
	session := auth.OpenIDConnectSession{
		Provider: "shauth", Issuer: "https://auth.example.test/", Subject: "subject-1", SID: "sid-1",
		RawIDToken: "signed.id.token", ClientID: "sharecrop", EndSessionEndpoint: "https://auth.example.test/logout",
		PostLogoutRedirectURI: "https://sharecrop.example.test/api/auth/signed-out", ExpiresAt: time.Now().Add(time.Hour),
	}
	if _, ok := firstStore.StoreOpenIDConnectSession(ctx, firstToken.Hash, session).(auth.OpenIDConnectSessionStored); !ok {
		t.Fatal("OpenID Connect session was not stored")
	}
	found, ok := firstStore.FindOpenIDConnectSession(ctx, firstToken.Hash).(auth.OpenIDConnectSessionFound)
	if !ok || found.Session.RawIDToken != session.RawIDToken || found.Session.Issuer != session.Issuer || found.Session.PostLogoutRedirectURI != session.PostLogoutRedirectURI {
		t.Fatalf("stored OpenID Connect session = %#v", found)
	}
	if _, ok := firstStore.ApplyFrontchannelLogout(ctx, auth.OpenIDConnectFrontchannelLogout{Provider: "shauth", Issuer: session.Issuer, ClientID: session.ClientID, SID: session.SID}).(auth.FrontchannelLogoutApplied); !ok {
		t.Fatal("Front-Channel Logout was not applied")
	}
	assertRefreshTokenStatus(t, handle, firstToken.Hash, "revoked")
	if _, err := handle.Exec(ctx, "update refresh_tokens set status = 'active' where token_hash = $1", firstToken.Hash.String()); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	claim := auth.OpenIDConnectLogoutClaim{
		Provider: "shauth", Issuer: session.Issuer, ClientID: "sharecrop", JWTID: "logout-jti",
		ExpiresAt: now.Add(10 * time.Minute), SID: session.SID, Subject: session.Subject,
	}
	if _, ok := firstStore.ApplyBackchannelLogout(ctx, claim, now).(auth.BackchannelLogoutApplied); !ok {
		t.Fatal("first Back-Channel Logout was not applied")
	}
	assertRefreshTokenStatus(t, handle, firstToken.Hash, "revoked")

	// A new session may legitimately reuse the provider sid. A replay through a
	// newly constructed store must not revoke the later refresh-token family.
	secondToken := auth.NewRefreshToken(time.Now()).(auth.RefreshTokenCreated).Value
	seedOIDCRefreshToken(t, handle, userID.String(), secondToken)
	restartedStore := NewOpenIDConnectSessionStore(handle)
	if _, ok := restartedStore.StoreOpenIDConnectSession(ctx, secondToken.Hash, session).(auth.OpenIDConnectSessionStored); !ok {
		t.Fatal("later OpenID Connect session was not stored")
	}
	if _, ok := restartedStore.ApplyBackchannelLogout(ctx, claim, now).(auth.BackchannelLogoutReplay); !ok {
		t.Fatal("durable logout-token replay was not rejected")
	}
	assertRefreshTokenStatus(t, handle, secondToken.Hash, "active")
	subjectClaim := claim
	subjectClaim.JWTID = "logout-subject"
	subjectClaim.SID = ""
	if _, ok := restartedStore.ApplyBackchannelLogout(ctx, subjectClaim, now).(auth.BackchannelLogoutApplied); !ok {
		t.Fatal("subject Back-Channel Logout was not applied")
	}
	assertRefreshTokenStatus(t, handle, secondToken.Hash, "revoked")
}

func seedOIDCRefreshToken(t *testing.T, handle Beginner, userID string, token auth.RefreshTokenIssued) {
	t.Helper()
	_, err := handle.Exec(context.Background(), `
		insert into refresh_tokens (id, family_id, token_hash, subject_kind, user_id, status, expires_at)
		values ($1, $1, $2, 'user', $3, 'active', $4)
	`, token.ID.String(), token.Hash.String(), userID, token.ExpiresAt)
	if err != nil {
		t.Fatal(err)
	}
}

func assertRefreshTokenStatus(t *testing.T, handle Beginner, hash auth.RefreshTokenHash, want string) {
	t.Helper()
	var got string
	if err := handle.QueryRow(context.Background(), "select status from refresh_tokens where token_hash = $1", hash.String()).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("refresh token status = %q, want %q", got, want)
	}
}
