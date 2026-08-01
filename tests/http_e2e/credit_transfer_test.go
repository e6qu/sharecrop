//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type creditTransferHTTPResponse struct {
	EntryID string `json:"entry_id"`
	Amount  int64  `json:"amount"`
}

func postCreditTransfer(t *testing.T, server *httptest.Server, accessToken string, body []byte) *http.Response {
	t.Helper()
	return postJSONWithBearer(t, server.URL+"/api/credits/transfers", body, accessToken)
}

func getOrganizationBalance(t *testing.T, server *httptest.Server, accessToken string, organizationID string) balanceHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/credits/balance", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body balanceHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode organization balance: %v", err)
	}
	return body
}

func decodeCreditTransferHTTPResponse(t *testing.T, response *http.Response) creditTransferHTTPResponse {
	t.Helper()
	var decoded creditTransferHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode credit transfer response: %v", err)
	}
	return decoded
}

func TestUserToUserCreditTransferWithReplayAndInsufficientFunds(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	sender := registerUser(t, server, "send-credits-sender")
	receiver := registerUser(t, server, "send-credits-receiver")

	senderBefore := getBalance(t, server, sender.AccessToken)
	receiverBefore := getBalance(t, server, receiver.AccessToken)

	sendBody := []byte(`{"source_kind":"self","target_kind":"user","target_id":"` + receiver.SubjectID + `","amount":25,"note":"thanks for the review","idempotency_key":"send-1-` + sender.SubjectID + `"}`)
	sendResponse := postCreditTransfer(t, server, sender.AccessToken, sendBody)
	defer sendResponse.Body.Close()
	assertStatus(t, sendResponse, http.StatusCreated)
	sent := decodeCreditTransferHTTPResponse(t, sendResponse)
	if sent.Amount != 25 || sent.EntryID == "" {
		t.Fatalf("credit transfer response = %+v, want amount 25 with an entry id", sent)
	}

	// A replayed idempotency key returns the original entry and moves nothing.
	replayResponse := postCreditTransfer(t, server, sender.AccessToken, sendBody)
	defer replayResponse.Body.Close()
	assertStatus(t, replayResponse, http.StatusCreated)
	replayed := decodeCreditTransferHTTPResponse(t, replayResponse)
	if replayed.EntryID != sent.EntryID || replayed.Amount != sent.Amount {
		t.Fatalf("replayed transfer = %+v, want the original %+v", replayed, sent)
	}

	senderAfter := getBalance(t, server, sender.AccessToken)
	receiverAfter := getBalance(t, server, receiver.AccessToken)
	if senderAfter.SpendableCredits != senderBefore.SpendableCredits-25 {
		t.Fatalf("sender spendable = %d, want %d", senderAfter.SpendableCredits, senderBefore.SpendableCredits-25)
	}
	if receiverAfter.SpendableCredits != receiverBefore.SpendableCredits+25 {
		t.Fatalf("receiver spendable = %d, want %d", receiverAfter.SpendableCredits, receiverBefore.SpendableCredits+25)
	}

	// Both sides see peer_transfer ledger rows.
	senderLedger := getLedger(t, server, sender.AccessToken)
	foundDebit := false
	for _, entry := range senderLedger.Entries {
		if entry.Kind == "peer_transfer" && entry.Amount == -25 {
			foundDebit = true
		}
	}
	if !foundDebit {
		t.Fatalf("sender ledger %+v has no peer_transfer debit of 25", senderLedger.Entries)
	}

	// The receiver is notified.
	if !hasNotificationKind(t, server, receiver.AccessToken, "credits_received") {
		t.Fatalf("receiver inbox has no credits_received notification")
	}

	// Self-sends are refused as invalid input.
	selfSend := postCreditTransfer(t, server, sender.AccessToken,
		[]byte(`{"source_kind":"self","target_kind":"user","target_id":"`+sender.SubjectID+`","amount":5,"idempotency_key":"self-`+sender.SubjectID+`"}`))
	defer selfSend.Body.Close()
	assertStatus(t, selfSend, http.StatusBadRequest)

	// Sending more than the spendable balance is refused.
	tooMuch := postCreditTransfer(t, server, sender.AccessToken,
		[]byte(`{"source_kind":"self","target_kind":"user","target_id":"`+receiver.SubjectID+`","amount":1000000,"idempotency_key":"too-much-`+sender.SubjectID+`"}`))
	defer tooMuch.Body.Close()
	assertStatus(t, tooMuch, http.StatusBadRequest)
	var insufficientBody errorHTTPResponse
	if err := json.NewDecoder(tooMuch.Body).Decode(&insufficientBody); err != nil {
		t.Fatalf("decode insufficient-funds error: %v", err)
	}
	if insufficientBody.Code != "invalid_argument" {
		t.Fatalf("insufficient-funds code = %q, want invalid_argument", insufficientBody.Code)
	}
}

