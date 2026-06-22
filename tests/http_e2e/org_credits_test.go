//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrganizationCreditAccountFunding(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "org-credits-owner")
	organizationID := createOrganization(t, server, owner, "Org Credits Labs")

	if balance := organizationBalance(t, server, owner.AccessToken, organizationID); balance != 100 {
		t.Fatalf("organization grant balance = %d, want 100", balance)
	}

	taskID := createOrganizationTask(t, server, owner, organizationID)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/funding", []byte(`{"amount":30,"idempotency_key":"org-fund-`+taskID+`","organization_id":"`+organizationID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)
	escrow := decodeEscrowHTTPResponse(t, fundResponse)
	if escrow.State != "held" {
		t.Fatalf("escrow state = %q, want held", escrow.State)
	}

	if balance := organizationBalance(t, server, owner.AccessToken, organizationID); balance != 70 {
		t.Fatalf("organization balance after funding = %d, want 70", balance)
	}

	// Funding more than the organization holds is rejected.
	over := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/funding", []byte(`{"amount":200,"idempotency_key":"org-fund-over","organization_id":"`+organizationID+`"}`), owner.AccessToken)
	defer over.Body.Close()
	assertStatus(t, over, http.StatusBadRequest)
}

func createOrganization(t *testing.T, server *httptest.Server, owner authHTTPResponse, name string) string {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"`+name+`"}`), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode organization: %v", err)
	}
	if body.ID == "" {
		t.Fatalf("organization id is empty")
	}
	return body.ID
}

func createOrganizationTask(t *testing.T, server *httptest.Server, owner authHTTPResponse, organizationID string) string {
	t.Helper()
	body := `{
		"owner":{"kind":"organization","user_id":"","team_id":"","organization_id":"` + organizationID + `"},
		"title":"Organization task",
		"description":"A task owned by an organization.",
		"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response).ID
}

func organizationBalance(t *testing.T, server *httptest.Server, accessToken string, organizationID string) int64 {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/credits/balance", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body balanceHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode organization balance: %v", err)
	}
	return body.Amount
}
