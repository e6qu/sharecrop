package wasmdemo

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

// AuthBrowserStore implements auth.Store against BrowserStorage, so the real
// auth.Service (the same code cmd/sharecrop runs against Postgres, including
// real password hashing/verification and real refresh-token rotation with
// reuse detection) can serve the browser demo directly, instead of
// internal/wasmdemo's own simplified auth handling (which today accepts every
// password and never rotates refresh tokens).
type AuthBrowserStore struct {
	storage BrowserStorage
	ids     InteractionIDSource
}

func NewAuthBrowserStore(storage BrowserStorage, ids InteractionIDSource) AuthBrowserStore {
	return AuthBrowserStore{storage: storage, ids: ids}
}

type storedAuthUser struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	PasswordHash    string `json:"password_hash"`
	Status          string `json:"status"`
	EmailVerifiedAt int64  `json:"email_verified_at_unix,omitempty"`
}

type storedRefreshToken struct {
	ID          string `json:"id"`
	FamilyID    string `json:"family_id"`
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	Status      string `json:"status"`
	ExpiresAt   int64  `json:"expires_at_unix"`
}

type storedAccountToken struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Kind      string `json:"kind"`
	Status    string `json:"status"`
	ExpiresAt int64  `json:"expires_at_unix"`
}

func authUserKey(id string) string           { return "auth:user:" + id }
func authUserEmailKey(email string) string   { return "auth:user_email:" + strings.ToLower(email) }
func authUserIndexKey() string               { return "auth:user:index" }
func authGuestKey(id string) string          { return "auth:guest:" + id }
func authRefreshKey(hash string) string      { return "auth:refresh:" + hash }
func authAccountTokenKey(hash string) string { return "auth:account_token:" + hash }
func authAccountTokenActiveKey(userID string, kind string) string {
	return "auth:account_token_active:" + userID + ":" + kind
}

func putStoredAuthUserJSON(storage BrowserStorage, rawKey string, record storedAuthUser) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

// getStoredAuthUserJSON returns (record, found, ok) - found is false for a
// missing key (not an error); ok is false only for a genuine storage/
// decoding failure.
func getStoredAuthUserJSON(storage BrowserStorage, rawKey string) (storedAuthUser, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedAuthUser{}, found, ok
	}
	var record storedAuthUser
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedAuthUser{}, false, false
	}
	return record, true, true
}

func putStoredRefreshTokenJSON(storage BrowserStorage, rawKey string, record storedRefreshToken) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredRefreshTokenJSON(storage BrowserStorage, rawKey string) (storedRefreshToken, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedRefreshToken{}, found, ok
	}
	var record storedRefreshToken
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedRefreshToken{}, false, false
	}
	return record, true, true
}

func putStoredAccountTokenJSON(storage BrowserStorage, rawKey string, record storedAccountToken) bool {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false
	}
	return putStorageString(storage, rawKey, string(encoded))
}

func getStoredAccountTokenJSON(storage BrowserStorage, rawKey string) (storedAccountToken, bool, bool) {
	raw, found, ok := getStorageString(storage, rawKey)
	if !ok || !found {
		return storedAccountToken{}, found, ok
	}
	var record storedAccountToken
	if err := json.Unmarshal([]byte(raw), &record); err != nil {
		return storedAccountToken{}, false, false
	}
	return record, true, true
}

func invalidState(reason string) core.DomainError {
	return core.NewDomainError(core.ErrorCodeInvalidState, reason)
}

