package wasmdemo

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type counterLedgerIDs struct {
	next int
}

func (ids *counterLedgerIDs) NextSubmissionID() string   { panic("not needed") }
func (ids *counterLedgerIDs) NextCommentID() string      { panic("not needed") }
func (ids *counterLedgerIDs) NextReservationID() string  { panic("not needed") }
func (ids *counterLedgerIDs) NextNotificationID() string { panic("not needed") }

// NextLedgerEntryID must return a real UUID-shaped id, not just a unique
// string: entries are round-tripped through core.ParseLedgerEntryID when
// read back (e.g. by ListEntries), which rejects non-UUID input.
func (ids *counterLedgerIDs) NextLedgerEntryID() string {
	ids.next++
	return core.NewLedgerEntryID().(core.LedgerEntryIDCreated).Value.String()
}

type fixedTestClock struct {
	now time.Time
}

func (clock fixedTestClock) Now() time.Time { return clock.now }

func testAccessTokenSecret(t *testing.T) auth.AccessTokenSecret {
	t.Helper()
	result := auth.NewAccessTokenSecret("test-secret-at-least-16-bytes-long")
	accepted, matched := result.(auth.AccessTokenSecretAccepted)
	if !matched {
		t.Fatalf("new access token secret failed: %#v", result)
	}
	return accepted.Value
}

func testEmail(t *testing.T, raw string) auth.EmailAddress {
	t.Helper()
	result := auth.NewEmailAddress(raw)
	accepted, matched := result.(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("new email %q failed: %#v", raw, result)
	}
	return accepted.Value
}

func testPassword(t *testing.T, raw string) auth.PasswordSecret {
	t.Helper()
	result := auth.NewPasswordSecret(raw)
	accepted, matched := result.(auth.PasswordSecretAccepted)
	if !matched {
		t.Fatalf("new password failed: %#v", result)
	}
	return accepted.Value
}

