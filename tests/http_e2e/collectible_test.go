//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMultipleCollectiblesAwardedOnAcceptHTTP(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "multi-collectible-owner")
	worker := registerUser(t, server, "multi-collectible-worker")

	firstID := mintCollectible(t, server, owner.AccessToken, "First medal")
	secondID := mintCollectible(t, server, owner.AccessToken, "Second medal")

	task := createPublicUserTask(t, server, owner)
	for _, collectibleID := range []string{firstID, secondID} {
		fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectibleID+`"}`), owner.AccessToken)
		assertStatus(t, fundResponse, http.StatusCreated)
		fundResponse.Body.Close()
	}

	openTask(t, server, owner.AccessToken, task.ID)
	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	accept := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "multi-collectible-accept-"+task.ID)
	if len(accept.CollectibleIDs) != 2 {
		t.Fatalf("payout collectible count = %d, want 2", len(accept.CollectibleIDs))
	}

	workerCollectibles := listCollectibles(t, server, worker.AccessToken)
	if len(workerCollectibles) != 2 {
		t.Fatalf("worker collectible count = %d, want 2", len(workerCollectibles))
	}
	if owned := listCollectibles(t, server, owner.AccessToken); len(owned) != 0 {
		t.Fatalf("owner collectible count after award = %d, want 0", len(owned))
	}
}

func TestCollectibleRewardAwardedOnAccept(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "collectible-owner")
	worker := registerUser(t, server, "collectible-worker")

	collectibleID := mintCollectible(t, server, owner.AccessToken, "Golden harvest badge")
	if collectibles := listCollectibles(t, server, owner.AccessToken); len(collectibles) != 1 {
		t.Fatalf("owner collectible count = %d, want 1", len(collectibles))
	}

	task := createPublicUserTask(t, server, owner)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectibleID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)
	escrowed := decodeCollectibleHTTPResponse(t, fundResponse)
	if escrowed.State != "escrowed" {
		t.Fatalf("collectible state after funding = %q, want escrowed", escrowed.State)
	}

	openTask(t, server, owner.AccessToken, task.ID)
	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	accept := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "collectible-accept-"+task.ID)
	if accept.PayoutKind != "collectible" {
		t.Fatalf("payout kind = %q, want collectible", accept.PayoutKind)
	}
	if len(accept.CollectibleIDs) != 1 || accept.CollectibleIDs[0] != collectibleID {
		t.Fatalf("payout collectible ids = %v, want [%q]", accept.CollectibleIDs, collectibleID)
	}

	if owned := listCollectibles(t, server, owner.AccessToken); len(owned) != 0 {
		t.Fatalf("owner collectible count after award = %d, want 0", len(owned))
	}
	workerCollectibles := listCollectibles(t, server, worker.AccessToken)
	if len(workerCollectibles) != 1 {
		t.Fatalf("worker collectible count = %d, want 1", len(workerCollectibles))
	}
	if workerCollectibles[0].State != "awarded" {
		t.Fatalf("worker collectible state = %q, want awarded", workerCollectibles[0].State)
	}
}

func TestBundleRewardRequiresBothComponentsAndPaysBothOnAccept(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "bundle-owner")
	worker := registerUser(t, server, "bundle-worker")
	collectibleID := mintCollectible(t, server, owner.AccessToken, "Bundle medal")
	task := createPublicBundleUserTask(t, server, owner, 25)
	if task.RewardKind != "bundle" {
		t.Fatalf("task reward kind = %q, want bundle", task.RewardKind)
	}
	if task.RewardCreditAmount != 25 {
		t.Fatalf("task credit reward = %d, want 25", task.RewardCreditAmount)
	}
	if task.RewardCollectibleCount != 1 {
		t.Fatalf("task collectible reward count = %d, want 1", task.RewardCollectibleCount)
	}

	openBeforeFunding := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/open", []byte(`{}`), owner.AccessToken)
	defer openBeforeFunding.Body.Close()
	assertStatus(t, openBeforeFunding, http.StatusConflict)

	fundTask(t, server, owner.AccessToken, task.ID, 25, "bundle-fund-"+task.ID)
	openBeforeCollectible := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/open", []byte(`{}`), owner.AccessToken)
	defer openBeforeCollectible.Body.Close()
	assertStatus(t, openBeforeCollectible, http.StatusConflict)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectibleID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)

	openTask(t, server, owner.AccessToken, task.ID)
	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	accept := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "bundle-accept-"+task.ID)
	if accept.PayoutKind != "bundle" {
		t.Fatalf("payout kind = %q, want bundle", accept.PayoutKind)
	}
	if accept.PayoutAmount != 25 {
		t.Fatalf("payout amount = %d, want 25", accept.PayoutAmount)
	}
	if len(accept.CollectibleIDs) != 1 || accept.CollectibleIDs[0] != collectibleID {
		t.Fatalf("collectible payout ids = %v, want [%q]", accept.CollectibleIDs, collectibleID)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 125 {
		t.Fatalf("worker balance after bundle payout = %d, want 125", balance.Amount)
	}
	if workerCollectibles := listCollectibles(t, server, worker.AccessToken); len(workerCollectibles) != 1 || workerCollectibles[0].ID != collectibleID {
		t.Fatalf("worker did not receive bundled collectible")
	}

	repeat := acceptSubmission(t, server, owner.AccessToken, task.ID, submission.Submission.ID, "bundle-accept-"+task.ID)
	if repeat.PayoutKind != "bundle" {
		t.Fatalf("idempotent payout kind = %q, want bundle", repeat.PayoutKind)
	}
	if repeat.PayoutAmount != 25 || len(repeat.CollectibleIDs) != 1 || repeat.CollectibleIDs[0] != collectibleID {
		t.Fatalf("idempotent bundle payout = %d/%v, want 25/[%q]", repeat.PayoutAmount, repeat.CollectibleIDs, collectibleID)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 125 {
		t.Fatalf("worker balance after idempotent bundle accept = %d, want 125", balance.Amount)
	}
}

func TestCollectibleRewardRefundReturnsToOwner(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "collectible-refund-owner")
	collectibleID := mintCollectible(t, server, owner.AccessToken, "Refundable medal")
	task := createPublicUserTask(t, server, owner)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectibleID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)

	refundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-refund", []byte(`{}`), owner.AccessToken)
	defer refundResponse.Body.Close()
	assertStatus(t, refundResponse, http.StatusOK)
	refunded := decodeCollectiblesHTTPResponse(t, refundResponse)
	if len(refunded) != 1 || refunded[0].ID != collectibleID || refunded[0].State != "minted" {
		t.Fatalf("collectibles after refund = %v, want one minted %q", refunded, collectibleID)
	}

	owned := listCollectibles(t, server, owner.AccessToken)
	if len(owned) != 1 || owned[0].State != "minted" {
		t.Fatalf("owner should still hold the minted collectible after refund")
	}
}

