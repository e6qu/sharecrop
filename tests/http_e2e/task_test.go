//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestTaskHTTPFlow(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "task-owner-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer ownerResponse.Body.Close()
	assertStatus(t, ownerResponse, http.StatusCreated)
	ownerBody := decodeAuthHTTPResponse(t, ownerResponse)

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(userTaskRequestJSON(ownerBody.SubjectID)), ownerBody.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createTaskResponse)
	if taskBody.State != "draft" {
		t.Fatalf("task state = %q, want draft", taskBody.State)
	}
	if taskBody.VisibilityKind != "user" {
		t.Fatalf("task visibility = %q, want user", taskBody.VisibilityKind)
	}

	openResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/open", []byte(`{}`), ownerBody.AccessToken)
	defer openResponse.Body.Close()
	assertStatus(t, openResponse, http.StatusOK)

	listResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user", ownerBody.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	listBody := decodeTasksHTTPResponse(t, listResponse)
	if len(listBody.Tasks) != 1 {
		t.Fatalf("task count = %d, want 1", len(listBody.Tasks))
	}

	tokenResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/capability-tokens", []byte(`{}`), ownerBody.AccessToken)
	defer tokenResponse.Body.Close()
	assertStatus(t, tokenResponse, http.StatusCreated)
	tokenBody := decodeTaskCapabilityTokenHTTPResponse(t, tokenResponse)
	if strings.Contains(tokenBody.Token, taskBody.ID) {
		t.Fatalf("capability token contained task id")
	}

	cancelResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/cancel", []byte(`{}`), ownerBody.AccessToken)
	defer cancelResponse.Body.Close()
	assertStatus(t, cancelResponse, http.StatusOK)
}

func TestOrganizationPublicTaskRequiresPublisherRole(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    "org-owner-" + uniqueTestSuffix(t) + "@example.com",
		Password: "correct horse battery staple",
	}, nil)
	defer ownerResponse.Body.Close()
	assertStatus(t, ownerResponse, http.StatusCreated)
	ownerBody := decodeAuthHTTPResponse(t, ownerResponse)

	createOrganizationResponse := postJSONWithBearer(t, server.URL+"/api/organizations", []byte(`{"name":"Task Org"}`), ownerBody.AccessToken)
	defer createOrganizationResponse.Body.Close()
	assertStatus(t, createOrganizationResponse, http.StatusCreated)
	organizationBody := decodeOrganizationHTTPResponse(t, createOrganizationResponse)

	adminEmail := "task-admin-" + uniqueTestSuffix(t) + "@example.com"
	adminResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    adminEmail,
		Password: "correct horse battery staple",
	}, nil)
	defer adminResponse.Body.Close()
	assertStatus(t, adminResponse, http.StatusCreated)
	adminBody := decodeAuthHTTPResponse(t, adminResponse)

	provisionAdminResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationBody.ID+"/members", []byte(`{"email":"`+adminEmail+`","roles":["admin"]}`), ownerBody.AccessToken)
	defer provisionAdminResponse.Body.Close()
	assertStatus(t, provisionAdminResponse, http.StatusCreated)

	deniedResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(organizationBody.ID)), adminBody.AccessToken)
	defer deniedResponse.Body.Close()
	assertStatus(t, deniedResponse, http.StatusForbidden)

	publisherEmail := "task-publisher-" + uniqueTestSuffix(t) + "@example.com"
	publisherResponse := postAuthJSON(t, server.URL+"/api/auth/register", authHTTPRequest{
		Email:    publisherEmail,
		Password: "correct horse battery staple",
	}, nil)
	defer publisherResponse.Body.Close()
	assertStatus(t, publisherResponse, http.StatusCreated)
	publisherBody := decodeAuthHTTPResponse(t, publisherResponse)

	provisionPublisherResponse := postJSONWithBearer(t, server.URL+"/api/organizations/"+organizationBody.ID+"/members", []byte(`{"email":"`+publisherEmail+`","roles":["admin","public_publisher"]}`), ownerBody.AccessToken)
	defer provisionPublisherResponse.Body.Close()
	assertStatus(t, provisionPublisherResponse, http.StatusCreated)

	acceptedResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(organizationPublicTaskRequestJSON(organizationBody.ID)), publisherBody.AccessToken)
	defer acceptedResponse.Body.Close()
	assertStatus(t, acceptedResponse, http.StatusCreated)
}

type taskHTTPResponse struct {
	ID             string `json:"id"`
	State          string `json:"state"`
	VisibilityKind string `json:"visibility_kind"`
}

type tasksHTTPResponse struct {
	Tasks []taskHTTPResponse `json:"tasks"`
}

type taskCapabilityTokenHTTPResponse struct {
	Token string `json:"token"`
}

func userTaskRequestJSON(userID string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Review schema samples",
		"description":"Review response examples against the local schema parser.",
		"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func organizationPublicTaskRequestJSON(organizationID string) string {
	return `{
		"owner":{"kind":"organization","user_id":"","team_id":"","organization_id":"` + organizationID + `"},
		"title":"Publish public task",
		"description":"Publish a task that can be discovered publicly.",
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func getWithBearer(t *testing.T, url string, accessToken string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, http.NoBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("get with bearer: %v", err)
	}
	return response
}

func decodeTaskHTTPResponse(t *testing.T, response *http.Response) taskHTTPResponse {
	t.Helper()
	var body taskHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode task response: %v", err)
	}
	if body.ID == "" {
		t.Fatalf("task id is empty")
	}
	return body
}

func decodeTasksHTTPResponse(t *testing.T, response *http.Response) tasksHTTPResponse {
	t.Helper()
	var body tasksHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode tasks response: %v", err)
	}
	return body
}

func decodeTaskCapabilityTokenHTTPResponse(t *testing.T, response *http.Response) taskCapabilityTokenHTTPResponse {
	t.Helper()
	var body taskCapabilityTokenHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode task capability token response: %v", err)
	}
	if body.Token == "" {
		t.Fatalf("capability token is empty")
	}
	return body
}