func newTestAuthService(t *testing.T) (auth.Service, *counterLedgerIDs) {
	t.Helper()
	ids := &counterLedgerIDs{}
	store := NewAuthBrowserStore(newTestBrowserStorage(), ids)
	result := auth.NewService(store, testAccessTokenSecret(t), fixedTestClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	created, matched := result.(auth.ServiceCreated)
	if !matched {
		t.Fatalf("new auth service failed: %#v", result)
	}
	return created.Value, ids
}

func TestAuthBrowserStoreRegisterAndLogin(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	password := testPassword(t, "correct horse battery staple")

	registerResult := service.Register(ctx, email, password)
	registered, matched := registerResult.(auth.RegisterAccepted)
	if !matched {
		t.Fatalf("register: want RegisterAccepted, got %#v", registerResult)
	}
	if registered.RefreshToken.String() == "" {
		t.Fatalf("register: refresh token is empty")
	}

	loginResult := service.Login(ctx, email, password)
	if _, matched := loginResult.(auth.LoginAccepted); !matched {
		t.Fatalf("login with correct password: want LoginAccepted, got %#v", loginResult)
	}
}

func TestAuthBrowserStoreRejectsDuplicateRegistration(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	password := testPassword(t, "correct horse battery staple")

	if _, matched := service.Register(ctx, email, password).(auth.RegisterAccepted); !matched {
		t.Fatalf("first register failed")
	}
	secondResult := service.Register(ctx, email, testPassword(t, "a different password"))
	if _, matched := secondResult.(auth.RegisterRejected); !matched {
		t.Fatalf("duplicate register: want RegisterRejected, got %#v", secondResult)
	}
}

func TestAuthBrowserStoreRejectsWrongPassword(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	service.Register(ctx, email, testPassword(t, "correct horse battery staple"))

	loginResult := service.Login(ctx, email, testPassword(t, "wrong password entirely"))
	if _, matched := loginResult.(auth.LoginRejected); !matched {
		t.Fatalf("login with wrong password: want LoginRejected, got %#v", loginResult)
	}
}

func TestAuthBrowserStoreGrantsSignupBonus(t *testing.T) {
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}
	store := NewAuthBrowserStore(storage, ids)
	serviceResult := auth.NewService(store, testAccessTokenSecret(t), fixedTestClock{now: time.Now()})
	service := serviceResult.(auth.ServiceCreated).Value
	ctx := context.Background()
	email := testEmail(t, "person@example.com")

	registerResult := service.Register(ctx, email, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	// Assert on the entry itself, not LedgerBalance's total: LedgerBalance
	// (interaction_storage.go) hardcodes a 100-credit baseline for every
	// user-kind owner, on the assumption that the signup grant is implicit
	// and never has its own ledger entry - true for wasmdemo's current
	// simplified auth handler, which doesn't write one. This new store
	// writes a real, explicit entry (matching the real backend), so the
	// two would double-count if this store's auth ever replaces the live
	// handler without also removing that hardcoded baseline at the same
	// time - a cutover-time fix, not a bug in this store.
	entriesResult := ListLedgerEntries(storage, "user", registerResult.Subject.ID.String(), StoredListPage{limit: 10, offset: 0})
	entries, matched := entriesResult.(LedgerEntriesStored)
	if !matched {
		t.Fatalf("list ledger entries: want LedgerEntriesStored, got %#v", entriesResult)
	}
	if len(entries.Values) != 1 {
		t.Fatalf("ledger entries count = %d, want 1", len(entries.Values))
	}
	if entries.Values[0].Kind != "signup_grant" || entries.Values[0].Amount != 100 {
		t.Fatalf("signup grant entry = %+v, want kind=signup_grant amount=100", entries.Values[0])
	}
}

func TestAuthBrowserStoreRefreshRotatesToken(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	registered := service.Register(ctx, testEmail(t, "person@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	refreshResult := service.Refresh(ctx, registered.RefreshToken)
	refreshed, matched := refreshResult.(auth.RefreshAccepted)
	if !matched {
		t.Fatalf("refresh: want RefreshAccepted, got %#v", refreshResult)
	}
	if refreshed.RefreshToken.String() == registered.RefreshToken.String() {
		t.Fatalf("refresh token was not rotated")
	}

	// The old (now-consumed) refresh token must no longer work.
	staleResult := service.Refresh(ctx, registered.RefreshToken)
	if _, matched := staleResult.(auth.RefreshRejected); !matched {
		t.Fatalf("refresh with stale token: want RefreshRejected, got %#v", staleResult)
	}
}

func TestAuthBrowserStoreRefreshReuseRevokesFamily(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	registered := service.Register(ctx, testEmail(t, "person@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	rotatedResult := service.Refresh(ctx, registered.RefreshToken).(auth.RefreshAccepted)

	// Reusing the already-consumed original token is treated as theft: the
	// whole family (including the token that replaced it) gets revoked.
	service.Refresh(ctx, registered.RefreshToken)

	afterReuseResult := service.Refresh(ctx, rotatedResult.RefreshToken)
	if _, matched := afterReuseResult.(auth.RefreshRejected); !matched {
		t.Fatalf("refresh after reuse: want the whole family revoked (RefreshRejected), got %#v", afterReuseResult)
	}
}

func TestAuthBrowserStoreLogoutRevokesFamily(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	registered := service.Register(ctx, testEmail(t, "person@example.com"), testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	logoutResult := service.Logout(ctx, registered.RefreshToken)
	if _, matched := logoutResult.(auth.LogoutDone); !matched {
		t.Fatalf("logout: want LogoutDone, got %#v", logoutResult)
	}

	refreshResult := service.Refresh(ctx, registered.RefreshToken)
	if _, matched := refreshResult.(auth.RefreshRejected); !matched {
		t.Fatalf("refresh after logout: want RefreshRejected, got %#v", refreshResult)
	}
}

func TestAuthBrowserStoreChangePassword(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	registered := service.Register(ctx, email, testPassword(t, "correct horse battery staple")).(auth.RegisterAccepted)

	changeResult := service.ChangePassword(ctx, registered.Subject.ID, testPassword(t, "correct horse battery staple"), testPassword(t, "a brand new password"))
	if _, matched := changeResult.(auth.AccountActionAccepted); !matched {
		t.Fatalf("change password: want AccountActionAccepted, got %#v", changeResult)
	}

	if _, matched := service.Login(ctx, email, testPassword(t, "correct horse battery staple")).(auth.LoginRejected); !matched {
		t.Fatalf("login with old password after change: want LoginRejected")
	}
	if _, matched := service.Login(ctx, email, testPassword(t, "a brand new password")).(auth.LoginAccepted); !matched {
		t.Fatalf("login with new password after change: want LoginAccepted")
	}
}

func TestAuthBrowserStoreDeactivateAccountBlocksLogin(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	password := testPassword(t, "correct horse battery staple")
	registered := service.Register(ctx, email, password).(auth.RegisterAccepted)

	deactivateResult := service.DeactivateAccount(ctx, registered.Subject.ID)
	if _, matched := deactivateResult.(auth.AccountActionAccepted); !matched {
		t.Fatalf("deactivate: want AccountActionAccepted, got %#v", deactivateResult)
	}

	loginResult := service.Login(ctx, email, password)
	if _, matched := loginResult.(auth.LoginRejected); !matched {
		t.Fatalf("login after deactivate: want LoginRejected, got %#v", loginResult)
	}
}

func TestAuthBrowserStorePasswordResetFlow(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	email := testEmail(t, "person@example.com")
	service.Register(ctx, email, testPassword(t, "correct horse battery staple"))

	issueResult := service.RequestPasswordReset(ctx, email)
	issued, matched := issueResult.(auth.AccountTokenIssued)
	if !matched {
		t.Fatalf("request password reset: want AccountTokenIssued, got %#v", issueResult)
	}

	resetResult := service.ResetPassword(ctx, issued.Token, testPassword(t, "a reset password"))
	if _, matched := resetResult.(auth.AccountActionAccepted); !matched {
		t.Fatalf("reset password: want AccountActionAccepted, got %#v", resetResult)
	}

	if _, matched := service.Login(ctx, email, testPassword(t, "a reset password")).(auth.LoginAccepted); !matched {
		t.Fatalf("login with reset password: want LoginAccepted")
	}

	// A consumed reset token cannot be used again.
	secondResetResult := service.ResetPassword(ctx, issued.Token, testPassword(t, "yet another password"))
	if _, matched := secondResetResult.(auth.AccountActionRejected); !matched {
		t.Fatalf("reusing a consumed reset token: want AccountActionRejected, got %#v", secondResetResult)
	}
}

func TestAuthBrowserStoreGuestSession(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()

	guestResult := service.CreateGuest(ctx)
	guest, matched := guestResult.(auth.GuestAccepted)
	if !matched {
		t.Fatalf("create guest: want GuestAccepted, got %#v", guestResult)
	}

	refreshResult := service.Refresh(ctx, guest.RefreshToken)
	if _, matched := refreshResult.(auth.RefreshAccepted); !matched {
		t.Fatalf("refresh guest session: want RefreshAccepted, got %#v", refreshResult)
	}
}

func TestAuthBrowserStoreListUsers(t *testing.T) {
	service, _ := newTestAuthService(t)
	ctx := context.Background()
	service.Register(ctx, testEmail(t, "alice@example.com"), testPassword(t, "correct horse battery staple"))
	service.Register(ctx, testEmail(t, "bob@example.com"), testPassword(t, "correct horse battery staple"))

	pageResult := core.NewPage(10, 0)
	page := pageResult.(core.PageAccepted).Value
	listResult := service.ListUsers(ctx, "", page)
	listed, matched := listResult.(auth.UsersListed)
	if !matched {
		t.Fatalf("list users: want UsersListed, got %#v", listResult)
	}
	if len(listed.Values) != 2 {
		t.Fatalf("listed users count = %d, want 2", len(listed.Values))
	}
	// Ordered by email ascending: alice before bob.
	if listed.Values[0].Email.String() != "alice@example.com" {
		t.Fatalf("first listed user = %q, want alice@example.com", listed.Values[0].Email.String())
	}
}
