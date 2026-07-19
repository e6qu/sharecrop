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

func TestRefreshTokenReuseRevokesFamily(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)
	email := acceptedEmail(t, "reuse@example.com")
	password := acceptedPassword(t, "correct horse battery staple")

	registerResult := service.Register(context.Background(), email, password)
	registerAccepted, registerMatched := registerResult.(RegisterAccepted)
	if !registerMatched {
		t.Fatalf("register result = %T, want RegisterAccepted", registerResult)
	}

	// Rotate once: the original token is consumed and a new token is issued in
	// the same family.
	rotatedResult := service.Refresh(context.Background(), registerAccepted.RefreshToken)
	rotated, rotatedMatched := rotatedResult.(RefreshAccepted)
	if !rotatedMatched {
		t.Fatalf("rotation result = %T, want RefreshAccepted", rotatedResult)
	}

	// Reusing the already-consumed original token is detected and revokes the family.
	reuse := service.Refresh(context.Background(), registerAccepted.RefreshToken)
	if _, matched := reuse.(RefreshRejected); !matched {
		t.Fatalf("reuse result = %T, want RefreshRejected", reuse)
	}

	// The rotated token belongs to the revoked family, so it can no longer refresh.
	afterRevoke := service.Refresh(context.Background(), rotated.RefreshToken)
	if _, matched := afterRevoke.(RefreshRejected); !matched {
		t.Fatalf("post-revocation refresh = %T, want RefreshRejected", afterRevoke)
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

func TestServiceCreatesAndReusesExternalIdentitySession(t *testing.T) {
	service := acceptedService(t, newMemoryStore())
	email := acceptedEmail(t, "oidc@example.com")
	first := service.LoginExternal(context.Background(), "https://auth.dev.e6qu.dev/realms/dev", "sha-subject", email)
	firstAccepted, ok := first.(ExternalLoginAccepted)
	if !ok {
		t.Fatalf("first external login = %T, want ExternalLoginAccepted", first)
	}
	second := service.LoginExternal(context.Background(), "https://auth.dev.e6qu.dev/realms/dev", "sha-subject", email)
	secondAccepted, ok := second.(ExternalLoginAccepted)
	if !ok {
		t.Fatalf("second external login = %T, want ExternalLoginAccepted", second)
	}
	if secondAccepted.Subject.ID != firstAccepted.Subject.ID {
		t.Fatalf("external identity user = %q, want %q", secondAccepted.Subject.ID.String(), firstAccepted.Subject.ID.String())
	}
}

func TestServiceDoesNotLinkExternalIdentityToPasswordEmail(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)
	email := acceptedEmail(t, "local@example.com")
	if _, ok := service.Register(context.Background(), email, acceptedPassword(t, "correct horse battery staple")).(RegisterAccepted); !ok {
		t.Fatal("local registration failed")
	}
	result := service.LoginExternal(context.Background(), "https://auth.dev.e6qu.dev/realms/dev", "different-subject", email)
	if _, ok := result.(ExternalLoginRejected); !ok {
		t.Fatalf("external login = %T, want ExternalLoginRejected", result)
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
	externalByKey      map[string]core.UserID
	refreshByHash      map[string]RefreshTokenRecord
	consumedByHash     map[string]RefreshTokenRecord
	guestsByID         map[string]core.GuestID
	accountTokens      map[string]storedAccountToken
}

type storedAccountToken struct {
	userID     core.UserID
	kind       AccountTokenKind
	token      AccountToken
	consumed   bool
	consumedAt time.Time
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		credentialsByEmail: make(map[string]CredentialRecord),
		externalByKey:      make(map[string]core.UserID),
		refreshByHash:      make(map[string]RefreshTokenRecord),
		consumedByHash:     make(map[string]RefreshTokenRecord),
		guestsByID:         make(map[string]core.GuestID),
		accountTokens:      make(map[string]storedAccountToken),
	}
}

func (store *memoryStore) FindOrCreateExternalIdentity(_ context.Context, identity ExternalIdentity, email EmailAddress) ExternalIdentityResult {
	key := identity.Issuer + "\x00" + identity.Subject
	if id, ok := store.externalByKey[key]; ok {
		return ExternalIdentityFound{UserID: id}
	}
	if _, exists := store.credentialsByEmail[email.String()]; exists {
		return ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeConflict, "email address is already associated with another account")}
	}
	created := core.NewUserID()
	id, ok := created.(core.UserIDCreated)
	if !ok {
		return ExternalIdentityRejected{Reason: created.(core.UserIDRejected).Reason}
	}
	store.externalByKey[key] = id.Value
	return ExternalIdentityFound{UserID: id.Value}
}

