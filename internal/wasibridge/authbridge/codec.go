// Package authbridge is the WASI bridge for internal/auth's Store, built the
// same way as auditbridge/notificationbridge: hand-written per-type codecs (this
// file) plus a generated dispatcher and guest client (bridge_gen.go). Shared
// core types (ids, page, time, plain strings) are serialized by
// internal/wasibridge/corewire; only auth-specific types live here.
//
// auth is the largest store and the one whose value types most needed care: the
// refresh-token and account-token hashes are opaque stored strings with no
// public "from plaintext" meaning, so they round-trip through the reconstruction
// constructors auth exposes for storage adapters (RefreshTokenHashFromString,
// AccountTokenHashFromString, AccountTokenKindFromString) - never by re-hashing.
package authbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- value types ----

func encodeEmail(email auth.EmailAddress) string { return email.String() }

func decodeEmail(raw string) (auth.EmailAddress, error) {
	accepted, matched := auth.NewEmailAddress(raw).(auth.EmailAddressAccepted)
	if !matched {
		return auth.EmailAddress{}, fmt.Errorf("invalid email address %q", raw)
	}
	return accepted.Value, nil
}

func encodePasswordHash(hash auth.PasswordHash) string { return hash.String() }

func decodePasswordHash(raw string) (auth.PasswordHash, error) {
	created, matched := auth.ParsePasswordHash(raw).(auth.PasswordHashCreated)
	if !matched {
		return auth.PasswordHash{}, fmt.Errorf("invalid password hash")
	}
	return created.Value, nil
}

// The hash/kind types are opaque stored strings; decode reconstructs from the
// stored form (no error is possible, but the signature matches the others so
// the generated dispatcher can call them uniformly).

func encodeRefreshTokenHash(hash auth.RefreshTokenHash) string { return hash.String() }

func decodeRefreshTokenHash(raw string) (auth.RefreshTokenHash, error) {
	return auth.RefreshTokenHashFromString(raw), nil
}

func encodeAccountTokenHash(hash auth.AccountTokenHash) string { return hash.String() }

func decodeAccountTokenHash(raw string) (auth.AccountTokenHash, error) {
	return auth.AccountTokenHashFromString(raw), nil
}

func encodeAccountTokenKind(kind auth.AccountTokenKind) string { return kind.String() }

func decodeAccountTokenKind(raw string) (auth.AccountTokenKind, error) {
	return auth.AccountTokenKindFromString(raw), nil
}

func encodeAccountTokenPlain(plain auth.AccountTokenPlain) string { return plain.String() }

func decodeAccountTokenPlain(raw string) (auth.AccountTokenPlain, error) {
	accepted, matched := auth.ParseAccountTokenPlain(raw).(auth.AccountTokenPlainAccepted)
	if !matched {
		return auth.AccountTokenPlain{}, fmt.Errorf("invalid account token")
	}
	return accepted.Value, nil
}

type externalIdentityWire struct {
	Issuer  string `json:"issuer"`
	Subject string `json:"subject"`
}

func encodeExternalIdentity(value auth.ExternalIdentity) externalIdentityWire {
	return externalIdentityWire{Issuer: value.Issuer, Subject: value.Subject}
}

func decodeExternalIdentity(value externalIdentityWire) (auth.ExternalIdentity, error) {
	if value.Issuer == "" || value.Subject == "" {
		return auth.ExternalIdentity{}, fmt.Errorf("external identity issuer and subject are required")
	}
	return auth.ExternalIdentity{Issuer: value.Issuer, Subject: value.Subject}, nil
}

// ---- auth.Subject (a three-variant union) ----

type subjectWire struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

func encodeSubject(subject auth.Subject) subjectWire {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return subjectWire{Kind: "user", ID: corewire.EncodeUserID(typed.ID)}
	case auth.GuestSubject:
		return subjectWire{Kind: "guest", ID: corewire.EncodeGuestID(typed.ID)}
	case auth.OrgSubject:
		return subjectWire{Kind: "org", ID: corewire.EncodeOrganizationID(typed.ID)}
	default:
		return subjectWire{Kind: "unknown"}
	}
}

