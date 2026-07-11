package authbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

func mustUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}

func mustRefreshTokenID(t *testing.T) core.RefreshTokenID {
	t.Helper()
	created, matched := core.NewRefreshTokenID().(core.RefreshTokenIDCreated)
	if !matched {
		t.Fatalf("refresh token id rejected")
	}
	return created.Value
}

func mustEmail(t *testing.T) auth.EmailAddress {
	t.Helper()
	accepted, matched := auth.NewEmailAddress("worker@example.com").(auth.EmailAddressAccepted)
	if !matched {
		t.Fatalf("email rejected")
	}
	return accepted.Value
}

func mustPasswordHash(t *testing.T) auth.PasswordHash {
	t.Helper()
	secret, matched := auth.NewPasswordSecret("correct horse battery staple").(auth.PasswordSecretAccepted)
	if !matched {
		t.Fatalf("secret rejected")
	}
	hash, matched := auth.HashPassword(secret.Value).(auth.PasswordHashCreated)
	if !matched {
		t.Fatalf("hash rejected")
	}
	return hash.Value
}

func TestSubjectRoundTrip(t *testing.T) {
	guestID, matched := core.NewGuestID().(core.GuestIDCreated)
	if !matched {
		t.Fatalf("guest id rejected")
	}
	orgID, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("org id rejected")
	}
	cases := []auth.Subject{
		auth.UserSubject{ID: mustUserID(t)},
		auth.GuestSubject{ID: guestID.Value},
		auth.OrgSubject{ID: orgID.Value},
	}
	for _, subject := range cases {
		restored, err := decodeSubject(encodeSubject(subject))
		if err != nil {
			t.Fatalf("decode subject %T: %v", subject, err)
		}
		if restored != subject {
			t.Errorf("subject %T did not round-trip: got %v", subject, restored)
		}
	}
	if _, err := decodeSubject(subjectWire{Kind: "bogus"}); err == nil {
		t.Errorf("decodeSubject accepted a bogus kind")
	}
}

func TestCredentialLookupResultRoundTrip(t *testing.T) {
	record := auth.CredentialRecord{UserID: mustUserID(t), Email: mustEmail(t), PasswordHash: mustPasswordHash(t), Status: "active"}

	found, err := decodeCredentialLookupResult(encodeCredentialLookupResult(auth.CredentialFound{Record: record}))
	if err != nil {
		t.Fatalf("decode found: %v", err)
	}
	typed, matched := found.(auth.CredentialFound)
	if !matched {
		t.Fatalf("found result = %T", found)
	}
	if typed.Record.UserID != record.UserID || typed.Record.Email.String() != record.Email.String() ||
		typed.Record.PasswordHash.String() != record.PasswordHash.String() || typed.Record.Status != record.Status {
		t.Errorf("credential record did not round-trip: %+v", typed.Record)
	}

	missing, err := decodeCredentialLookupResult(encodeCredentialLookupResult(auth.CredentialMissing{}))
	if err != nil || missing == nil {
		t.Fatalf("decode missing: %v", err)
	}
	if _, matched := missing.(auth.CredentialMissing); !matched {
		t.Errorf("missing result = %T", missing)
	}

	rejected, err := decodeCredentialLookupResult(encodeCredentialLookupResult(auth.CredentialLookupRejected{
		Reason: core.NewDomainError(core.ErrorCodeConflict, "dup"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(auth.CredentialLookupRejected); !matched || typed.Reason.Code() != core.ErrorCodeConflict {
		t.Errorf("rejection not preserved: %T", rejected)
	}
}

func TestRefreshTokenRecordRoundTrip(t *testing.T) {
	record := auth.RefreshTokenRecord{
		ID:        mustRefreshTokenID(t),
		FamilyID:  mustRefreshTokenID(t),
		Subject:   auth.UserSubject{ID: mustUserID(t)},
		Hash:      auth.RefreshTokenHashFromString("stored-refresh-hash"),
		ExpiresAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
	}
	restored, err := decodeRefreshTokenRecord(encodeRefreshTokenRecord(record))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.ID != record.ID || restored.FamilyID != record.FamilyID || restored.Subject != record.Subject ||
		restored.Hash.String() != record.Hash.String() || !restored.ExpiresAt.Equal(record.ExpiresAt) {
		t.Errorf("refresh token record did not round-trip: %+v", restored)
	}
}

func TestAccountTokenRoundTrip(t *testing.T) {
	plain, matched := auth.ParseAccountTokenPlain("plain-token-value").(auth.AccountTokenPlainAccepted)
	if !matched {
		t.Fatalf("plain rejected")
	}
	token := auth.AccountToken{
		ID:        mustRefreshTokenID(t),
		Plain:     plain.Value,
		Hash:      auth.AccountTokenHashFromString("stored-account-hash"),
		ExpiresAt: time.Date(2026, 7, 11, 12, 0, 0, 0, time.UTC),
	}
	restored, err := decodeAccountToken(encodeAccountToken(token))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if restored.ID != token.ID || restored.Plain.String() != token.Plain.String() ||
		restored.Hash.String() != token.Hash.String() || !restored.ExpiresAt.Equal(token.ExpiresAt) {
		t.Errorf("account token did not round-trip: %+v", restored)
	}
}

func TestConsumeRefreshTokenResultRoundTrip(t *testing.T) {
	family := mustRefreshTokenID(t)
	consumed, err := decodeConsumeRefreshTokenResult(encodeConsumeRefreshTokenResult(auth.RefreshTokenConsumed{
		Subject: auth.UserSubject{ID: mustUserID(t)},
		Family:  family,
	}))
	if err != nil {
		t.Fatalf("decode consumed: %v", err)
	}
	typed, matched := consumed.(auth.RefreshTokenConsumed)
	if !matched || typed.Family != family {
		t.Errorf("consumed did not round-trip: %T", consumed)
	}

	for _, variant := range []auth.ConsumeRefreshTokenResult{auth.RefreshTokenNotConsumed{}, auth.RefreshTokenReuseDetected{}} {
		restored, err := decodeConsumeRefreshTokenResult(encodeConsumeRefreshTokenResult(variant))
		if err != nil {
			t.Fatalf("decode %T: %v", variant, err)
		}
		if restored != variant {
			t.Errorf("%T did not round-trip: got %T", variant, restored)
		}
	}
}

func TestAcceptedRejectedRoundTrip(t *testing.T) {
	accepted, err := decodeStoreUserResult(encodeStoreUserResult(auth.StoreUserAccepted{}))
	if err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if _, matched := accepted.(auth.StoreUserAccepted); !matched {
		t.Errorf("accepted result = %T", accepted)
	}
	rejected, err := decodeAccountMutationResult(encodeAccountMutationResult(auth.AccountMutationRejected{
		Reason: core.NewDomainError(core.ErrorCodeNotFound, "missing"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(auth.AccountMutationRejected); !matched || typed.Reason.Code() != core.ErrorCodeNotFound {
		t.Errorf("account mutation rejection not preserved: %T", rejected)
	}
}
