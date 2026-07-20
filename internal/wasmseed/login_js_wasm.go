//go:build js && wasm

package wasmseed

import (
	"context"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/wasibridge/appmux"
)

// LoginDemoAdmin logs the demo admin (mara) in over the given stores, returning
// a fresh session. It is used after loading a pre-generated seed snapshot so the
// demo boots already logged in without re-running the bcrypt-heavy seed.
func LoginDemoAdmin(ctx context.Context, secret auth.AccessTokenSecret, stores appmux.Stores) SeedResult {
	authServiceResult := auth.NewService(stores.Auth, secret, auth.SystemClock{})
	authService, matched := authServiceResult.(auth.ServiceCreated)
	if !matched {
		return seedErr(authServiceResult.(auth.ServiceRejected).Reason.Description())
	}

	email, matched := auth.NewEmailAddress(demoUsers[0].email).(auth.EmailAddressAccepted)
	if !matched {
		return seedErr("demo admin email rejected")
	}
	password, matched := auth.NewPasswordSecret(demoPassword).(auth.PasswordSecretAccepted)
	if !matched {
		return seedErr("demo password rejected")
	}

	loginResult := authService.Value.Login(ctx, email.Value, password.Value)
	accepted, matched := loginResult.(auth.LoginAccepted)
	if !matched {
		return seedErr(loginResult.(auth.LoginRejected).Reason.Description())
	}
	return SeedResult{AdminUserID: accepted.Subject.ID, AdminRefreshToken: refreshCookie(accepted.RefreshToken)}
}
