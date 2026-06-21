//go:build http_e2e

package http_e2e_test

import (
	"bytes"
	"encoding/json"
	"net/http"
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
