package auth

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestPasswordHashVerifiesSecret(t *testing.T) {
	secret := acceptedPassword(t, "correct horse battery staple")
	result := HashPassword(secret)
	created, matched := result.(PasswordHashCreated)
	if !matched {
		t.Fatalf("result = %T, want PasswordHashCreated", result)
	}

	verification := VerifyPassword(secret, created.Value)
	if _, matched := verification.(PasswordAccepted); !matched {
		t.Fatalf("verification = %T, want PasswordAccepted", verification)
	}
}

func TestAccessTokenIsSignedJWT(t *testing.T) {
	userResult := core.NewUserID()
	userCreated := userResult.(core.UserIDCreated)
	secret := acceptedAccessTokenSecret(t)

	result := SignAccessToken(secret, UserSubject{ID: userCreated.Value}, fixedTestTime())
	accepted, matched := result.(AccessTokenAccepted)
	if !matched {
		t.Fatalf("result = %T, want AccessTokenAccepted", result)
	}

	if countJWTSeparators(accepted.Value.String()) != 2 {
		t.Fatalf("token = %q, want three JWT segments", accepted.Value.String())
	}
}

func TestAccessTokenVerifiesUserSubject(t *testing.T) {
	userResult := core.NewUserID()
	userCreated := userResult.(core.UserIDCreated)
	secret := acceptedAccessTokenSecret(t)
	tokenResult := SignAccessToken(secret, UserSubject{ID: userCreated.Value}, fixedTestTime())
	tokenAccepted := tokenResult.(AccessTokenAccepted)

	verifyResult := VerifyAccessToken(secret, tokenAccepted.Value, fixedTestTime())
	verified, matched := verifyResult.(SubjectVerified)
	if !matched {
		t.Fatalf("verify result = %T, want SubjectVerified", verifyResult)
	}

	subject, matched := verified.Value.(UserSubject)
	if !matched {
		t.Fatalf("subject = %T, want UserSubject", verified.Value)
	}

	if subject.ID.String() != userCreated.Value.String() {
		t.Fatalf("subject id = %q, want %q", subject.ID.String(), userCreated.Value.String())
	}
}

func TestAccessTokenRejectsExpiredToken(t *testing.T) {
	userResult := core.NewUserID()
	userCreated := userResult.(core.UserIDCreated)
	secret := acceptedAccessTokenSecret(t)
	tokenResult := SignAccessToken(secret, UserSubject{ID: userCreated.Value}, fixedTestTime())
	tokenAccepted := tokenResult.(AccessTokenAccepted)

	verifyResult := VerifyAccessToken(secret, tokenAccepted.Value, fixedTestTime().Add(16*time.Minute))
	if _, matched := verifyResult.(SubjectVerifyRejected); !matched {
		t.Fatalf("verify result = %T, want SubjectVerifyRejected", verifyResult)
	}
}

func TestServiceRegistersLogsInAndRefreshesUser(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)
	email := acceptedEmail(t, "person@example.com")
	password := acceptedPassword(t, "correct horse battery staple")

	registerResult := service.Register(context.Background(), email, password)
	registerAccepted, registerMatched := registerResult.(RegisterAccepted)
	if !registerMatched {
		t.Fatalf("register result = %T, want RegisterAccepted", registerResult)
	}

	loginResult := service.Login(context.Background(), email, password)
	if _, matched := loginResult.(LoginAccepted); !matched {
		t.Fatalf("login result = %T, want LoginAccepted", loginResult)
	}

	refreshResult := service.Refresh(context.Background(), registerAccepted.RefreshToken)
	if _, matched := refreshResult.(RefreshAccepted); !matched {
		t.Fatalf("refresh result = %T, want RefreshAccepted", refreshResult)
	}

	reuseResult := service.Refresh(context.Background(), registerAccepted.RefreshToken)
	if _, matched := reuseResult.(RefreshRejected); !matched {
		t.Fatalf("reuse result = %T, want RefreshRejected", reuseResult)
	}
}

