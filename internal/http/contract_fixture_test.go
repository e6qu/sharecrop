package httpserver

import (
	"encoding/json"
	"testing"
)

// These fixtures pin the wire JSON shape of API responses so accidental
// field renames or shape changes are caught before they reach clients and
// the generated Elm contracts drift.

func TestAuthResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(authResponse{SubjectKind: "user", SubjectID: "subject-1", AccessToken: "token-1", Role: "member"})
	assertWireShape(t, encoded, err, `{"subject_kind":"user","subject_id":"subject-1","access_token":"token-1","role":"member"}`)
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

func TestAcceptSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(acceptSubmissionResponse{TaskID: "task-1", SubmissionID: "submission-1", PayoutKind: "bundle", PayoutAmount: 25, WorkerUserID: "user-1", CollectibleIDs: []string{"collectible-1", "collectible-2"}, TipAmount: 5})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","submission_id":"submission-1","payout_kind":"bundle","payout_amount":25,"worker_user_id":"user-1","collectible_ids":["collectible-1","collectible-2"],"tip_amount":5}`)
}

func TestReviewSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(reviewSubmissionResponse{TaskID: "task-1", SubmissionID: "submission-1", State: "rejected", ReviewNote: "Not current.", PayoutKind: "credit", PayoutAmount: 10, WorkerUserID: "user-1", TipAmount: 2})
	assertWireShape(t, encoded, err, `{"task_id":"task-1","submission_id":"submission-1","state":"rejected","review_note":"Not current.","payout_kind":"credit","payout_amount":10,"worker_user_id":"user-1","tip_amount":2}`)
}

func TestReservationResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(reservationResponse{ID: "reservation-1", TaskID: "task-1", AssigneeKind: "user", AssigneeID: "user-1", State: "active", RequestedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"reservation-1","task_id":"task-1","assignee_kind":"user","assignee_id":"user-1","state":"active","requested_by":"user-1"}`)
}

func TestTeamResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(teamResponse{ID: "team-1", OwnerKind: "user", OrganizationID: "", OwnerUserID: "user-1", Name: "Survey crew", CreatedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"team-1","owner_kind":"user","organization_id":"","owner_user_id":"user-1","name":"Survey crew","created_by":"user-1"}`)
}

func TestOrganizationResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationResponse{ID: "org-1", Name: "Lattice Field Co", CreatedBy: "user-1"})
	assertWireShape(t, encoded, err, `{"id":"org-1","name":"Lattice Field Co","created_by":"user-1"}`)
}

func TestOrganizationMemberResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(organizationMemberResponse{ID: "member-1", OrganizationID: "org-1", UserID: "user-1", Status: "active", Roles: []string{"owner"}})
	assertWireShape(t, encoded, err, `{"id":"member-1","organization_id":"org-1","user_id":"user-1","status":"active","roles":["owner"]}`)
}

func TestTaskCapabilityTokenResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskCapabilityTokenResponse{ID: "cap-1", TaskID: "task-1", State: "active", Token: "secret-token"})
	assertWireShape(t, encoded, err, `{"id":"cap-1","task_id":"task-1","state":"active","token":"secret-token"}`)
}

func TestSubmissionCreatedResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionCreatedResponse{Submission: submissionResponse{ID: "submission-1", TaskID: "task-1", SubmitterID: "user-1", State: "submitted", ResponseJSON: "{}", ReviewNote: "", ValidationErrors: []submissionValidationErrorResponse{}}, ReceiptToken: "receipt-1"})
	assertWireShape(t, encoded, err, `{"submission":{"id":"submission-1","task_id":"task-1","submitter_id":"user-1","state":"submitted","response_json":"{}","review_note":"","validation_errors":[]},"receipt_token":"receipt-1"}`)
}

func TestSubmissionValidationErrorResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionValidationErrorResponse{Path: "email", Message: "is required"})
	assertWireShape(t, encoded, err, `{"path":"email","message":"is required"}`)
}

func TestSubmissionResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(submissionResponse{ID: "submission-1", TaskID: "task-1", SubmitterID: "user-1", State: "changes_requested", ResponseJSON: "{}", ReviewNote: "Use the current API.", ValidationErrors: []submissionValidationErrorResponse{}})
	assertWireShape(t, encoded, err, `{"id":"submission-1","task_id":"task-1","submitter_id":"user-1","state":"changes_requested","response_json":"{}","review_note":"Use the current API.","validation_errors":[]}`)
}

func TestAgentCredentialResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(agentCredentialResponse{ID: "cred-1", Label: "Local agent", Scopes: []string{"tasks_read", "submissions_write"}, State: "active"})
	assertWireShape(t, encoded, err, `{"id":"cred-1","label":"Local agent","scopes":["tasks_read","submissions_write"],"state":"active"}`)
}

func TestTaskListItemResponseWireShape(t *testing.T) {
	encoded, err := json.Marshal(taskListItemResponse{
		ID:                     "task-1",
		OwnerKind:              "user",
		Title:                  "Label receipts",
		RewardKind:             "credit",
		RewardCreditAmount:     25,
		RewardCollectibleCount: 0,
		ParticipationPolicy:    "reservation_required",
		AssigneeScope:          "user",
		ReservationExpiryHours: 48,
		State:                  "open",
		VisibilityKind:         "public",
		AvailabilityKind:       "reserved",
		ViewerAction:           "wait",
		CreatedBy:              "user-1",
		ActiveAssigneeKind:     "user",
		ActiveAssigneeID:       "user-2",
	})
	assertWireShape(t, encoded, err, `{"id":"task-1","owner_kind":"user","title":"Label receipts","reward_kind":"credit","reward_credit_amount":25,"reward_collectible_count":0,"participation_policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48,"state":"open","visibility_kind":"public","availability_kind":"reserved","viewer_action":"wait","created_by":"user-1","active_assignee_kind":"user","active_assignee_id":"user-2"}`)
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
