//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestMCPAgentDiscoverSubmitAcceptFlow(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-owner")
	worker := registerUser(t, server, "mcp-worker")

	task := createPublicCreditUserTask(t, server, owner, 30)
	fundTask(t, server, owner.AccessToken, task.ID, 30, "fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read", "submissions_read", "submissions_review"})
	workerAgent := createAgentCredential(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write", "submissions_read"})

	// initialize and tools/list
	initialize := decodeRPC(t, mcpRequest(t, server, ownerAgent, `1`, "initialize", `{}`))
	if initialize.Error != nil {
		t.Fatalf("initialize error: %s", initialize.Error.Message)
	}
	toolsList := decodeRPC(t, mcpRequest(t, server, workerAgent, `2`, "tools/list", `{}`))
	if !strings.Contains(string(toolsList.Result), "sharecrop.submit_response") {
		t.Fatalf("tools/list missing submit tool: %s", string(toolsList.Result))
	}

	// worker reads the task and its schema
	getTask := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, `3`, "sharecrop.get_task", `{"task_id":"`+task.ID+`"}`)))
	if !strings.Contains(getTask, task.ID) {
		t.Fatalf("get_task missing task id: %s", getTask)
	}
	getSchema := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, `4`, "sharecrop.get_task_schema", `{"task_id":"`+task.ID+`"}`)))
	if !strings.Contains(getSchema, "freeform") {
		t.Fatalf("get_task_schema missing schema: %s", getSchema)
	}

	// worker submits a response through MCP
	submit := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, `5`, "sharecrop.submit_response", `{"task_id":"`+task.ID+`","response_json":"{\"answer\":\"done\"}"}`)))
	var submitPayload struct {
		SubmissionID string `json:"submission_id"`
		State        string `json:"state"`
		ReceiptToken string `json:"receipt_token"`
	}
	if err := json.Unmarshal([]byte(submit), &submitPayload); err != nil {
		t.Fatalf("decode submit payload: %v", err)
	}
	if submitPayload.State != "submitted" {
		t.Fatalf("submission state = %q, want submitted", submitPayload.State)
	}

	// worker checks submission status through MCP
	status := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, `6`, "sharecrop.get_submission_status", `{"receipt_token":"`+submitPayload.ReceiptToken+`"}`)))
	if !strings.Contains(status, submitPayload.SubmissionID) {
		t.Fatalf("status missing submission id: %s", status)
	}

	// owner lists submissions and accepts through MCP, paying the escrow
	list := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, `7`, "sharecrop.list_task_submissions", `{"task_id":"`+task.ID+`"}`)))
	if !strings.Contains(list, submitPayload.SubmissionID) {
		t.Fatalf("list_task_submissions missing submission: %s", list)
	}
	accept := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, `8`, "sharecrop.accept_submission", `{"task_id":"`+task.ID+`","submission_id":"`+submitPayload.SubmissionID+`","idempotency_key":"mcp-accept-`+task.ID+`"}`)))
	if !strings.Contains(accept, "\"payout_kind\":\"credit\"") {
		t.Fatalf("accept payout not credit: %s", accept)
	}

	if balance := getBalance(t, server, worker.AccessToken); balance.Amount != 130 {
		t.Fatalf("worker balance after MCP payout = %d, want 130", balance.Amount)
	}
}

func TestMCPEnforcesScopes(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-scope")
	task := createUserTask(t, server, owner)
	readOnly := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})

	response := decodeRPC(t, mcpCall(t, server, readOnly, `1`, "sharecrop.submit_response", `{"task_id":"`+task.ID+`","response_json":"{}"}`))
	if response.Error == nil {
		t.Fatalf("expected scope-denied error")
	}
	if response.Error.Code != -32001 {
		t.Fatalf("error code = %d, want -32001", response.Error.Code)
	}
}

func TestMCPRejectsRevokedCredential(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-revoke")
	credential := createAgentCredentialResponse(t, server, owner.AccessToken, []string{"tasks_read"})

	revoke := postJSONWithBearer(t, server.URL+"/api/agent-credentials/"+credential.Credential.ID+"/revoke", []byte(`{}`), owner.AccessToken)
	defer revoke.Body.Close()
	assertStatus(t, revoke, http.StatusOK)

	response := mcpRequest(t, server, credential.Secret, `1`, "tools/list", `{}`)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusUnauthorized)
}