func TestServiceCreatesGuestSession(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)

	result := service.CreateGuest(context.Background())
	accepted, matched := result.(GuestAccepted)
	if !matched {
		t.Fatalf("result = %T, want GuestAccepted", result)
	}

	refreshResult := service.Refresh(context.Background(), accepted.RefreshToken)
	if _, matched := refreshResult.(RefreshAccepted); !matched {
		t.Fatalf("refresh result = %T, want RefreshAccepted", refreshResult)
	}
}

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return fixedTestTime()
}

func fixedTestTime() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}

func acceptedService(t *testing.T, store *memoryStore) Service {
	t.Helper()
	result := NewService(store, acceptedAccessTokenSecret(t), fixedClock{})
	created, matched := result.(ServiceCreated)
	if !matched {
		t.Fatalf("result = %T, want ServiceCreated", result)
	}
	return created.Value
}

func acceptedEmail(t *testing.T, raw string) EmailAddress {
	t.Helper()
	result := NewEmailAddress(raw)
	accepted, matched := result.(EmailAddressAccepted)
	if !matched {
		t.Fatalf("email result = %T, want EmailAddressAccepted", result)
	}
	return accepted.Value
}

func acceptedPassword(t *testing.T, raw string) PasswordSecret {
	t.Helper()
	result := NewPasswordSecret(raw)
	accepted, matched := result.(PasswordSecretAccepted)
	if !matched {
		t.Fatalf("password result = %T, want PasswordSecretAccepted", result)
	}
	return accepted.Value
}

func acceptedAccessTokenSecret(t *testing.T) AccessTokenSecret {
	t.Helper()
	result := NewAccessTokenSecret("01234567890123456789012345678901")
	accepted, matched := result.(AccessTokenSecretAccepted)
	if !matched {
		t.Fatalf("secret result = %T, want AccessTokenSecretAccepted", result)
	}
	return accepted.Value
}

func countJWTSeparators(token string) int {
	count := 0
	for _, char := range token {
		if char == '.' {
			count++
		}
	}
	return count
}

type memoryStore struct {
	credentialsByEmail map[string]CredentialRecord
	refreshByHash      map[string]RefreshTokenRecord
	consumedByHash     map[string]RefreshTokenRecord
	guestsByID         map[string]core.GuestID
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		credentialsByEmail: make(map[string]CredentialRecord),
		refreshByHash:      make(map[string]RefreshTokenRecord),
		consumedByHash:     make(map[string]RefreshTokenRecord),
		guestsByID:         make(map[string]core.GuestID),
	}
}

func (store *memoryStore) CreateUserCredential(_ context.Context, id core.UserID, email EmailAddress, passwordHash PasswordHash) StoreUserResult {
	if _, exists := store.credentialsByEmail[email.String()]; exists {
		return StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
	}

	store.credentialsByEmail[email.String()] = CredentialRecord{
		UserID:       id,
		Email:        email,
		PasswordHash: passwordHash,
	}
	return StoreUserAccepted{}
}

func (store *memoryStore) FindCredentialByEmail(_ context.Context, email EmailAddress) CredentialLookupResult {
	record, exists := store.credentialsByEmail[email.String()]
	if !exists {
		return CredentialMissing{}
	}

	return CredentialFound{Record: record}
}

func (store *memoryStore) CreateGuestSubject(_ context.Context, id core.GuestID) StoreGuestResult {
	store.guestsByID[id.String()] = id
	return StoreGuestAccepted{}
}

func (store *memoryStore) StoreRefreshToken(_ context.Context, record RefreshTokenRecord) StoreRefreshTokenResult {
	store.refreshByHash[record.Hash.String()] = record
	return StoreRefreshTokenAccepted{}
}

func (store *memoryStore) ConsumeRefreshToken(_ context.Context, hash RefreshTokenHash, consumedAt time.Time) ConsumeRefreshTokenResult {
	record, exists := store.refreshByHash[hash.String()]
	if !exists {
		return RefreshTokenNotConsumed{}
	}

	if !record.ExpiresAt.After(consumedAt) {
		return RefreshTokenNotConsumed{}
	}

	delete(store.refreshByHash, hash.String())
	store.consumedByHash[hash.String()] = record
	return RefreshTokenConsumed{Subject: record.Subject}
}
