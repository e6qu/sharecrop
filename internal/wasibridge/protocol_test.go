package wasibridge

import (
	"bytes"
	"io"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

// These tests cover the bridge's serialization and guest-side transport without
// a WASM guest or Postgres — the integration test proves the same paths end to
// end, but the wire format is where fidelity can quietly break, so it is worth
// exercising directly and fast.

func sampleCredential(t *testing.T) auth.CredentialRecord {
	t.Helper()
	userID, ok := core.NewUserID().(core.UserIDCreated)
	if !ok {
		t.Fatalf("user id rejected")
	}
	email, ok := auth.NewEmailAddress("round-trip@example.com").(auth.EmailAddressAccepted)
	if !ok {
		t.Fatalf("email rejected")
	}
	secret, ok := auth.NewPasswordSecret("correct horse battery staple").(auth.PasswordSecretAccepted)
	if !ok {
		t.Fatalf("secret rejected")
	}
	hash, ok := auth.HashPassword(secret.Value).(auth.PasswordHashCreated)
	if !ok {
		t.Fatalf("hash rejected")
	}
	return auth.CredentialRecord{
		UserID:       userID.Value,
		Email:        email.Value,
		PasswordHash: hash.Value,
		Status:       "active",
	}
}

func TestCredentialWireRoundTripFound(t *testing.T) {
	record := sampleCredential(t)

	restored, ok := credentialFromWire(credentialToWire(auth.CredentialFound{Record: record})).(auth.CredentialFound)
	if !ok {
		t.Fatalf("restored result is not CredentialFound")
	}
	if restored.Record.UserID != record.UserID {
		t.Errorf("user id: got %s, want %s", restored.Record.UserID, record.UserID)
	}
	if restored.Record.Email.String() != record.Email.String() {
		t.Errorf("email: got %s, want %s", restored.Record.Email, record.Email)
	}
	if restored.Record.PasswordHash.String() != record.PasswordHash.String() {
		t.Errorf("password hash did not survive the round trip")
	}
	if restored.Record.Status != record.Status {
		t.Errorf("status: got %q, want %q", restored.Record.Status, record.Status)
	}
}

func TestCredentialWireRoundTripMissing(t *testing.T) {
	if _, ok := credentialFromWire(credentialToWire(auth.CredentialMissing{})).(auth.CredentialMissing); !ok {
		t.Fatalf("missing result did not survive the round trip")
	}
}

func TestCredentialWireRoundTripRejectedPreservesCode(t *testing.T) {
	codes := []core.ErrorCode{
		core.ErrorCodeInvalidArgument,
		core.ErrorCodeNotFound,
		core.ErrorCodePermissionDenied,
		core.ErrorCodeConflict,
		core.ErrorCodeInvalidState,
	}
	for _, code := range codes {
		original := auth.CredentialLookupRejected{Reason: core.NewDomainError(code, "boundary message")}
		restored, ok := credentialFromWire(credentialToWire(original)).(auth.CredentialLookupRejected)
		if !ok {
			t.Fatalf("code %s: restored result is not CredentialLookupRejected", code)
		}
		if restored.Reason.Code() != code {
			t.Errorf("code %s: restored as %s", code, restored.Reason.Code())
		}
		if restored.Reason.Description() != "boundary message" {
			t.Errorf("code %s: description lost, got %q", code, restored.Reason.Description())
		}
	}
}

func TestUnknownErrorCodeDegradesToInvalidState(t *testing.T) {
	result := credentialFromWire(credentialWire{Variant: "rejected", ErrorCode: "not_a_real_code", ErrorDescription: "x"})
	rejected, ok := result.(auth.CredentialLookupRejected)
	if !ok {
		t.Fatalf("result is not CredentialLookupRejected")
	}
	if rejected.Reason.Code() != core.ErrorCodeInvalidState {
		t.Errorf("unknown code degraded to %s, want %s", rejected.Reason.Code(), core.ErrorCodeInvalidState)
	}
}

func TestFrameRoundTrip(t *testing.T) {
	var buffer bytes.Buffer
	sent := guestFrame{Kind: "store_call", Call: &lookupRequest{Store: "auth", Method: "FindCredentialByEmail", Email: "a@b.com"}}
	if err := writeGuestFrame(&buffer, sent); err != nil {
		t.Fatalf("write frame: %v", err)
	}

	received, err := readGuestFrame(&buffer)
	if err != nil {
		t.Fatalf("read frame: %v", err)
	}
	if received.Kind != "store_call" || received.Call == nil || received.Call.Email != "a@b.com" {
		t.Fatalf("frame did not round-trip: %+v", received)
	}
}

func TestReadFrameCleanEOF(t *testing.T) {
	if _, err := readGuestFrame(bytes.NewReader(nil)); err != io.EOF {
		t.Fatalf("empty stream error = %v, want io.EOF", err)
	}
}

func TestReadFrameRejectsOversizedLength(t *testing.T) {
	// A length prefix past the cap must be refused before allocating.
	header := []byte{0xff, 0xff, 0xff, 0xff}
	if _, err := readGuestFrame(bytes.NewReader(header)); err == nil {
		t.Fatalf("oversized frame was accepted")
	}
}

// TestGuestLookupTransport drives the guest client against a stand-in host over
// pipes: the guest writes a store_call, the host replies with a rejected frame,
// and the guest must reconstruct it with the DomainError code intact.
func TestGuestLookupTransport(t *testing.T) {
	guestReqReader, guestReqWriter := io.Pipe()
	hostRespReader, hostRespWriter := io.Pipe()

	go func() {
		request, err := readGuestFrame(guestReqReader)
		if err != nil {
			return
		}
		if request.Call == nil || request.Call.Email != "worker@example.com" {
			return
		}
		reply := credentialWire{Variant: "rejected", ErrorCode: core.ErrorCodeNotFound.String(), ErrorDescription: "no such worker"}
		_ = writeHostFrame(hostRespWriter, hostFrame{Response: &reply})
	}()

	result := guestLookupCredential(guestReqWriter, hostRespReader, "worker@example.com")
	rejected, ok := result.(auth.CredentialLookupRejected)
	if !ok {
		t.Fatalf("result = %T, want CredentialLookupRejected", result)
	}
	if rejected.Reason.Code() != core.ErrorCodeNotFound {
		t.Errorf("code = %s, want not_found", rejected.Reason.Code())
	}
}
