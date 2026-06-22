//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	if accept.CollectibleID != collectibleID {
		t.Fatalf("payout collectible id = %q, want %q", accept.CollectibleID, collectibleID)
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
	refunded := decodeCollectibleHTTPResponse(t, refundResponse)
	if refunded.State != "minted" {
		t.Fatalf("collectible state after refund = %q, want minted", refunded.State)
	}

	owned := listCollectibles(t, server, owner.AccessToken)
	if len(owned) != 1 || owned[0].State != "minted" {
		t.Fatalf("owner should still hold the minted collectible after refund")
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
