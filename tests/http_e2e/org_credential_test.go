//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type orgCredentialCreatedHTTPResponse struct {
	Credential struct {
		ID             string   `json:"id"`
		OrganizationID string   `json:"organization_id"`
		Label          string   `json:"label"`
		Scopes         []string `json:"scopes"`
		State          string   `json:"state"`
	} `json:"credential"`
	Secret string `json:"secret"`
}

func decodeOrgCredentialCreatedHTTPResponse(t *testing.T, response *http.Response) orgCredentialCreatedHTTPResponse {
	t.Helper()
	var body orgCredentialCreatedHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode org credential response: %v", err)
	}
	if body.Secret == "" {
		t.Fatalf("org credential secret is empty")
	}
	return body
}

// TestOrgCredentialActsWithFullParityOnItsOwnOrgOnly is the automated
// regression test for the exact flow manually verified with curl against a
// real server: an org-wide credential can drive owner-level task actions on
// its own organization (full parity with an org-admin member), and is
// rejected outright against a different organization's resources — the org
// id boundary is the actual authorization gate, not just the scope check.
func TestOrgCredentialActsWithFullParityOnItsOwnOrgOnly(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerAResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "org-cred-owner-a-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer ownerAResponse.Body.Close()
	assertStatus(t, ownerAResponse, http.StatusCreated)
	ownerA := decodeAuthHTTPResponse(t, ownerAResponse)

	orgAResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Org A"}`), ownerA.AccessToken)
	defer orgAResponse.Body.Close()
	assertStatus(t, orgAResponse, http.StatusCreated)
	orgA := decodeOrganizationHTTPResponse(t, orgAResponse)

	credentialResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+orgA.ID+"/credentials", []byte(`{"label":"Org A automation","scopes":["tasks_read","tasks_write","org_manage"],"expires_at":""}`), ownerA.AccessToken)
	defer credentialResponse.Body.Close()
	assertStatus(t, credentialResponse, http.StatusCreated)
	credential := decodeOrgCredentialCreatedHTTPResponse(t, credentialResponse)

	createTaskAResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(orgA.ID)), ownerA.AccessToken)
	defer createTaskAResponse.Body.Close()
	assertStatus(t, createTaskAResponse, http.StatusCreated)
	taskA := decodeTaskHTTPResponse(t, createTaskAResponse)

	// Org A's own token opens org A's own task: allowed (full parity).
	openOwnResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskA.ID+"/open", []byte(`{}`), credential.Secret)
	defer openOwnResponse.Body.Close()
	assertStatus(t, openOwnResponse, http.StatusOK)

	// A second, unrelated organization and task.
	ownerBResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "org-cred-owner-b-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer ownerBResponse.Body.Close()
	assertStatus(t, ownerBResponse, http.StatusCreated)
	ownerB := decodeAuthHTTPResponse(t, ownerBResponse)

	orgBResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Org B"}`), ownerB.AccessToken)
	defer orgBResponse.Body.Close()
	assertStatus(t, orgBResponse, http.StatusCreated)
	orgB := decodeOrganizationHTTPResponse(t, orgBResponse)

	createTaskBResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(orgB.ID)), ownerB.AccessToken)
	defer createTaskBResponse.Body.Close()
	assertStatus(t, createTaskBResponse, http.StatusCreated)
	taskB := decodeTaskHTTPResponse(t, createTaskBResponse)

	// Org A's token against org B's task: rejected outright, not silently
	// scoped down — the org id boundary is the actual gate.
	openOtherResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskB.ID+"/open", []byte(`{}`), credential.Secret)
	defer openOtherResponse.Body.Close()
	assertStatus(t, openOtherResponse, http.StatusForbidden)

	// Org A's token listing org B's tasks: also rejected.
	listOtherResponse := getWithBearer(t, server.URL+"/api/tasks?scope=organization&organization_id="+orgB.ID, credential.Secret)
	defer listOtherResponse.Body.Close()
	assertStatus(t, listOtherResponse, http.StatusForbidden)
}
