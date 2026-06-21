package core

import "testing"

func TestNewTaskIDCreatesUUID(t *testing.T) {
	result := NewTaskID()

	created, matched := result.(TaskIDCreated)
	if !matched {
		t.Fatalf("result = %T, want TaskIDCreated", result)
	}

	if created.Value.String() == "" {
		t.Fatalf("task id string was empty")
	}
}

func TestParseTaskIDRejectsInvalidInput(t *testing.T) {
	result := ParseTaskID("not-a-uuid")

	rejected, matched := result.(TaskIDRejected)
	if !matched {
		t.Fatalf("result = %T, want TaskIDRejected", result)
	}

	if rejected.Reason.Code().String() != ErrorCodeInvalidID.String() {
		t.Fatalf("code = %q, want %q", rejected.Reason.Code().String(), ErrorCodeInvalidID.String())
	}
}

func TestParseUserIDRoundTrips(t *testing.T) {
	created, matched := NewUserID().(UserIDCreated)
	if !matched {
		t.Fatalf("new user id did not create")
	}

	parsed, matched := ParseUserID(created.Value.String()).(UserIDCreated)
	if !matched {
		t.Fatalf("parse user id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseOrganizationIDRoundTrips(t *testing.T) {
	created, matched := NewOrganizationID().(OrganizationIDCreated)
	if !matched {
		t.Fatalf("new organization id did not create")
	}

	parsed, matched := ParseOrganizationID(created.Value.String()).(OrganizationIDCreated)
	if !matched {
		t.Fatalf("parse organization id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseGuestIDRoundTrips(t *testing.T) {
	created, matched := NewGuestID().(GuestIDCreated)
	if !matched {
		t.Fatalf("new guest id did not create")
	}

	parsed, matched := ParseGuestID(created.Value.String()).(GuestIDCreated)
	if !matched {
		t.Fatalf("parse guest id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseRefreshTokenIDRoundTrips(t *testing.T) {
	created, matched := NewRefreshTokenID().(RefreshTokenIDCreated)
	if !matched {
		t.Fatalf("new refresh token id did not create")
	}

	parsed, matched := ParseRefreshTokenID(created.Value.String()).(RefreshTokenIDCreated)
	if !matched {
		t.Fatalf("parse refresh token id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}
