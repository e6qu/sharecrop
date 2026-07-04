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

func TestParseTaskSeriesIDRoundTrips(t *testing.T) {
	created, matched := NewTaskSeriesID().(TaskSeriesIDCreated)
	if !matched {
		t.Fatalf("new task series id did not create")
	}

	parsed, matched := ParseTaskSeriesID(created.Value.String()).(TaskSeriesIDCreated)
	if !matched {
		t.Fatalf("parse task series id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseSubmissionIDRoundTrips(t *testing.T) {
	created, matched := NewSubmissionID().(SubmissionIDCreated)
	if !matched {
		t.Fatalf("new submission id did not create")
	}

	parsed, matched := ParseSubmissionID(created.Value.String()).(SubmissionIDCreated)
	if !matched {
		t.Fatalf("parse submission id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseSubmissionReceiptTokenIDRoundTrips(t *testing.T) {
	created, matched := NewSubmissionReceiptTokenID().(SubmissionReceiptTokenIDCreated)
	if !matched {
		t.Fatalf("new submission receipt token id did not create")
	}

	parsed, matched := ParseSubmissionReceiptTokenID(created.Value.String()).(SubmissionReceiptTokenIDCreated)
	if !matched {
		t.Fatalf("parse submission receipt token id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
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

func TestParseAgentCredentialIDRoundTrips(t *testing.T) {
	created, matched := NewAgentCredentialID().(AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("new agent credential id did not create")
	}

	parsed, matched := ParseAgentCredentialID(created.Value.String()).(AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("parse agent credential id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseCreditAccountIDRoundTrips(t *testing.T) {
	created, matched := NewCreditAccountID().(CreditAccountIDCreated)
	if !matched {
		t.Fatalf("new credit account id did not create")
	}

	parsed, matched := ParseCreditAccountID(created.Value.String()).(CreditAccountIDCreated)
	if !matched {
		t.Fatalf("parse credit account id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseLedgerEntryIDRoundTrips(t *testing.T) {
	created, matched := NewLedgerEntryID().(LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("new ledger entry id did not create")
	}

	parsed, matched := ParseLedgerEntryID(created.Value.String()).(LedgerEntryIDCreated)
	if !matched {
		t.Fatalf("parse ledger entry id did not create")
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

func TestParseTeamIDRoundTrips(t *testing.T) {
	created, matched := NewTeamID().(TeamIDCreated)
	if !matched {
		t.Fatalf("new team id did not create")
	}

	parsed, matched := ParseTeamID(created.Value.String()).(TeamIDCreated)
	if !matched {
		t.Fatalf("parse team id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}

func TestParseOrganizationMembershipIDRoundTrips(t *testing.T) {
	created, matched := NewOrganizationMembershipID().(OrganizationMembershipIDCreated)
	if !matched {
		t.Fatalf("new organization membership id did not create")
	}

	parsed, matched := ParseOrganizationMembershipID(created.Value.String()).(OrganizationMembershipIDCreated)
	if !matched {
		t.Fatalf("parse organization membership id did not create")
	}

	if parsed.Value.String() != created.Value.String() {
		t.Fatalf("parsed = %q, want %q", parsed.Value.String(), created.Value.String())
	}
}