func (store AuthBrowserStore) CreateUserCredential(_ context.Context, id core.UserID, email auth.EmailAddress, passwordHash auth.PasswordHash) auth.StoreUserResult {
	emailKey := authUserEmailKey(email.String())
	_, found, ok := getStorageString(store.storage, emailKey)
	if !ok {
		return auth.StoreUserRejected{Reason: invalidState("user email lookup failed")}
	}
	if found {
		return auth.StoreUserRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "email address is already registered")}
	}

	record := storedAuthUser{ID: id.String(), Email: email.String(), PasswordHash: passwordHash.String(), Status: "active"}
	if !putStoredAuthUserJSON(store.storage, authUserKey(id.String()), record) {
		return auth.StoreUserRejected{Reason: invalidState("insert user failed")}
	}
	if !putStorageString(store.storage, emailKey, id.String()) {
		return auth.StoreUserRejected{Reason: invalidState("insert user email index failed")}
	}
	indexResult := appendStringIndex(store.storage, authUserIndexKey(), id.String(), "user")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return auth.StoreUserRejected{Reason: invalidState("update user index failed")}
	}

	if !store.insertSignupGrant("user", id.String()) {
		return auth.StoreUserRejected{Reason: invalidState("insert signup grant failed")}
	}

	return auth.StoreUserAccepted{}
}

// insertSignupGrant mirrors internal/db's insertSignupGrant, reusing the
// existing SaveLedgerEntry helper internal/wasmdemo already relies on for
// funding/balance - the same ledger storage the demo's existing task-funding
// code writes into, so a browser-store user's balance shows up consistently
// with everything else already using StoredLedgerEntry.
func (store AuthBrowserStore) insertSignupGrant(ownerKind string, ownerID string) bool {
	entryID := strings.TrimSpace(store.ids.NextLedgerEntryID())
	result := SaveLedgerEntry(store.storage, StoredLedgerEntry{
		ID:        entryID,
		OwnerKind: ownerKind,
		OwnerID:   ownerID,
		Kind:      "signup_grant",
		Amount:    ledger.SignupGrantAmount().Int64(),
	})
	_, matched := result.(LedgerEntryStored)
	return matched
}

func (store AuthBrowserStore) FindCredentialByEmail(_ context.Context, email auth.EmailAddress) auth.CredentialLookupResult {
	userID, found, ok := getStorageString(store.storage, authUserEmailKey(email.String()))
	if !ok {
		return auth.CredentialLookupRejected{Reason: invalidState("user email lookup failed")}
	}
	if !found {
		return auth.CredentialMissing{}
	}
	return store.findCredentialByUserID(userID)
}

func (store AuthBrowserStore) FindCredentialByUserID(_ context.Context, userID core.UserID) auth.CredentialLookupResult {
	return store.findCredentialByUserID(userID.String())
}

func (store AuthBrowserStore) findCredentialByUserID(rawUserID string) auth.CredentialLookupResult {
	record, found, ok := getStoredAuthUserJSON(store.storage, authUserKey(rawUserID))
	if !ok {
		return auth.CredentialLookupRejected{Reason: invalidState("user lookup failed")}
	}
	if !found {
		return auth.CredentialMissing{}
	}
	userResult := core.ParseUserID(record.ID)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return auth.CredentialLookupRejected{Reason: userResult.(core.UserIDRejected).Reason}
	}
	emailResult := auth.NewEmailAddress(record.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		return auth.CredentialLookupRejected{Reason: emailResult.(auth.EmailAddressRejected).Reason}
	}
	hashResult := auth.ParsePasswordHash(record.PasswordHash)
	hashCreated, hashMatched := hashResult.(auth.PasswordHashCreated)
	if !hashMatched {
		return auth.CredentialLookupRejected{Reason: hashResult.(auth.PasswordHashRejected).Reason}
	}
	return auth.CredentialFound{Record: auth.CredentialRecord{
		UserID:       userCreated.Value,
		Email:        emailAccepted.Value,
		PasswordHash: hashCreated.Value,
		Status:       record.Status,
	}}
}