func decodeSubject(wire subjectWire) (auth.Subject, error) {
	switch wire.Kind {
	case "user":
		id, err := corewire.DecodeUserID(wire.ID)
		if err != nil {
			return nil, err
		}
		return auth.UserSubject{ID: id}, nil
	case "guest":
		id, err := corewire.DecodeGuestID(wire.ID)
		if err != nil {
			return nil, err
		}
		return auth.GuestSubject{ID: id}, nil
	case "org":
		id, err := corewire.DecodeOrganizationID(wire.ID)
		if err != nil {
			return nil, err
		}
		return auth.OrgSubject{ID: id}, nil
	default:
		return nil, fmt.Errorf("unknown subject kind %q", wire.Kind)
	}
}

// ---- records ----

type credentialRecordWire struct {
	UserID       string `json:"user_id"`
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	Status       string `json:"status"`
}

type externalIdentityResultWire struct {
	Kind   string `json:"kind"`
	UserID string `json:"user_id,omitempty"`
	Error  string `json:"error,omitempty"`
}

func encodeExternalIdentityResult(result auth.ExternalIdentityResult) externalIdentityResultWire {
	switch typed := result.(type) {
	case auth.ExternalIdentityFound:
		return externalIdentityResultWire{Kind: "found", UserID: corewire.EncodeUserID(typed.UserID)}
	case auth.ExternalIdentityRejected:
		return externalIdentityResultWire{Kind: "rejected", Error: typed.Reason.Description()}
	default:
		return externalIdentityResultWire{Kind: "rejected", Error: "unknown external identity result"}
	}
}
func decodeExternalIdentityResult(wire externalIdentityResultWire) (auth.ExternalIdentityResult, error) {
	if wire.Kind == "found" {
		id, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return auth.ExternalIdentityRejected{Reason: guestError(err)}, err
		}
		return auth.ExternalIdentityFound{UserID: id}, nil
	}
	return auth.ExternalIdentityRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, wire.Error)}, nil
}

func encodeCredentialRecord(record auth.CredentialRecord) credentialRecordWire {
	return credentialRecordWire{
		UserID:       corewire.EncodeUserID(record.UserID),
		Email:        encodeEmail(record.Email),
		PasswordHash: encodePasswordHash(record.PasswordHash),
		Status:       record.Status,
	}
}

func decodeCredentialRecord(wire credentialRecordWire) (auth.CredentialRecord, error) {
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return auth.CredentialRecord{}, err
	}
	email, err := decodeEmail(wire.Email)
	if err != nil {
		return auth.CredentialRecord{}, err
	}
	passwordHash, err := decodePasswordHash(wire.PasswordHash)
	if err != nil {
		return auth.CredentialRecord{}, err
	}
	return auth.CredentialRecord{UserID: userID, Email: email, PasswordHash: passwordHash, Status: wire.Status}, nil
}

type userDirectoryEntryWire struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Status string `json:"status"`
}

func encodeUserDirectoryEntry(entry auth.UserDirectoryEntry) userDirectoryEntryWire {
	return userDirectoryEntryWire{ID: corewire.EncodeUserID(entry.ID), Email: encodeEmail(entry.Email), Status: entry.Status}
}

func decodeUserDirectoryEntry(wire userDirectoryEntryWire) (auth.UserDirectoryEntry, error) {
	id, err := corewire.DecodeUserID(wire.ID)
	if err != nil {
		return auth.UserDirectoryEntry{}, err
	}
	email, err := decodeEmail(wire.Email)
	if err != nil {
		return auth.UserDirectoryEntry{}, err
	}
	return auth.UserDirectoryEntry{ID: id, Email: email, Status: wire.Status}, nil
}

