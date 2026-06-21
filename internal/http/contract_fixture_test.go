package httpserver

import (
	"encoding/json"
	"testing"
)

// These fixtures pin the wire JSON shape of API responses so accidental
// field renames or shape changes are caught before they reach clients and
// the generated Elm contracts drift.

func TestAuthResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(authResponse{SubjectKind: "user", SubjectID: "subject-1", AccessToken: "token-1"})
	assertWireShape(t, encoded, err, `{"subject_kind":"user","subject_id":"subject-1","access_token":"token-1"}`)
}

func TestBalanceResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(balanceResponse{Amount: 100})
	assertWireShape(t, encoded, err, `{"amount":100}`)
}

func TestLedgerEntryResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(ledgerEntryResponse{ID: "entry-1", Kind: "signup_grant", Amount: 100, TaskID: ""})
	assertWireShape(t, encoded, err, `{"id":"entry-1","kind":"signup_grant","amount":100,"task_id":""}`)
}

func TestTaskEscrowResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskEscrowResponse{TaskID: "task-1", Amount: 40, State: "held"})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","amount":40,"state":"held"}`)
}

func TestSubmissionValidationErrorResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionValidationErrorResponse{Path: "email", Message: "is required"})
	assertWireShape(t, encoded, err, `{"path":"email","message":"is required"}`)
}

func TestAgentCredentialResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialResponse{ID: "cred-1", Label: "Local agent", Scopes: []string{"tasks_read", "submissions_write"}, State: "active"})
	assertWireShape(t, encoded, err, `{"id":"cred-1","label":"Local agent","scopes":["tasks_read","submissions_write"],"state":"active"}`)
}

func assertWireShape(t *testing.T, got []byte, err error, want string) {
	t.Helper()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if string(got) != want {
		t.Fatalf("wire shape =\n  %s\nwant\n  %s", string(got), want)
	}
}