func (store AuthBrowserStore) ListUsers(_ context.Context, query string, page core.Page) auth.UserDirectoryResult {
	indexResult := loadStringIndex(store.storage, authUserIndexKey(), "user")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return auth.UserDirectoryRejected{Reason: invalidState(indexResult.(stringIndexRejected).reason)}
	}

	cleanQuery := strings.ToLower(strings.TrimSpace(query))
	matching := make([]auth.UserDirectoryEntry, 0, len(loaded.values))
	for _, id := range loaded.values {
		record, found, ok := getStoredAuthUserJSON(store.storage, authUserKey(id))
		if !ok {
			return auth.UserDirectoryRejected{Reason: invalidState("user lookup failed")}
		}
		if !found || record.Status != "active" {
			continue
		}
		if cleanQuery != "" && !strings.Contains(strings.ToLower(record.Email), cleanQuery) && record.ID != query {
			continue
		}
		userResult := core.ParseUserID(record.ID)
		userCreated, userMatched := userResult.(core.UserIDCreated)
		if !userMatched {
			return auth.UserDirectoryRejected{Reason: userResult.(core.UserIDRejected).Reason}
		}
		emailResult := auth.NewEmailAddress(record.Email)
		emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
		if !emailMatched {
			return auth.UserDirectoryRejected{Reason: emailResult.(auth.EmailAddressRejected).Reason}
		}
		matching = append(matching, auth.UserDirectoryEntry{ID: userCreated.Value, Email: emailAccepted.Value, Status: record.Status})
	}

	sort.Slice(matching, func(i, j int) bool { return matching[i].Email.String() < matching[j].Email.String() })

	start := page.Offset()
	if start > len(matching) {
		start = len(matching)
	}
	end := start + page.Limit()
	if end > len(matching) {
		end = len(matching)
	}
	values := make([]auth.UserDirectoryEntry, end-start)
	copy(values, matching[start:end])
	return auth.UsersListed{Values: values}
}

func (store AuthBrowserStore) updateUser(userID string, mutate func(*storedAuthUser)) auth.AccountMutationResult {
	record, found, ok := getStoredAuthUserJSON(store.storage, authUserKey(userID))
	if !ok {
		return auth.AccountMutationRejected{Reason: invalidState("user lookup failed")}
	}
	if !found {
		return auth.AccountMutationRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "user was not found")}
	}
	mutate(&record)
	if !putStoredAuthUserJSON(store.storage, authUserKey(userID), record) {
		return auth.AccountMutationRejected{Reason: invalidState("update user failed")}
	}
	return auth.AccountMutationAccepted{}
}

func (store AuthBrowserStore) UpdateUserEmail(_ context.Context, userID core.UserID, email auth.EmailAddress) auth.AccountMutationResult {
	return store.updateUser(userID.String(), func(record *storedAuthUser) { record.Email = email.String() })
}

func (store AuthBrowserStore) UpdatePassword(_ context.Context, userID core.UserID, passwordHash auth.PasswordHash) auth.AccountMutationResult {
	return store.updateUser(userID.String(), func(record *storedAuthUser) { record.PasswordHash = passwordHash.String() })
}

func (store AuthBrowserStore) DeactivateUser(_ context.Context, userID core.UserID) auth.AccountMutationResult {
	return store.updateUser(userID.String(), func(record *storedAuthUser) { record.Status = "deactivated" })
}

func (store AuthBrowserStore) CreateGuestSubject(_ context.Context, id core.GuestID) auth.StoreGuestResult {
	if !putStorageString(store.storage, authGuestKey(id.String()), id.String()) {
		return auth.StoreGuestRejected{Reason: invalidState("insert guest subject failed")}
	}
	return auth.StoreGuestAccepted{}
}

func (store AuthBrowserStore) StoreRefreshToken(_ context.Context, record auth.RefreshTokenRecord) auth.StoreRefreshTokenResult {
	subjectKind := ""
	subjectID := ""
	switch subject := record.Subject.(type) {
	case auth.UserSubject:
		subjectKind = "user"
		subjectID = subject.ID.String()
	case auth.GuestSubject:
		subjectKind = "guest"
		subjectID = subject.ID.String()
	default:
		return auth.StoreRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject is invalid")}
	}
	stored := storedRefreshToken{
		ID:          record.ID.String(),
		FamilyID:    record.FamilyID.String(),
		SubjectKind: subjectKind,
		SubjectID:   subjectID,
		Status:      "active",
		ExpiresAt:   record.ExpiresAt.UnixNano(),
	}
	if !putStoredRefreshTokenJSON(store.storage, authRefreshKey(record.Hash.String()), stored) {
		return auth.StoreRefreshTokenRejected{Reason: invalidState("insert refresh token failed")}
	}
	indexResult := appendStringIndex(store.storage, authFamilyIndexKey(stored.FamilyID), record.Hash.String(), "refresh token family")
	if _, matched := indexResult.(stringIndexStored); !matched {
		return auth.StoreRefreshTokenRejected{Reason: invalidState("update refresh token family index failed")}
	}
	return auth.StoreRefreshTokenAccepted{}
}