type refreshTokenRecordWire struct {
	ID        string      `json:"id"`
	FamilyID  string      `json:"family_id"`
	Subject   subjectWire `json:"subject"`
	Hash      string      `json:"hash"`
	ExpiresAt string      `json:"expires_at"`
}

func encodeRefreshTokenRecord(record auth.RefreshTokenRecord) refreshTokenRecordWire {
	return refreshTokenRecordWire{
		ID:        corewire.EncodeRefreshTokenID(record.ID),
		FamilyID:  corewire.EncodeRefreshTokenID(record.FamilyID),
		Subject:   encodeSubject(record.Subject),
		Hash:      encodeRefreshTokenHash(record.Hash),
		ExpiresAt: corewire.EncodeTime(record.ExpiresAt),
	}
}

func decodeRefreshTokenRecord(wire refreshTokenRecordWire) (auth.RefreshTokenRecord, error) {
	id, err := corewire.DecodeRefreshTokenID(wire.ID)
	if err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	familyID, err := corewire.DecodeRefreshTokenID(wire.FamilyID)
	if err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	subject, err := decodeSubject(wire.Subject)
	if err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	hash, err := decodeRefreshTokenHash(wire.Hash)
	if err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	expiresAt, err := corewire.DecodeTime(wire.ExpiresAt)
	if err != nil {
		return auth.RefreshTokenRecord{}, err
	}
	return auth.RefreshTokenRecord{ID: id, FamilyID: familyID, Subject: subject, Hash: hash, ExpiresAt: expiresAt}, nil
}

type accountTokenWire struct {
	ID        string `json:"id"`
	Plain     string `json:"plain"`
	Hash      string `json:"hash"`
	ExpiresAt string `json:"expires_at"`
}

func encodeAccountToken(token auth.AccountToken) accountTokenWire {
	return accountTokenWire{
		ID:        corewire.EncodeRefreshTokenID(token.ID),
		Plain:     encodeAccountTokenPlain(token.Plain),
		Hash:      encodeAccountTokenHash(token.Hash),
		ExpiresAt: corewire.EncodeTime(token.ExpiresAt),
	}
}

func decodeAccountToken(wire accountTokenWire) (auth.AccountToken, error) {
	id, err := corewire.DecodeRefreshTokenID(wire.ID)
	if err != nil {
		return auth.AccountToken{}, err
	}
	plain, err := decodeAccountTokenPlain(wire.Plain)
	if err != nil {
		return auth.AccountToken{}, err
	}
	hash, err := decodeAccountTokenHash(wire.Hash)
	if err != nil {
		return auth.AccountToken{}, err
	}
	expiresAt, err := corewire.DecodeTime(wire.ExpiresAt)
	if err != nil {
		return auth.AccountToken{}, err
	}
	return auth.AccountToken{ID: id, Plain: plain, Hash: hash, ExpiresAt: expiresAt}, nil
}

// ---- result unions ----
//
// acceptedRejectedWire backs every union that is just an accept/reject pair; the
// per-union encode/decode below name the concrete Go types.

type acceptedRejectedWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}

func encodeStoreUserResult(result auth.StoreUserResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.StoreUserAccepted:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.StoreUserRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeStoreUserResult(wire acceptedRejectedWire) (auth.StoreUserResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.StoreUserAccepted{}, nil
	case "rejected":
		return auth.StoreUserRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown store user result variant %q", wire.Variant)
	}
}

func encodeStoreGuestResult(result auth.StoreGuestResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.StoreGuestAccepted:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.StoreGuestRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeStoreGuestResult(wire acceptedRejectedWire) (auth.StoreGuestResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.StoreGuestAccepted{}, nil
	case "rejected":
		return auth.StoreGuestRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown store guest result variant %q", wire.Variant)
	}
}

