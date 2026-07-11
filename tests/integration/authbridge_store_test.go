//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/authbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestAuthBridgeDualRun exercises the auth Store - the largest and most complex
// one - through both the direct-db path and the compiled wasip1 guest + host
// bridge. It covers all 13 methods, including the refresh-token and
// account-token flows whose opaque hash types round-trip through the
// reconstruction constructors added to internal/auth.
func TestAuthBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewAuthStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return authbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := authbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	page := requirePage(t, 50, 0)

	t.Run("credential create, lookup, list", func(t *testing.T) {
		userID := newUserID(t)
		email := mustAuthEmail(t, "authbridge-"+userID.String()+"@example.com")
		hash := mustAuthPasswordHash(t)

		// CreateUserCredential (write through the bridge).
		if _, matched := bridgeStore.CreateUserCredential(ctx, userID, email, hash).(auth.StoreUserAccepted); !matched {
			t.Fatalf("bridge CreateUserCredential did not accept")
		}

		// FindCredentialByEmail: bridge matches direct.
		viaBridge := requireCredentialFound(t, bridgeStore.FindCredentialByEmail(ctx, email))
		direct := requireCredentialFound(t, dbStore.FindCredentialByEmail(ctx, email))
		if viaBridge.UserID != direct.UserID || viaBridge.Email.String() != direct.Email.String() ||
			viaBridge.PasswordHash.String() != direct.PasswordHash.String() || viaBridge.Status != direct.Status {
			t.Errorf("credential mismatch: bridge %+v, direct %+v", viaBridge, direct)
		}
		if viaBridge.UserID != userID {
			t.Errorf("credential user id = %s, want %s", viaBridge.UserID, userID)
		}

		// Missing email rejects the same way both paths.
		missing := mustAuthEmail(t, "authbridge-missing-"+userID.String()+"@example.com")
		if _, matched := bridgeStore.FindCredentialByEmail(ctx, missing).(auth.CredentialMissing); !matched {
			t.Errorf("bridge lookup of missing email did not return CredentialMissing")
		}

		// FindCredentialByUserID through the bridge.
		if requireCredentialFound(t, bridgeStore.FindCredentialByUserID(ctx, userID)).Email.String() != email.String() {
			t.Errorf("FindCredentialByUserID returned the wrong email")
		}

		// ListUsers filtered to this user's email - matches direct and avoids
		// picking up the shared db-checks database's other users.
		bridgeList := requireUsersListed(t, bridgeStore.ListUsers(ctx, email.String(), page))
		directList := requireUsersListed(t, dbStore.ListUsers(ctx, email.String(), page))
		if len(bridgeList) != 1 || len(directList) != 1 || bridgeList[0].ID != userID || directList[0].ID != userID {
			t.Errorf("ListUsers mismatch: bridge %d, direct %d", len(bridgeList), len(directList))
		}
	})

	t.Run("account mutations", func(t *testing.T) {
		userID := newUserID(t)
		email := mustAuthEmail(t, "authbridge-mutate-"+userID.String()+"@example.com")
		if _, matched := dbStore.CreateUserCredential(ctx, userID, email, mustAuthPasswordHash(t)).(auth.StoreUserAccepted); !matched {
			t.Fatalf("seed credential rejected")
		}

		newEmail := mustAuthEmail(t, "authbridge-updated-"+userID.String()+"@example.com")
		if _, matched := bridgeStore.UpdateUserEmail(ctx, userID, newEmail).(auth.AccountMutationAccepted); !matched {
			t.Fatalf("bridge UpdateUserEmail did not accept")
		}
		if requireCredentialFound(t, dbStore.FindCredentialByUserID(ctx, userID)).Email.String() != newEmail.String() {
			t.Errorf("email was not updated through the bridge")
		}

		if _, matched := bridgeStore.UpdatePassword(ctx, userID, mustAuthPasswordHash(t)).(auth.AccountMutationAccepted); !matched {
			t.Errorf("bridge UpdatePassword did not accept")
		}

		if _, matched := bridgeStore.DeactivateUser(ctx, userID).(auth.AccountMutationAccepted); !matched {
			t.Errorf("bridge DeactivateUser did not accept")
		}
		// A deactivated user's credential is no longer looked up - the bridge's
		// mutation persisted, so a direct lookup now misses.
		if _, matched := dbStore.FindCredentialByUserID(ctx, userID).(auth.CredentialMissing); !matched {
			t.Errorf("user was not deactivated through the bridge (still found)")
		}
	})

	t.Run("guest subject", func(t *testing.T) {
		guestID, matched := core.NewGuestID().(core.GuestIDCreated)
		if !matched {
			t.Fatalf("guest id rejected")
		}
		if _, accepted := bridgeStore.CreateGuestSubject(ctx, guestID.Value).(auth.StoreGuestAccepted); !accepted {
			t.Errorf("bridge CreateGuestSubject did not accept")
		}
	})

	t.Run("refresh token store, consume, revoke", func(t *testing.T) {
		user := createUser(t, pool, "authbridge-refresh")
		now := time.Now().UTC()

		record := auth.RefreshTokenRecord{
			ID:        newRefreshTokenID(t),
			FamilyID:  newRefreshTokenID(t),
			Subject:   auth.UserSubject{ID: user},
			Hash:      auth.RefreshTokenHashFromString("refresh-hash-" + newRefreshTokenID(t).String()),
			ExpiresAt: now.Add(time.Hour),
		}
		if _, matched := bridgeStore.StoreRefreshToken(ctx, record).(auth.StoreRefreshTokenAccepted); !matched {
			t.Fatalf("bridge StoreRefreshToken did not accept")
		}

		consumed, matched := bridgeStore.ConsumeRefreshToken(ctx, record.Hash, now).(auth.RefreshTokenConsumed)
		if !matched {
			t.Fatalf("bridge ConsumeRefreshToken did not report consumed")
		}
		if consumed.Family != record.FamilyID {
			t.Errorf("consumed family = %s, want %s", consumed.Family, record.FamilyID)
		}
		if subject, ok := consumed.Subject.(auth.UserSubject); !ok || subject.ID != user {
			t.Errorf("consumed subject = %+v, want UserSubject %s", consumed.Subject, user)
		}

		// A hash that was never stored is not consumed - both paths agree.
		missing := auth.RefreshTokenHashFromString("refresh-missing-" + newRefreshTokenID(t).String())
		if _, matched := bridgeStore.ConsumeRefreshToken(ctx, missing, now).(auth.RefreshTokenNotConsumed); !matched {
			t.Errorf("bridge consume of a missing hash was not NotConsumed")
		}
		if _, matched := dbStore.ConsumeRefreshToken(ctx, missing, now).(auth.RefreshTokenNotConsumed); !matched {
			t.Errorf("direct consume of a missing hash was not NotConsumed")
		}

		// RevokeRefreshFamily through the bridge.
		revokeRecord := auth.RefreshTokenRecord{
			ID:        newRefreshTokenID(t),
			FamilyID:  newRefreshTokenID(t),
			Subject:   auth.UserSubject{ID: user},
			Hash:      auth.RefreshTokenHashFromString("refresh-revoke-" + newRefreshTokenID(t).String()),
			ExpiresAt: now.Add(time.Hour),
		}
		if _, matched := dbStore.StoreRefreshToken(ctx, revokeRecord).(auth.StoreRefreshTokenAccepted); !matched {
			t.Fatalf("seed refresh token for revoke rejected")
		}
		if _, matched := bridgeStore.RevokeRefreshFamily(ctx, revokeRecord.Hash).(auth.RefreshFamilyRevoked); !matched {
			t.Errorf("bridge RevokeRefreshFamily did not report revoked")
		}
	})

	t.Run("account token store and consume", func(t *testing.T) {
		user := createUser(t, pool, "authbridge-account")
		now := time.Now().UTC()
		plain, matched := auth.ParseAccountTokenPlain("account-plain-" + newRefreshTokenID(t).String()).(auth.AccountTokenPlainAccepted)
		if !matched {
			t.Fatalf("account token plain rejected")
		}
		token := auth.AccountToken{
			ID:        newRefreshTokenID(t),
			Plain:     plain.Value,
			Hash:      auth.AccountTokenHashFromString("account-hash-" + newRefreshTokenID(t).String()),
			ExpiresAt: now.Add(time.Hour),
		}
		kind := auth.AccountTokenKindEmailVerification

		if _, matched := bridgeStore.StoreAccountToken(ctx, user, kind, token).(auth.AccountTokenStored); !matched {
			t.Fatalf("bridge StoreAccountToken did not accept")
		}

		consumed, matched := bridgeStore.ConsumeAccountToken(ctx, kind, token.Hash, now).(auth.AccountTokenConsumed)
		if !matched {
			t.Fatalf("bridge ConsumeAccountToken did not report consumed")
		}
		if consumed.UserID != user {
			t.Errorf("consumed user id = %s, want %s", consumed.UserID, user)
		}

		missing := auth.AccountTokenHashFromString("account-missing-" + newRefreshTokenID(t).String())
		if _, matched := bridgeStore.ConsumeAccountToken(ctx, kind, missing, now).(auth.AccountTokenNotConsumed); !matched {
			t.Errorf("bridge consume of a missing account token was not NotConsumed")
		}
		if _, matched := dbStore.ConsumeAccountToken(ctx, kind, missing, now).(auth.AccountTokenNotConsumed); !matched {
			t.Errorf("direct consume of a missing account token was not NotConsumed")
		}
	})
}

