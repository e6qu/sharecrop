package auth

import (
	"strings"
	"testing"
)

func TestNewDisplayNameTrimsAndBounds(t *testing.T) {
	accepted, matched := NewDisplayName("  Mara Ellison  ").(DisplayNameAccepted)
	if !matched {
		t.Fatalf("display name rejected")
	}
	if accepted.Value.String() != "Mara Ellison" {
		t.Fatalf("display name = %q, want trimmed", accepted.Value.String())
	}
	if _, rejected := NewDisplayName("   ").(DisplayNameRejected); !rejected {
		t.Fatalf("blank display name was accepted")
	}
	if _, rejected := NewDisplayName(strings.Repeat("x", 121)).(DisplayNameRejected); !rejected {
		t.Fatalf("over-long display name was accepted")
	}
	if _, ok := NewDisplayName(strings.Repeat("x", 120)).(DisplayNameAccepted); !ok {
		t.Fatalf("display name at the length bound was rejected")
	}
}

func TestDeriveDisplayNameFromEmailUsesLocalPart(t *testing.T) {
	email, matched := NewEmailAddress("jules.arden@example.com").(EmailAddressAccepted)
	if !matched {
		t.Fatalf("email rejected")
	}
	if got := DeriveDisplayNameFromEmail(email.Value).String(); got != "jules.arden" {
		t.Fatalf("derived name = %q, want jules.arden", got)
	}
}

func TestDeriveDisplayNameTruncatesLongLocalParts(t *testing.T) {
	long := strings.Repeat("a", 200)
	email, matched := NewEmailAddress(long + "@example.com").(EmailAddressAccepted)
	if !matched {
		t.Fatalf("email rejected")
	}
	derived := DeriveDisplayNameFromEmail(email.Value).String()
	if len(derived) != 120 {
		t.Fatalf("derived name length = %d, want 120", len(derived))
	}
	// The derived name must itself be a valid display name.
	if _, ok := NewDisplayName(derived).(DisplayNameAccepted); !ok {
		t.Fatalf("derived name failed its own constructor")
	}
}

func TestRegisterDerivesDisplayNameWhenNotProvided(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)
	email := acceptedEmail(t, "derive.me@example.com")

	registered, matched := service.Register(t.Context(), email, acceptedPassword(t, "correct horse battery staple"), DeriveDisplayName{}).(RegisterAccepted)
	if !matched {
		t.Fatalf("register rejected")
	}
	record := store.credentialsByEmail[email.String()]
	if record.UserID != registered.Subject.ID {
		t.Fatalf("stored record user mismatch")
	}
	if record.DisplayName.String() != "derive.me" {
		t.Fatalf("derived display name = %q, want derive.me", record.DisplayName.String())
	}
}

func TestRegisterUsesProvidedDisplayName(t *testing.T) {
	store := newMemoryStore()
	service := acceptedService(t, store)
	email := acceptedEmail(t, "provided@example.com")
	name, matched := NewDisplayName("Tala Reyes").(DisplayNameAccepted)
	if !matched {
		t.Fatalf("display name rejected")
	}

	if _, ok := service.Register(t.Context(), email, acceptedPassword(t, "correct horse battery staple"), ProvidedDisplayName{Value: name.Value}).(RegisterAccepted); !ok {
		t.Fatalf("register rejected")
	}
	if got := store.credentialsByEmail[email.String()].DisplayName.String(); got != "Tala Reyes" {
		t.Fatalf("stored display name = %q, want Tala Reyes", got)
	}
}