func encodeStoreRefreshTokenResult(result auth.StoreRefreshTokenResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.StoreRefreshTokenAccepted:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.StoreRefreshTokenRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeStoreRefreshTokenResult(wire acceptedRejectedWire) (auth.StoreRefreshTokenResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.StoreRefreshTokenAccepted{}, nil
	case "rejected":
		return auth.StoreRefreshTokenRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown store refresh token result variant %q", wire.Variant)
	}
}

func encodeAccountTokenStoreResult(result auth.AccountTokenStoreResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.AccountTokenStored:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.AccountTokenStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeAccountTokenStoreResult(wire acceptedRejectedWire) (auth.AccountTokenStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.AccountTokenStored{}, nil
	case "rejected":
		return auth.AccountTokenStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown account token store result variant %q", wire.Variant)
	}
}

func encodeAccountMutationResult(result auth.AccountMutationResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.AccountMutationAccepted:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.AccountMutationRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeAccountMutationResult(wire acceptedRejectedWire) (auth.AccountMutationResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.AccountMutationAccepted{}, nil
	case "rejected":
		return auth.AccountMutationRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown account mutation result variant %q", wire.Variant)
	}
}

func encodeRevokeRefreshFamilyResult(result auth.RevokeRefreshFamilyResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case auth.RefreshFamilyRevoked:
		return acceptedRejectedWire{Variant: "accepted"}
	case auth.RevokeRefreshFamilyRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeRevokeRefreshFamilyResult(wire acceptedRejectedWire) (auth.RevokeRefreshFamilyResult, error) {
	switch wire.Variant {
	case "accepted":
		return auth.RefreshFamilyRevoked{}, nil
	case "rejected":
		return auth.RevokeRefreshFamilyRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown revoke refresh family result variant %q", wire.Variant)
	}
}

type credentialLookupResultWire struct {
	Variant string                  `json:"variant"`
	Record  *credentialRecordWire   `json:"record,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCredentialLookupResult(result auth.CredentialLookupResult) credentialLookupResultWire {
	switch typed := result.(type) {
	case auth.CredentialFound:
		record := encodeCredentialRecord(typed.Record)
		return credentialLookupResultWire{Variant: "found", Record: &record}
	case auth.CredentialMissing:
		return credentialLookupResultWire{Variant: "missing"}
	case auth.CredentialLookupRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialLookupResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialLookupResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeCredentialLookupResult(wire credentialLookupResultWire) (auth.CredentialLookupResult, error) {
	switch wire.Variant {
	case "found":
		if wire.Record == nil {
			return nil, fmt.Errorf("found credential result is missing its record")
		}
		record, err := decodeCredentialRecord(*wire.Record)
		if err != nil {
			return nil, err
		}
		return auth.CredentialFound{Record: record}, nil
	case "missing":
		return auth.CredentialMissing{}, nil
	case "rejected":
		return auth.CredentialLookupRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown credential lookup result variant %q", wire.Variant)
	}
}

type userDirectoryResultWire struct {
	Variant string                   `json:"variant"`
	Entries []userDirectoryEntryWire `json:"entries,omitempty"`
	Error   *domainwire.DomainError  `json:"error,omitempty"`
}

func encodeUserDirectoryResult(result auth.UserDirectoryResult) userDirectoryResultWire {
	switch typed := result.(type) {
	case auth.UsersListed:
		entries := make([]userDirectoryEntryWire, 0, len(typed.Values))
		for index := range typed.Values {
			entries = append(entries, encodeUserDirectoryEntry(typed.Values[index]))
		}
		return userDirectoryResultWire{Variant: "listed", Entries: entries}
	case auth.UserDirectoryRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return userDirectoryResultWire{Variant: "rejected", Error: &reason}
	default:
		return userDirectoryResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeUserDirectoryResult(wire userDirectoryResultWire) (auth.UserDirectoryResult, error) {
	switch wire.Variant {
	case "listed":
		entries := make([]auth.UserDirectoryEntry, 0, len(wire.Entries))
		for index := range wire.Entries {
			entry, err := decodeUserDirectoryEntry(wire.Entries[index])
			if err != nil {
				return nil, err
			}
			entries = append(entries, entry)
		}
		return auth.UsersListed{Values: entries}, nil
	case "rejected":
		return auth.UserDirectoryRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown user directory result variant %q", wire.Variant)
	}
}

type consumeRefreshTokenResultWire struct {
	Variant string                  `json:"variant"`
	Subject *subjectWire            `json:"subject,omitempty"`
	Family  string                  `json:"family,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeConsumeRefreshTokenResult(result auth.ConsumeRefreshTokenResult) consumeRefreshTokenResultWire {
	switch typed := result.(type) {
	case auth.RefreshTokenConsumed:
		subject := encodeSubject(typed.Subject)
		return consumeRefreshTokenResultWire{Variant: "consumed", Subject: &subject, Family: corewire.EncodeRefreshTokenID(typed.Family)}
	case auth.RefreshTokenNotConsumed:
		return consumeRefreshTokenResultWire{Variant: "not_consumed"}
	case auth.RefreshTokenReuseDetected:
		return consumeRefreshTokenResultWire{Variant: "reuse_detected"}
	case auth.ConsumeRefreshTokenRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return consumeRefreshTokenResultWire{Variant: "rejected", Error: &reason}
	default:
		return consumeRefreshTokenResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeConsumeRefreshTokenResult(wire consumeRefreshTokenResultWire) (auth.ConsumeRefreshTokenResult, error) {
	switch wire.Variant {
	case "consumed":
		if wire.Subject == nil {
			return nil, fmt.Errorf("consumed refresh token result is missing its subject")
		}
		subject, err := decodeSubject(*wire.Subject)
		if err != nil {
			return nil, err
		}
		family, err := corewire.DecodeRefreshTokenID(wire.Family)
		if err != nil {
			return nil, err
		}
		return auth.RefreshTokenConsumed{Subject: subject, Family: family}, nil
	case "not_consumed":
		return auth.RefreshTokenNotConsumed{}, nil
	case "reuse_detected":
		return auth.RefreshTokenReuseDetected{}, nil
	case "rejected":
		return auth.ConsumeRefreshTokenRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown consume refresh token result variant %q", wire.Variant)
	}
}

type accountTokenConsumeResultWire struct {
	Variant string                  `json:"variant"`
	UserID  string                  `json:"user_id,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeAccountTokenConsumeResult(result auth.AccountTokenConsumeResult) accountTokenConsumeResultWire {
	switch typed := result.(type) {
	case auth.AccountTokenConsumed:
		return accountTokenConsumeResultWire{Variant: "consumed", UserID: corewire.EncodeUserID(typed.UserID)}
	case auth.AccountTokenNotConsumed:
		return accountTokenConsumeResultWire{Variant: "not_consumed"}
	case auth.AccountTokenConsumeRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return accountTokenConsumeResultWire{Variant: "rejected", Error: &reason}
	default:
		return accountTokenConsumeResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown auth result %T", result))}
	}
}

func decodeAccountTokenConsumeResult(wire accountTokenConsumeResultWire) (auth.AccountTokenConsumeResult, error) {
	switch wire.Variant {
	case "consumed":
		userID, err := corewire.DecodeUserID(wire.UserID)
		if err != nil {
			return nil, err
		}
		return auth.AccountTokenConsumed{UserID: userID}, nil
	case "not_consumed":
		return auth.AccountTokenNotConsumed{}, nil
	case "rejected":
		return auth.AccountTokenConsumeRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown account token consume result variant %q", wire.Variant)
	}
}

// decodeReason rebuilds a DomainError from a rejected variant, tolerating a
// missing error payload (a malformed rejection) with an invalid_state stand-in.
func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "auth bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}
