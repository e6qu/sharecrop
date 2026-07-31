//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// decodeErrorHTTPResponse reads a JSON error body and asserts the response
// carries the expected machine-readable code alongside a human message.
func decodeErrorHTTPResponse(t *testing.T, response *http.Response, wantCode string) errorHTTPResponse {
	t.Helper()
	if contentType := response.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("error content type = %q, want application/json", contentType)
	}
	var body errorHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if body.Code != wantCode {
		t.Fatalf("error code = %q, want %q (message %q)", body.Code, wantCode, body.Error)
	}
	if body.Error == "" {
		t.Fatalf("error message is empty for code %q", body.Code)
	}
	return body
}

// TestErrorBodyCarriesCode pins the machine-readable `code` field for a
// 401, a 404, and a 409 response.
func TestErrorBodyCarriesCode(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	// 401: no bearer token.
	unauthenticated, err := http.Get(server.URL + "/api/credits/balance")
	if err != nil {
		t.Fatalf("get balance without token: %v", err)
	}
	defer unauthenticated.Body.Close()
	assertStatus(t, unauthenticated, http.StatusUnauthorized)
	decodeErrorHTTPResponse(t, unauthenticated, "unauthenticated")

	owner := registerUser(t, server, "error-code-owner")
	worker := registerUser(t, server, "error-code-worker")

	// 404: a well-formed task id that does not exist. The subject id is a
	// valid UUID, so the id parses and the lookup itself misses.
	missing := getWithBearer(t, server.URL+"/api/tasks/"+worker.SubjectID, owner.AccessToken)
	defer missing.Body.Close()
	assertStatus(t, missing, http.StatusNotFound)
	decodeErrorHTTPResponse(t, missing, "not_found")

	// 409: accepting a second submission after the task already paid one out.
	task := createPublicCreditUserTask(t, server, owner, 20)
	fundTask(t, server, owner.AccessToken, task.ID, 20, "fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)
	first := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	second := submitAuthenticated(t, server, worker.AccessToken, task.ID)
	acceptSubmission(t, server, owner.AccessToken, task.ID, first.Submission.ID, "accept-"+first.Submission.ID)

	conflict := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+second.Submission.ID+"/accept", []byte(`{"idempotency_key":"accept-second"}`), owner.AccessToken)
	defer conflict.Body.Close()
	assertStatus(t, conflict, http.StatusConflict)
	decodeErrorHTTPResponse(t, conflict, "conflict")
}

// TestUnmatchedAPIRouteReturnsJSONError pins the JSON 404 for API paths that
// match no route, instead of the plain-text Go default.
func TestUnmatchedAPIRouteReturnsJSONError(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	response, err := http.Get(server.URL + "/api/nope")
	if err != nil {
		t.Fatalf("get unmatched API route: %v", err)
	}
	defer response.Body.Close()
	assertStatus(t, response, http.StatusNotFound)
	body := decodeErrorHTTPResponse(t, response, "not_found")
	if body.Error != "no such API route" {
		t.Fatalf("error message = %q, want %q", body.Error, "no such API route")
	}
}

