//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupGrantCreatesBalanceAndLedgerEntry(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-signup")

	balance := getBalance(t, server, owner.AccessToken)
	if balance.Amount != 100 {
		t.Fatalf("signup balance = %d, want 100", balance.Amount)
	}

	ledgerBody := getLedger(t, server, owner.AccessToken)
	if len(ledgerBody.Entries) != 1 {
		t.Fatalf("ledger entry count = %d, want 1", len(ledgerBody.Entries))
	}
	if ledgerBody.Entries[0].Kind != "signup_grant" {
		t.Fatalf("entry kind = %q, want signup_grant", ledgerBody.Entries[0].Kind)
	}
	if ledgerBody.Entries[0].Amount != 100 {
		t.Fatalf("entry amount = %d, want 100", ledgerBody.Entries[0].Amount)
	}
}

func TestFundAcceptPayoutFlow(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-owner")
	worker := registerUser(t, server, "ledger-worker")

	task := createPublicCreditUserTask(t, server, owner, 40)

	escrow := fundTask(t, server, owner.AccessToken, task.ID, 40, "fund-"+task.ID)
	if escrow.State != "held" {
		t.Fatalf("escrow state = %q, want held", escrow.State)
	}
	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 60 {
		t.Fatalf("owner balance after funding = %d, want 60", balance.Amount)
	}

	openTask(t, server, owner.AccessToken, task.ID)

	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	other := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	accept := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "accept-"+submission.Submission.ID)
	if accept.PayoutKind != "credit" {
		t.Fatalf("payout kind = %q, want credit", accept.PayoutKind)
	}
	if accept.PayoutAmount != 40 {
		t.Fatalf("payout amount = %d, want 40", accept.PayoutAmount)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 140 {
		t.Fatalf("worker balance after payout = %d, want 140", balance.Amount)
	}
	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 60 {
		t.Fatalf("owner balance after payout = %d, want 60", balance.Amount)
	}

	// Re-accepting with the same idempotency key must not pay out twice.
	repeat := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "accept-"+submission.Submission.ID)
	if repeat.PayoutKind != "credit" {
		t.Fatalf("idempotent payout kind = %q, want credit", repeat.PayoutKind)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 140 {
		t.Fatalf("worker balance after idempotent accept = %d, want 140", balance.Amount)
	}

	// A second submission cannot become the accepted one.
	otherAccept := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+other.Submission.ID+"/accept", []byte(`{"idempotency_key":"accept-other"}`), owner.AccessToken)
	defer otherAccept.Body.Close()
	assertStatus(t, otherAccept, http.StatusConflict)
}

func TestRefundReturnsCredits(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-refund")
	task := createCreditUserTask(t, server, owner, 30)

	fundTask(t, server, owner.AccessToken, task.ID, 30, "fund-"+task.ID)
	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 70 {
		t.Fatalf("owner balance after funding = %d, want 70", balance.Amount)
	}

	refund := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/refund", []byte(`{"idempotency_key":"refund-`+task.ID+`"}`), owner.AccessToken)
	defer refund.Body.Close()
	assertStatus(t, refund, http.StatusOK)
	refundBody := decodeEscrowHTTPResponse(t, refund)
	if refundBody.State != "refunded" {
		t.Fatalf("refund state = %q, want refunded", refundBody.State)
	}

	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 100 {
		t.Fatalf("owner balance after refund = %d, want 100", balance.Amount)
	}
}

func TestFundRejectsInsufficientCredits(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-insufficient")
	task := createCreditUserTask(t, server, owner, 101)

	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/funding", []byte(`{"amount":101,"idempotency_key":"fund-too-much"}`), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusBadRequest)

	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 100 {
		t.Fatalf("owner balance after failed funding = %d, want 100", balance.Amount)
	}
}

func TestAcceptWithoutEscrowHasNoPayout(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-noreward")
	worker := registerUser(t, server, "ledger-noreward-worker")
	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	accept := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "accept-noreward")
	if accept.PayoutKind != "none" {
		t.Fatalf("payout kind = %q, want none", accept.PayoutKind)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 100 {
		t.Fatalf("worker balance after no-reward accept = %d, want 100", balance.Amount)
	}
}

type balanceHTTPResponse struct {
	Amount int64 `json:"amount"`
}

type ledgerEntryHTTPResponse struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	TaskID string `json:"task_id"`
}

type ledgerHTTPResponse struct {
	Entries []ledgerEntryHTTPResponse `json:"entries"`
}

type escrowHTTPResponse struct {
	TaskID string `json:"task_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
}

type acceptHTTPResponse struct {
	TaskID         string   `json:"task_id"`
	SubmissionID   string   `json:"submission_id"`
	PayoutKind     string   `json:"payout_kind"`
	PayoutAmount   int64    `json:"payout_amount"`
	WorkerUserID   string   `json:"worker_user_id"`
	CollectibleIDs []string `json:"collectible_ids"`
}

func registerUser(t *testing.T, server *httptest.Server, prefix string) authHTTPResponse {
	t.Helper()
	return registerUserWithEmail(t, server, prefix+"-"+uniqueTestSuffix(t)+"@example.com")
}

func createUserTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) taskHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(userTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response)
}

func createCreditUserTask(t *testing.T, server *httptest.Server, owner authHTTPResponse, amount int64) taskHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(userCreditTaskRequestJSON(owner.SubjectID, amount)), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response)
}

func openTask(t *testing.T, server *httptest.Server, accessToken string, taskID string) {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/open", []byte(`{}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
}

func submitAuthenticated(t *testing.T, server *httptest.Server, accessToken string, taskID string) submissionCreatedHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/submissions", []byte(`{"response_json":"{\"answer\":\"done\"}"}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeSubmissionCreatedHTTPResponse(t, response)
}

func fundTask(t *testing.T, server *httptest.Server, accessToken string, taskID string, amount int64, key string) escrowHTTPResponse {
	t.Helper()
	body, err := json.Marshal(fundingHTTPRequest{Amount: amount, IdempotencyKey: key})
	if err != nil {
		t.Fatalf("encode funding request: %v", err)
	}
	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/funding", body, accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeEscrowHTTPResponse(t, response)
}

func acceptSubmission(t *testing.T, server *httptest.Server, accessToken string, taskID string, submissionID string, key string) acceptHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/submissions/"+submissionID+"/accept", []byte(`{"idempotency_key":"`+key+`"}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body acceptHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode accept response: %v", err)
	}
	return body
}

func getBalance(t *testing.T, server *httptest.Server, accessToken string) balanceHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/credits/balance", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body balanceHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode balance response: %v", err)
	}
	return body
}

func getLedger(t *testing.T, server *httptest.Server, accessToken string) ledgerHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/credits/ledger", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body ledgerHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode ledger response: %v", err)
	}
	return body
}

func decodeEscrowHTTPResponse(t *testing.T, response *http.Response) escrowHTTPResponse {
	t.Helper()
	var body escrowHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode escrow response: %v", err)
	}
	return body
}

type fundingHTTPRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
}