func (store AuthBrowserStore) RevokeRefreshFamily(_ context.Context, hash auth.RefreshTokenHash) auth.RevokeRefreshFamilyResult {
	presented, found, ok := getStoredRefreshTokenJSON(store.storage, authRefreshKey(hash.String()))
	if !ok {
		return auth.RevokeRefreshFamilyRejected{Reason: invalidState("revoke refresh family failed")}
	}
	if !found {
		// An unknown token matches no family and revokes nothing - not an error.
		return auth.RefreshFamilyRevoked{}
	}
	if !store.revokeFamily(presented.FamilyID) {
		return auth.RevokeRefreshFamilyRejected{Reason: invalidState("revoke refresh family failed")}
	}
	return auth.RefreshFamilyRevoked{}
}

// revokeFamily marks every active token in the family as revoked, via the
// family-to-token-hashes index StoreRefreshToken maintains.
func (store AuthBrowserStore) revokeFamily(familyID string) bool {
	indexResult := loadStringIndex(store.storage, authFamilyIndexKey(familyID), "refresh token family")
	loaded, matched := indexResult.(stringIndexLoaded)
	if !matched {
		return false
	}
	for _, hash := range loaded.values {
		token, found, ok := getStoredRefreshTokenJSON(store.storage, authRefreshKey(hash))
		if !ok {
			return false
		}
		if !found || token.Status != "active" {
			continue
		}
		token.Status = "revoked"
		if !putStoredRefreshTokenJSON(store.storage, authRefreshKey(hash), token) {
			return false
		}
	}
	return true
}

func authFamilyIndexKey(familyID string) string { return "auth:refresh_family:" + familyID }

func (store AuthBrowserStore) ConsumeRefreshToken(_ context.Context, hash auth.RefreshTokenHash, consumedAt time.Time) auth.ConsumeRefreshTokenResult {
	token, found, ok := getStoredRefreshTokenJSON(store.storage, authRefreshKey(hash.String()))
	if !ok {
		return auth.ConsumeRefreshTokenRejected{Reason: invalidState("consume refresh token failed")}
	}
	if !found {
		return auth.RefreshTokenNotConsumed{}
	}

	if token.Status != "active" {
		// Reuse of an already-consumed/revoked token: revoke the whole family.
		if !store.revokeFamily(token.FamilyID) {
			return auth.ConsumeRefreshTokenRejected{Reason: invalidState("revoke refresh token family failed")}
		}
		return auth.RefreshTokenReuseDetected{}
	}

	if !time.Unix(0, token.ExpiresAt).After(consumedAt) {
		return auth.RefreshTokenNotConsumed{}
	}

	token.Status = "consumed"
	if !putStoredRefreshTokenJSON(store.storage, authRefreshKey(hash.String()), token) {
		return auth.ConsumeRefreshTokenRejected{Reason: invalidState("consume refresh token failed")}
	}

	familyResult := core.ParseRefreshTokenID(token.FamilyID)
	familyCreated, familyMatched := familyResult.(core.RefreshTokenIDCreated)
	if !familyMatched {
		return auth.ConsumeRefreshTokenRejected{Reason: familyResult.(core.RefreshTokenIDRejected).Reason}
	}

	switch token.SubjectKind {
	case "user":
		userResult := core.ParseUserID(token.SubjectID)
		userCreated, userMatched := userResult.(core.UserIDCreated)
		if !userMatched {
			return auth.ConsumeRefreshTokenRejected{Reason: userResult.(core.UserIDRejected).Reason}
		}
		return auth.RefreshTokenConsumed{Subject: auth.UserSubject{ID: userCreated.Value}, Family: familyCreated.Value}
	case "guest":
		guestResult := core.ParseGuestID(token.SubjectID)
		guestCreated, guestMatched := guestResult.(core.GuestIDCreated)
		if !guestMatched {
			return auth.ConsumeRefreshTokenRejected{Reason: guestResult.(core.GuestIDRejected).Reason}
		}
		return auth.RefreshTokenConsumed{Subject: auth.GuestSubject{ID: guestCreated.Value}, Family: familyCreated.Value}
	default:
		return auth.ConsumeRefreshTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "refresh token subject kind is invalid")}
	}
}

