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
	OwnerKind      string `json:"owner_kind"`
	Art            string `json:"art"`
}

func TestDefaultCollectibleCatalogAwardAndTransfer(t *testing.T) {
	// Register the users on a bootstrap server first so their ids are known, then
	// rebuild the server with the admin allowlist pointing at the admin user (the
	// shared database persists the registrations).
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "collectible-admin")
	recipient := registerUser(t, bootstrap, "collectible-recipient")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	// A non-admin cannot award default collectibles.
	forbidden := postJSONWithBearer(t, server.URL+"/api/collectibles/award",
		[]byte(`{"slug":"harvest-star","recipient_kind":"user","recipient_id":"`+recipient.SubjectID+`"}`), recipient.AccessToken)
	defer forbidden.Body.Close()
	assertStatus(t, forbidden, http.StatusForbidden)

	// The catalog exposes the 25 default collectibles, each with a sprite slug.
	catalogResponse := getWithBearer(t, server.URL+"/api/collectibles/catalog", admin.AccessToken)
	defer catalogResponse.Body.Close()
	assertStatus(t, catalogResponse, http.StatusOK)
	var catalog struct {
		Entries []struct {
			Slug string `json:"slug"`
			Art  string `json:"art"`
		} `json:"entries"`
	}
	if err := json.NewDecoder(catalogResponse.Body).Decode(&catalog); err != nil {
		t.Fatalf("decode catalog: %v", err)
	}
	if len(catalog.Entries) != 25 {
		t.Fatalf("catalog has %d entries, want 25", len(catalog.Entries))
	}
	if catalog.Entries[0].Slug == "" || catalog.Entries[0].Art == "" {
		t.Fatalf("catalog entry missing slug/art: %+v", catalog.Entries[0])
	}

	// Awarding a default mints a fresh copy owned by the recipient, carrying the
	// catalog sprite.
	awardResponse := postJSONWithBearer(t, server.URL+"/api/collectibles/award",
		[]byte(`{"slug":"harvest-star","recipient_kind":"user","recipient_id":"`+recipient.SubjectID+`"}`), admin.AccessToken)
	defer awardResponse.Body.Close()
	assertStatus(t, awardResponse, http.StatusCreated)
	awarded := decodeCollectibleHTTPResponse(t, awardResponse)
	if awarded.Art != "harvest-star" || awarded.OwnerID != recipient.SubjectID {
		t.Fatalf("awarded collectible = %+v, want art harvest-star owned by recipient", awarded)
	}

	// The recipient now holds it and can trade it back to the admin.
	if held := listCollectibles(t, server, recipient.AccessToken); len(held) != 1 || held[0].ID != awarded.ID {
		t.Fatalf("recipient holdings = %+v, want the awarded collectible", held)
	}
	transferResponse := postJSONWithBearer(t, server.URL+"/api/collectibles/"+awarded.ID+"/transfer",
		[]byte(`{"recipient_id":"`+admin.SubjectID+`"}`), recipient.AccessToken)
	defer transferResponse.Body.Close()
	assertStatus(t, transferResponse, http.StatusOK)
	if held := listCollectibles(t, server, admin.AccessToken); len(held) != 1 || held[0].ID != awarded.ID {
		t.Fatalf("admin holdings after trade = %+v, want the traded collectible", held)
	}

	// An admin can also award to a team or organization; the holding shows up in
	// that owner's collectibles (the recipient subject id stands in for a team id).
	teamAward := postJSONWithBearer(t, server.URL+"/api/collectibles/award",
		[]byte(`{"slug":"golden-sickle","recipient_kind":"team","recipient_id":"`+recipient.SubjectID+`"}`), admin.AccessToken)
	defer teamAward.Body.Close()
	assertStatus(t, teamAward, http.StatusCreated)
	teamCollectible := decodeCollectibleHTTPResponse(t, teamAward)
	if teamCollectible.OwnerKind != "team" {
		t.Fatalf("team-awarded owner kind = %q, want team", teamCollectible.OwnerKind)
	}
	teamHoldings := getWithBearer(t, server.URL+"/api/teams/"+recipient.SubjectID+"/collectibles", admin.AccessToken)
	defer teamHoldings.Body.Close()
	assertStatus(t, teamHoldings, http.StatusOK)
	var teamBody struct {
		Collectibles []collectibleHTTPResponse `json:"collectibles"`
	}
	if err := json.NewDecoder(teamHoldings.Body).Decode(&teamBody); err != nil {
		t.Fatalf("decode team holdings: %v", err)
	}
	if len(teamBody.Collectibles) != 1 || teamBody.Collectibles[0].ID != teamCollectible.ID {
		t.Fatalf("team holdings = %+v, want the team-awarded collectible", teamBody.Collectibles)
	}
}

func TestCollectibleTipTransfersOnAccept(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "tip-owner")
	worker := registerUser(t, server, "tip-worker")

	// A transferable collectible the requester will tip (mintCollectible uses a
	// non-transferable policy, which is correctly refused for tips).
	mintResponse := postJSONWithBearer(t, server.URL+"/api/collectibles", []byte(`{"name":"Gratitude token","kind":"badge","transfer_policy":"transferable_between_users"}`), owner.AccessToken)
	defer mintResponse.Body.Close()
	assertStatus(t, mintResponse, http.StatusCreated)
	tipID := decodeCollectibleHTTPResponse(t, mintResponse).ID

	// A credit-funded task so the accept produces a payout that identifies the worker.
	task := createPublicCreditUserTask(t, server, owner, 30)
	fundTask(t, server, owner.AccessToken, task.ID, 30, "tip-fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)
	submission := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	accept := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submission.Submission.ID+"/accept",
		[]byte(`{"idempotency_key":"tip-accept-`+task.ID+`","payout_amount":30,"tip_collectible_id":"`+tipID+`"}`), owner.AccessToken)
	defer accept.Body.Close()
	assertStatus(t, accept, http.StatusOK)

	workerOwned := listCollectibles(t, server, worker.AccessToken)
	if len(workerOwned) != 1 || workerOwned[0].ID != tipID {
		t.Fatalf("worker should own the tipped collectible, got %+v", workerOwned)
	}
	if ownerOwned := listCollectibles(t, server, owner.AccessToken); len(ownerOwned) != 0 {
		t.Fatalf("requester should no longer own the tipped collectible, got %d", len(ownerOwned))
	}
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