func TestBundleRefundReturnsCreditsAndCollectible(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "bundle-refund-owner")
	collectibleID := mintCollectible(t, server, owner.AccessToken, "Bundle refund medal")
	task := createPublicBundleUserTask(t, server, owner, 30)
	fundTask(t, server, owner.AccessToken, task.ID, 30, "bundle-refund-fund-"+task.ID)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectibleID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)

	separateCollectibleRefund := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-refund", []byte(`{}`), owner.AccessToken)
	defer separateCollectibleRefund.Body.Close()
	assertStatus(t, separateCollectibleRefund, http.StatusConflict)

	refund := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/refund", []byte(`{"idempotency_key":"bundle-refund-`+task.ID+`"}`), owner.AccessToken)
	defer refund.Body.Close()
	assertStatus(t, refund, http.StatusOK)
	refundBody := decodeEscrowHTTPResponse(t, refund)
	if refundBody.State != "refunded" {
		t.Fatalf("bundle refund state = %q, want refunded", refundBody.State)
	}
	if balance := getBalance(t, server, owner.AccessToken); balance.Amount != 100 {
		t.Fatalf("owner balance after bundle refund = %d, want 100", balance.Amount)
	}
	owned := listCollectibles(t, server, owner.AccessToken)
	if len(owned) != 1 || owned[0].ID != collectibleID || owned[0].State != "minted" {
		t.Fatalf("owner did not receive refunded bundled collectible")
	}
}

func TestMintRejectsIssuerControlledRewardOnFunding(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "collectible-policy-owner")
	mintResponse := postJSONWithBearer(t, server.URL+"/api/collectibles", []byte(`{"name":"Issuer controlled","kind":"unique","transfer_policy":"issuer_controlled"}`), owner.AccessToken)
	defer mintResponse.Body.Close()
	assertStatus(t, mintResponse, http.StatusCreated)
	collectible := decodeCollectibleHTTPResponse(t, mintResponse)

	task := createPublicUserTask(t, server, owner)
	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/collectible-reward", []byte(`{"collectible_id":"`+collectible.ID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusConflict)
}

type collectibleHTTPResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	TransferPolicy string `json:"transfer_policy"`
	OwnerID        string `json:"owner_id"`
}

func mintCollectible(t *testing.T, server *httptest.Server, accessToken string, name string) string {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/collectibles", []byte(`{"name":"`+name+`","kind":"badge","transfer_policy":"non_transferable_except_payout"}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeCollectibleHTTPResponse(t, response).ID
}

func listCollectibles(t *testing.T, server *httptest.Server, accessToken string) []collectibleHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/collectibles", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body struct {
		Collectibles []collectibleHTTPResponse `json:"collectibles"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode collectibles: %v", err)
	}
	return body.Collectibles
}

func decodeCollectiblesHTTPResponse(t *testing.T, response *http.Response) []collectibleHTTPResponse {
	t.Helper()
	var body struct {
		Collectibles []collectibleHTTPResponse `json:"collectibles"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode collectibles: %v", err)
	}
	return body.Collectibles
}

func decodeCollectibleHTTPResponse(t *testing.T, response *http.Response) collectibleHTTPResponse {
	t.Helper()
	var body collectibleHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode collectible: %v", err)
	}
	if body.ID == "" {
		t.Fatalf("collectible id is empty")
	}
	return body
}