func (store AuthBrowserStore) StoreAccountToken(_ context.Context, userID core.UserID, kind auth.AccountTokenKind, token auth.AccountToken) auth.AccountTokenStoreResult {
	activeKey := authAccountTokenActiveKey(userID.String(), kind.String())
	previousHash, found, ok := getStorageString(store.storage, activeKey)
	if !ok {
		return auth.AccountTokenStoreRejected{Reason: invalidState("revoke previous account tokens failed")}
	}
	if found {
		previous, previousFound, previousOK := getStoredAccountTokenJSON(store.storage, authAccountTokenKey(previousHash))
		if !previousOK {
			return auth.AccountTokenStoreRejected{Reason: invalidState("revoke previous account tokens failed")}
		}
		if previousFound {
			previous.Status = "revoked"
			if !putStoredAccountTokenJSON(store.storage, authAccountTokenKey(previousHash), previous) {
				return auth.AccountTokenStoreRejected{Reason: invalidState("revoke previous account tokens failed")}
			}
		}
	}

	stored := storedAccountToken{
		ID:        token.ID.String(),
		UserID:    userID.String(),
		Kind:      kind.String(),
		Status:    "active",
		ExpiresAt: token.ExpiresAt.UnixNano(),
	}
	if !putStoredAccountTokenJSON(store.storage, authAccountTokenKey(token.Hash.String()), stored) {
		return auth.AccountTokenStoreRejected{Reason: invalidState("store account token failed")}
	}
	if !putStorageString(store.storage, activeKey, token.Hash.String()) {
		return auth.AccountTokenStoreRejected{Reason: invalidState("store account token failed")}
	}
	return auth.AccountTokenStored{}
}

func (store AuthBrowserStore) ConsumeAccountToken(_ context.Context, kind auth.AccountTokenKind, hash auth.AccountTokenHash, now time.Time) auth.AccountTokenConsumeResult {
	token, found, ok := getStoredAccountTokenJSON(store.storage, authAccountTokenKey(hash.String()))
	if !ok {
		return auth.AccountTokenConsumeRejected{Reason: invalidState("account token lookup failed")}
	}
	if !found || token.Status != "active" || token.Kind != kind.String() {
		return auth.AccountTokenNotConsumed{}
	}
	if !time.Unix(0, token.ExpiresAt).After(now) {
		return auth.AccountTokenNotConsumed{}
	}
	token.Status = "consumed"
	if !putStoredAccountTokenJSON(store.storage, authAccountTokenKey(hash.String()), token) {
		return auth.AccountTokenConsumeRejected{Reason: invalidState("consume account token failed")}
	}
	if kind == auth.AccountTokenKindEmailVerification {
		markResult := store.updateUser(token.UserID, func(record *storedAuthUser) { record.EmailVerifiedAt = now.UnixNano() })
		if _, matched := markResult.(auth.AccountMutationAccepted); !matched {
			return auth.AccountTokenConsumeRejected{Reason: invalidState("mark email verified failed")}
		}
	}
	userResult := core.ParseUserID(token.UserID)
	userCreated, userMatched := userResult.(core.UserIDCreated)
	if !userMatched {
		return auth.AccountTokenConsumeRejected{Reason: userResult.(core.UserIDRejected).Reason}
	}
	return auth.AccountTokenConsumed{UserID: userCreated.Value}
}