func TestCreditTransfersBetweenUserAndOrganizationEnforceBillingPermission(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "org-send-owner")
	organizationID := createOrganization(t, server, owner, "Transfer Org")

	memberEmail := "org-send-member-" + uniqueTestSuffix(t) + "@example.com"
	member := registerUserWithEmail(t, server, memberEmail)
	provisionOrganizationMember(t, server, owner.AccessToken, organizationID, memberEmail, `["member"]`)

	orgBefore := getOrganizationBalance(t, server, owner.AccessToken, organizationID)

	// A user sends to the organization's account.
	toOrg := postCreditTransfer(t, server, member.AccessToken,
		[]byte(`{"source_kind":"self","target_kind":"organization","target_id":"`+organizationID+`","amount":40,"idempotency_key":"to-org-`+member.SubjectID+`"}`))
	defer toOrg.Body.Close()
	assertStatus(t, toOrg, http.StatusCreated)

	orgAfterReceive := getOrganizationBalance(t, server, owner.AccessToken, organizationID)
	if orgAfterReceive.SpendableCredits != orgBefore.SpendableCredits+40 {
		t.Fatalf("organization spendable = %d, want %d", orgAfterReceive.SpendableCredits, orgBefore.SpendableCredits+40)
	}

	// A plain member lacks the billing permission and cannot spend from the
	// organization account.
	denied := postCreditTransfer(t, server, member.AccessToken,
		[]byte(`{"source_kind":"organization","source_organization_id":"`+organizationID+`","target_kind":"user","target_id":"`+member.SubjectID+`","amount":10,"idempotency_key":"denied-`+member.SubjectID+`"}`))
	defer denied.Body.Close()
	assertStatus(t, denied, http.StatusForbidden)

	// The owner holds the billing permission and pays the member from the
	// organization balance.
	fromOrg := postCreditTransfer(t, server, owner.AccessToken,
		[]byte(`{"source_kind":"organization","source_organization_id":"`+organizationID+`","target_kind":"user","target_id":"`+member.SubjectID+`","amount":15,"idempotency_key":"from-org-`+organizationID+`"}`))
	defer fromOrg.Body.Close()
	assertStatus(t, fromOrg, http.StatusCreated)

	orgAfterSend := getOrganizationBalance(t, server, owner.AccessToken, organizationID)
	if orgAfterSend.SpendableCredits != orgAfterReceive.SpendableCredits-15 {
		t.Fatalf("organization spendable after send = %d, want %d", orgAfterSend.SpendableCredits, orgAfterReceive.SpendableCredits-15)
	}
	if !hasNotificationKind(t, server, member.AccessToken, "credits_received") {
		t.Fatalf("member inbox has no credits_received notification for the organization send")
	}

	// Organization-to-organization sends are refused as invalid input.
	orgToOrg := postCreditTransfer(t, server, owner.AccessToken,
		[]byte(`{"source_kind":"organization","source_organization_id":"`+organizationID+`","target_kind":"organization","target_id":"`+organizationID+`","amount":5,"idempotency_key":"org-org-`+organizationID+`"}`))
	defer orgToOrg.Body.Close()
	assertStatus(t, orgToOrg, http.StatusBadRequest)
}
