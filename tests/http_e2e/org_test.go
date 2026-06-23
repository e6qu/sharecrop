//go:build http_e2e

package http_e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOrganizationHTTPFlow(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerEmail := "owner-" + uniqueTestSuffix(t) + "@example.com"
	ownerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    ownerEmail,
		Password: "correct horse battery staple",
	}, nil)
	defer ownerResponse.Body.Close()
	assertStatus(t, ownerResponse, http.StatusCreated)
	ownerBody := decodeAuthHTTPResponse(t, ownerResponse)

	createOrganizationResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Sharecrop Labs"}`), ownerBody.AccessToken)
	defer createOrganizationResponse.Body.Close()
	assertStatus(t, createOrganizationResponse, http.StatusCreated)
	organizationBody := decodeOrganizationHTTPResponse(t, createOrganizationResponse)

	createTeamResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationBody.ID+"/teams", []byte(`{"name":"Reviewers"}`), ownerBody.AccessToken)
	defer createTeamResponse.Body.Close()
	assertStatus(t, createTeamResponse, http.StatusCreated)

	memberEmail := "member-" + uniqueTestSuffix(t) + "@example.com"
	memberResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    memberEmail,
		Password: "correct horse battery staple",
	}, nil)
	defer memberResponse.Body.Close()
	assertStatus(t, memberResponse, http.StatusCreated)

	provisionResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationBody.ID+"/members", []byte(`{"email":"`+memberEmail+`","roles":["member","reviewer"]}`), ownerBody.AccessToken)
	defer provisionResponse.Body.Close()
	assertStatus(t, provisionResponse, http.StatusCreated)
}

func TestOrganizationReviewerCanReviewSubmissions(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "org-rev-owner")
	organizationID := createOrganization(t, server, owner, "Org Reviewer Labs")

	reviewerEmail := "org-rev-reviewer-" + uniqueTestSuffix(t) + "@example.com"
	reviewer := registerUserWithEmail(t, server, reviewerEmail)
	provisionOrganizationMember(t, server, owner.AccessToken, organizationID, reviewerEmail, `["member","reviewer"]`)

	memberEmail := "org-rev-member-" + uniqueTestSuffix(t) + "@example.com"
	member := registerUserWithEmail(t, server, memberEmail)
	provisionOrganizationMember(t, server, owner.AccessToken, organizationID, memberEmail, `["member"]`)

	worker := registerUser(t, server, "org-rev-worker")
	taskID := createPublicOrganizationTask(t, server, owner, organizationID)

	fundResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/funding", []byte(`{"amount":30,"idempotency_key":"org-rev-fund-`+taskID+`","organization_id":"`+organizationID+`"}`), owner.AccessToken)
	defer fundResponse.Body.Close()
	assertStatus(t, fundResponse, http.StatusCreated)
	openTask(t, server, owner.AccessToken, taskID)
	submission := submitAuthenticated(t, server, worker.AccessToken, taskID)

	// A member without the reviewer role cannot accept a submission for the org task.
	memberAccept := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/submissions/"+submission.Submission.ID+"/accept", []byte(`{"idempotency_key":"org-rev-member-accept"}`), member.AccessToken)
	defer memberAccept.Body.Close()
	assertStatus(t, memberAccept, http.StatusForbidden)

	// An unrelated outsider cannot accept either.
	outsiderAccept := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/submissions/"+submission.Submission.ID+"/accept", []byte(`{"idempotency_key":"org-rev-outsider-accept"}`), worker.AccessToken)
	defer outsiderAccept.Body.Close()
	assertStatus(t, outsiderAccept, http.StatusForbidden)

	// The org reviewer, who did not create the task, can accept it.
	accept := acceptSubmission(t, server, reviewer.AccessToken, taskID, submission.Submission.ID, "org-rev-accept-"+taskID)
	if accept.SubmissionID != submission.Submission.ID {
		t.Fatalf("accepted submission id = %q, want %q", accept.SubmissionID, submission.Submission.ID)
	}
	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 130 {
		t.Fatalf("worker balance after org reviewer payout = %d, want 130", balance.Amount)
	}
}

func TestStandaloneTeamCreateAndList(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "standalone-team-owner")

	createResponse := postJSONWithBearer(t, server.URL+"/api/teams", []byte(`{"name":"Solo Crew"}`), owner.AccessToken)
	defer createResponse.Body.Close()
	assertStatus(t, createResponse, http.StatusCreated)
	var created teamHTTPResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode standalone team: %v", err)
	}
	if created.OwnerKind != "user" {
		t.Fatalf("owner kind = %q, want user", created.OwnerKind)
	}
	if created.OwnerUserID != owner.SubjectID {
		t.Fatalf("owner user id = %q, want %q", created.OwnerUserID, owner.SubjectID)
	}
	if created.OrganizationID != "" {
		t.Fatalf("organization id = %q, want empty for a standalone team", created.OrganizationID)
	}

	listResponse := getWithBearer(t, server.URL+"/api/teams", owner.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	var listing teamsHTTPResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&listing); err != nil {
		t.Fatalf("decode standalone teams: %v", err)
	}
	if len(listing.Teams) != 1 || listing.Teams[0].Name != "Solo Crew" {
		t.Fatalf("standalone teams = %+v, want one named Solo Crew", listing.Teams)
	}

	// A different user does not see another user's standalone team.
	other := registerUser(t, server, "standalone-team-other")
	otherResponse := getWithBearer(t, server.URL+"/api/teams", other.AccessToken)
	defer otherResponse.Body.Close()
	assertStatus(t, otherResponse, http.StatusOK)
	var otherListing teamsHTTPResponse
	if err := json.NewDecoder(otherResponse.Body).Decode(&otherListing); err != nil {
		t.Fatalf("decode other standalone teams: %v", err)
	}
	if len(otherListing.Teams) != 0 {
		t.Fatalf("other user standalone teams = %d, want 0", len(otherListing.Teams))
	}
}

type teamHTTPResponse struct {
	ID             string `json:"id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	OwnerUserID    string `json:"owner_user_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type teamsHTTPResponse struct {
	Teams []teamHTTPResponse `json:"teams"`
}

func registerUserWithEmail(t *testing.T, server *httptest.Server, email string) authHTTPResponse {
	t.Helper()
	response := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    email,
		Password: "correct horse battery staple",
	}, nil)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeAuthHTTPResponse(t, response)
}

func provisionOrganizationMember(t *testing.T, server *httptest.Server, accessToken string, organizationID string, email string, rolesJSON string) {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationID+"/members", []byte(`{"email":"`+email+`","roles":`+rolesJSON+`}`), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
}

func createPublicOrganizationTask(t *testing.T, server *httptest.Server, owner authHTTPResponse, organizationID string) string {
	t.Helper()
	body := `{
		"owner":{"kind":"organization","user_id":"","team_id":"","organization_id":"` + organizationID + `"},
		"title":"Public organization task",
		"description":"A public task owned by an organization.",
		"reward":{"kind":"credit","credit_amount":30},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response).ID
}

type organizationHTTPResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

func postJSONWithBearer(t *testing.T, url string, encoded []byte, accessToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(encoded))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post json with bearer: %v", err)
	}
	return response
}

func decodeOrganizationHTTPResponse(t *testing.T, response *http.Response) organizationHTTPResponse {
	t.Helper()
	var body organizationHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode organization response: %v", err)
	}
	if body.ID == "" {
		t.Fatalf("organization id is empty")
	}
	if body.Name == "" {
		t.Fatalf("organization name is empty")
	}
	if body.CreatedBy == "" {
		t.Fatalf("organization creator is empty")
	}
	return body
}