func newRefreshTokenID(t *testing.T) core.RefreshTokenID {
	t.Helper()
	created, matched := core.NewRefreshTokenID().(core.RefreshTokenIDCreated)
	if !matched {
		t.Fatalf("refresh token id rejected")
	}
	return created.Value
}

func mustAuthEmail(t *testing.T, raw string) auth.EmailAddress {
	t.Helper()
	accepted, matched := auth.NewEmailAddress(raw).(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("email %q rejected", raw)
	}
	return accepted.Value
}

func mustAuthPasswordHash(t *testing.T) auth.PasswordHash {
	t.Helper()
	secret, matched := auth.NewPasswordSecret("correct horse battery staple").(auth.PasswordSecretAccepted)
	if !matched {
		t.Fatalf("password secret rejected")
	}
	hash, matched := auth.HashPassword(secret.Value).(auth.PasswordHashCreated)
	if !matched {
		t.Fatalf("password hash rejected")
	}
	return hash.Value
}

func requireCredentialFound(t *testing.T, result auth.CredentialLookupResult) auth.CredentialRecord {
	t.Helper()
	found, matched := result.(auth.CredentialFound)
	if !matched {
		t.Fatalf("credential result = %T, want CredentialFound", result)
	}
	return found.Record
}

func requireUsersListed(t *testing.T, result auth.UserDirectoryResult) []auth.UserDirectoryEntry {
	t.Helper()
	listed, matched := result.(auth.UsersListed)
	if !matched {
		t.Fatalf("user directory result = %T, want UsersListed", result)
	}
	return listed.Values
}