func (store *memoryStore) CreateUserCredential(_ context.Context, id core.UserID, email EmailAddress, passwordHash PasswordHash) StoreUserResult {
	if _, exists := store.credentialsByEmail[email.String()]; exists {
		return StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
	}

	store.credentialsByEmail[email.String()] = CredentialRecord{
		UserID:       id,
		Email:        email,
		PasswordHash: passwordHash,
		Status:       "active",
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

func (store *memoryStore) FindCredentialByUserID(_ context.Context, id core.UserID) CredentialLookupResult {
	for _, record := range store.credentialsByEmail {
		if record.UserID.String() == id.String() {
			return CredentialFound{Record: record}
		}
	}
	return CredentialMissing{}
}

func (store *memoryStore) ListUsers(_ context.Context, _ string, _ core.Page) UserDirectoryResult {
	values := make([]UserDirectoryEntry, 0, len(store.credentialsByEmail))
	for _, record := range store.credentialsByEmail {
		if record.Status == "active" {
			values = append(values, UserDirectoryEntry{ID: record.UserID, Email: record.Email, Status: record.Status})
		}
	}
	return UsersListed{Values: values}
}

func (store *memoryStore) UpdateUserEmail(_ context.Context, id core.UserID, email EmailAddress) AccountMutationResult {
	for existingEmail, record := range store.credentialsByEmail {
		if record.UserID.String() == id.String() {
			if existing, exists := store.credentialsByEmail[email.String()]; exists && existing.UserID.String() != id.String() {
				return AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
			}
			delete(store.credentialsByEmail, existingEmail)
			record.Email = email
			store.credentialsByEmail[email.String()] = record
			return AccountMutationAccepted{}
		}
	}
	return AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account was not found")}
}

func (store *memoryStore) UpdatePassword(_ context.Context, id core.UserID, passwordHash PasswordHash) AccountMutationResult {
	for email, record := range store.credentialsByEmail {
		if record.UserID.String() == id.String() {
			record.PasswordHash = passwordHash
			store.credentialsByEmail[email] = record
			for hash, refresh := range store.refreshByHash {
				if subject, matched := refresh.Subject.(UserSubject); matched && subject.ID.String() == id.String() {
					delete(store.refreshByHash, hash)
				}
			}
			return AccountMutationAccepted{}
		}
	}
	return AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account was not found")}
}

func (store *memoryStore) DeactivateUser(_ context.Context, id core.UserID) AccountMutationResult {
	for email, record := range store.credentialsByEmail {
		if record.UserID.String() == id.String() {
			record.Status = "deactivated"
			store.credentialsByEmail[email] = record
			for hash, refresh := range store.refreshByHash {
				if subject, matched := refresh.Subject.(UserSubject); matched && subject.ID.String() == id.String() {
					delete(store.refreshByHash, hash)
				}
			}
			return AccountMutationAccepted{}
		}
	}
	return AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "account was not found")}
}

func (store *memoryStore) CreateGuestSubject(_ context.Context, id core.GuestID) StoreGuestResult {
	store.guestsByID[id.String()] = id
	return StoreGuestAccepted{}
}

func (store *memoryStore) StoreRefreshToken(_ context.Context, record RefreshTokenRecord) StoreRefreshTokenResult {
	store.refreshByHash[record.Hash.String()] = record
	return StoreRefreshTokenAccepted{}
}

func (store *memoryStore) StoreAccountToken(_ context.Context, id core.UserID, kind AccountTokenKind, token AccountToken) AccountTokenStoreResult {
	for hash, stored := range store.accountTokens {
		if stored.userID.String() == id.String() && stored.kind.String() == kind.String() && !stored.consumed {
			delete(store.accountTokens, hash)
		}
	}
	store.accountTokens[token.Hash.String()] = storedAccountToken{userID: id, kind: kind, token: token}
	return AccountTokenStored{}
}

func (store *memoryStore) ConsumeAccountToken(_ context.Context, kind AccountTokenKind, hash AccountTokenHash, consumedAt time.Time) AccountTokenConsumeResult {
	stored, exists := store.accountTokens[hash.String()]
	if !exists || stored.consumed || stored.kind.String() != kind.String() || !stored.token.ExpiresAt.After(consumedAt) {
		return AccountTokenNotConsumed{}
	}
	stored.consumed = true
	stored.consumedAt = consumedAt
	store.accountTokens[hash.String()] = stored
	return AccountTokenConsumed{UserID: stored.userID}
}

func (store *memoryStore) RevokeRefreshFamily(_ context.Context, hash RefreshTokenHash) RevokeRefreshFamilyResult {
	if record, exists := store.refreshByHash[hash.String()]; exists {
		for storedHash, active := range store.refreshByHash {
			if active.FamilyID.String() == record.FamilyID.String() {
				delete(store.refreshByHash, storedHash)
			}
		}
	}
	return RefreshFamilyRevoked{}
}

func (store *memoryStore) ConsumeRefreshToken(_ context.Context, hash RefreshTokenHash, consumedAt time.Time) ConsumeRefreshTokenResult {
	record, exists := store.refreshByHash[hash.String()]
	if !exists {
		if consumed, reused := store.consumedByHash[hash.String()]; reused {
			for storedHash, active := range store.refreshByHash {
				if active.FamilyID.String() == consumed.FamilyID.String() {
					delete(store.refreshByHash, storedHash)
				}
			}
			return RefreshTokenReuseDetected{}
		}
		return RefreshTokenNotConsumed{}
	}

	if !record.ExpiresAt.After(consumedAt) {
		return RefreshTokenNotConsumed{}
	}

	delete(store.refreshByHash, hash.String())
	store.consumedByHash[hash.String()] = record
	return RefreshTokenConsumed{Subject: record.Subject, Family: record.FamilyID}
}