func TestGetTaskEndpointReturnsSchema(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "get-task")
	task := createUserTask(t, server, owner)

	response := getWithBearer(t, server.URL+"/api/tasks/"+task.ID, owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	body := decodeTaskHTTPResponse(t, response)
	if body.ID != task.ID {
		t.Fatalf("task id = %q, want %q", body.ID, task.ID)
	}
}

func createPublicUserTask(t *testing.T, server *httptest.Server, owner authHTTPResponse) taskHTTPResponse {
	t.Helper()
	body := `{
		"owner":{"kind":"user","user_id":"` + owner.SubjectID + `","team_id":"","organization_id":""},
		"title":"Public agent task",
		"description":"A public task that agents can discover and submit to.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response)
}

func createPublicCreditUserTask(t *testing.T, server *httptest.Server, owner authHTTPResponse, amount int64) taskHTTPResponse {
	t.Helper()
	body := `{
		"owner":{"kind":"user","user_id":"` + owner.SubjectID + `","team_id":"","organization_id":""},
		"title":"Public credit agent task",
		"description":"A public task with a credit reward that agents can discover and submit to.",
		"reward":{"kind":"credit","credit_amount":` + strconv.FormatInt(amount, 10) + `},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(body), owner.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response)
}

type agentCredentialHTTPResponse struct {
	Credential struct {
		ID     string   `json:"id"`
		Label  string   `json:"label"`
		Scopes []string `json:"scopes"`
		State  string   `json:"state"`
	} `json:"credential"`
	Secret string `json:"secret"`
}

type rpcErrorBody struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type rpcEnvelope struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcErrorBody   `json:"error"`
}

func createAgentCredentialResponse(t *testing.T, server *httptest.Server, accessToken string, scopes []string) agentCredentialHTTPResponse {
	t.Helper()
	body, err := json.Marshal(struct {
		Label  string   `json:"label"`
		Scopes []string `json:"scopes"`
	}{Label: "Test agent", Scopes: scopes})
	if err != nil {
		t.Fatalf("encode agent credential request: %v", err)
	}
	response := postJSONWithBearer(t, server.URL+"/api/agent-credentials", body, accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var decoded agentCredentialHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode agent credential response: %v", err)
	}
	if decoded.Secret == "" {
		t.Fatalf("agent credential secret is empty")
	}
	return decoded
}

func createAgentCredential(t *testing.T, server *httptest.Server, accessToken string, scopes []string) string {
	t.Helper()
	return createAgentCredentialResponse(t, server, accessToken, scopes).Secret
}

func mcpRequest(t *testing.T, server *httptest.Server, agentToken string, id string, method string, params string) *http.Response {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":` + id + `,"method":"` + method + `","params":` + params + `}`
	request, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", strings.NewReader(body))
	if err != nil {
		t.Fatalf("create mcp request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+agentToken)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("post mcp request: %v", err)
	}
	return response
}

func mcpCall(t *testing.T, server *httptest.Server, agentToken string, id string, name string, arguments string) *http.Response {
	t.Helper()
	params := `{"name":"` + name + `","arguments":` + arguments + `}`
	return mcpRequest(t, server, agentToken, id, "tools/call", params)
}

func decodeRPC(t *testing.T, response *http.Response) rpcEnvelope {
	t.Helper()
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var envelope rpcEnvelope
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode rpc envelope: %v", err)
	}
	return envelope
}

func toolText(t *testing.T, envelope rpcEnvelope) string {
	t.Helper()
	if envelope.Error != nil {
		t.Fatalf("rpc error: %s", envelope.Error.Message)
	}
	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		IsError bool `json:"isError"`
	}
	if err := json.Unmarshal(envelope.Result, &result); err != nil {
		t.Fatalf("decode tool result: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned isError: %s", toolContent(result.Content))
	}
	if len(result.Content) != 1 {
		t.Fatalf("tool content count = %d, want 1", len(result.Content))
	}
	return result.Content[0].Text
}

func toolContent(content []struct {
	Text string `json:"text"`
}) string {
	if len(content) == 0 {
		return ""
	}
	return content[0].Text
}