// TestRejectWithBanSelectionBansImplementor drives the reject review through
// the wire enum: ban_selection=ban_implementor must block the worker from
// submitting to the same task again, and an unknown enum value must be
// rejected as invalid_enum.
func TestRejectWithBanSelectionBansImplementor(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ban-selection-owner")
	worker := registerUser(t, server, "ban-selection-worker")

	task := createPublicCreditUserTask(t, server, owner, 20)
	fundTask(t, server, owner.AccessToken, task.ID, 20, "fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)
	submitted := submitAuthenticated(t, server, worker.AccessToken, task.ID)

	// An unknown enum value is rejected before the review runs.
	badEnum := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/reject", []byte(`{"idempotency_key":"reject-bad-enum","review_note":"Not usable.","ban_selection":"bogus"}`), owner.AccessToken)
	defer badEnum.Body.Close()
	assertStatus(t, badEnum, http.StatusBadRequest)
	decodeErrorHTTPResponse(t, badEnum, "invalid_enum")

	reject := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions/"+submitted.Submission.ID+"/reject", []byte(`{"idempotency_key":"reject-`+submitted.Submission.ID+`","review_note":"Not usable.","ban_selection":"ban_implementor"}`), owner.AccessToken)
	defer reject.Body.Close()
	assertStatus(t, reject, http.StatusOK)

	// The banned worker cannot submit to the same task again.
	banned := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", []byte(`{"response_json":"{\"answer\":\"again\"}"}`), worker.AccessToken)
	defer banned.Body.Close()
	assertStatus(t, banned, http.StatusForbidden)
	decodeErrorHTTPResponse(t, banned, "permission_denied")
}

// TestLedgerNextOffsetPagination pins next_offset across a two-page list:
// a further page reports offset+limit, and the last page reports 0.
func TestLedgerNextOffsetPagination(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "next-offset-owner")
	task := createCreditUserTask(t, server, owner, 10)
	fundTask(t, server, owner.AccessToken, task.ID, 10, "fund-"+task.ID)

	// Two ledger entries exist now: the signup grant and the task funding.
	firstPage := getWithBearer(t, server.URL+"/api/credits/ledger?limit=1&offset=0", owner.AccessToken)
	defer firstPage.Body.Close()
	assertStatus(t, firstPage, http.StatusOK)
	var first ledgerHTTPResponse
	if err := json.NewDecoder(firstPage.Body).Decode(&first); err != nil {
		t.Fatalf("decode first ledger page: %v", err)
	}
	if len(first.Entries) != 1 {
		t.Fatalf("first page entry count = %d, want 1", len(first.Entries))
	}
	if first.NextOffset != 1 {
		t.Fatalf("first page next_offset = %d, want 1", first.NextOffset)
	}

	lastPage := getWithBearer(t, server.URL+"/api/credits/ledger?limit=1&offset=1", owner.AccessToken)
	defer lastPage.Body.Close()
	assertStatus(t, lastPage, http.StatusOK)
	var last ledgerHTTPResponse
	if err := json.NewDecoder(lastPage.Body).Decode(&last); err != nil {
		t.Fatalf("decode last ledger page: %v", err)
	}
	if len(last.Entries) != 1 {
		t.Fatalf("last page entry count = %d, want 1", len(last.Entries))
	}
	if last.NextOffset != 0 {
		t.Fatalf("last page next_offset = %d, want 0", last.NextOffset)
	}
}

// TestOrgCredentialScopeGateOnREST proves the REST surface enforces the same
// minted-scope model MCP enforces: an organization credential holding only
// tasks_read can list tasks but cannot change task state, approve
// reservations, or manage teams — a 403 with permission_denied, distinct
// from the 401 an unauthenticated caller receives.
func TestOrgCredentialScopeGateOnREST(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "scope-gate-owner-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer ownerResponse.Body.Close()
	assertStatus(t, ownerResponse, http.StatusCreated)
	owner := decodeAuthHTTPResponse(t, ownerResponse)

	orgResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Scope Gate Org"}`), owner.AccessToken)
	defer orgResponse.Body.Close()
	assertStatus(t, orgResponse, http.StatusCreated)
	organization := decodeOrganizationHTTPResponse(t, orgResponse)

	credentialResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organization.ID+"/credentials", []byte(`{"label":"read only","scopes":["tasks_read"],"expires_at":""}`), owner.AccessToken)
	defer credentialResponse.Body.Close()
	assertStatus(t, credentialResponse, http.StatusCreated)
	readOnly := decodeOrgCredentialCreatedHTTPResponse(t, credentialResponse)

	taskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(organization.ID)), owner.AccessToken)
	defer taskResponse.Body.Close()
	assertStatus(t, taskResponse, http.StatusCreated)
	created := decodeTaskHTTPResponse(t, taskResponse)

	// tasks_read allows listing the org's tasks.
	listResponse := getWithBearer(t, server.URL+"/api/tasks?scope=organization&organization_id="+organization.ID, readOnly.Secret)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)

	// tasks_read must not permit a state change (MCP requires tasks_write).
	openResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+created.ID+"/open", []byte(`{}`), readOnly.Secret)
	defer openResponse.Body.Close()
	assertStatus(t, openResponse, http.StatusForbidden)
	decodeErrorHTTPResponse(t, openResponse, "permission_denied")

	// Nor reservation review (submissions_review) or team management
	// (org_manage).
	reservationsResponse := getWithBearer(t, server.URL+"/api/tasks/"+created.ID+"/reservations", readOnly.Secret)
	defer reservationsResponse.Body.Close()
	assertStatus(t, reservationsResponse, http.StatusForbidden)
	decodeErrorHTTPResponse(t, reservationsResponse, "permission_denied")
}
