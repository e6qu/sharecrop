//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestSubmissionReceiptRedactionAndRequesterList(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "submission-owner")
	worker := registerUser(t, server, "submission-worker")

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicSensitiveTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	submitResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/submissions", []byte(`{"response_json":"{\"email\":\"person@example.com\"}"}`), worker.AccessToken)
	defer submitResponse.Body.Close()
	assertStatus(t, submitResponse, http.StatusCreated)
	createdBody := decodeSubmissionCreatedHTTPResponse(t, submitResponse)
	if createdBody.Submission.State != "submitted" {
		t.Fatalf("submission state = %q, want submitted", createdBody.Submission.State)
	}

	receiptResponse, err := http.Get(server.URL + "/api/submission-receipts/" + createdBody.ReceiptToken)
	if err != nil {
		t.Fatalf("get receipt: %v", err)
	}
	defer receiptResponse.Body.Close()
	assertStatus(t, receiptResponse, http.StatusOK)
	receiptBody := decodeSubmissionHTTPResponse(t, receiptResponse)
	if receiptBody.ResponseJSON != `{"email":"[redacted]"}` {
		t.Fatalf("receipt response = %q, want redacted email", receiptBody.ResponseJSON)
	}

	listResponse := getWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/submissions", owner.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	listBody := decodeSubmissionsHTTPResponse(t, listResponse)
	if len(listBody.Submissions) != 1 {
		t.Fatalf("submission count = %d, want 1", len(listBody.Submissions))
	}
}

func TestInvalidSubmissionIsRecorded(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "invalid-submission-owner")
	worker := registerUser(t, server, "invalid-submission-worker")

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicSensitiveTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	submitResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/submissions", []byte(`{"response_json":"{\"email\":12}"}`), worker.AccessToken)
	defer submitResponse.Body.Close()
	assertStatus(t, submitResponse, http.StatusCreated)
	createdBody := decodeSubmissionCreatedHTTPResponse(t, submitResponse)
	if createdBody.Submission.State != "invalid" {
		t.Fatalf("submission state = %q, want invalid", createdBody.Submission.State)
	}
	if len(createdBody.Submission.ValidationErrors) != 1 {
		t.Fatalf("validation error count = %d, want 1", len(createdBody.Submission.ValidationErrors))
	}
}

type submissionHTTPResponse struct {
	ID               string                             `json:"id"`
	TaskID           string                             `json:"task_id"`
	SubmitterID      string                             `json:"submitter_id"`
	State            string                             `json:"state"`
	ResponseJSON     string                             `json:"response_json"`
	ValidationErrors []submissionValidationHTTPResponse `json:"validation_errors"`
}

type submissionValidationHTTPResponse struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type submissionCreatedHTTPResponse struct {
	Submission   submissionHTTPResponse `json:"submission"`
	ReceiptToken string                 `json:"receipt_token"`
}

type submissionsHTTPResponse struct {
	Submissions []submissionHTTPResponse `json:"submissions"`
}

func publicSensitiveTaskRequestJSON(userID string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Collect contact",
		"description":"Collect a contact email for validation.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"object\",\"fields\":[{\"name\":\"email\",\"presence\":\"required\",\"schema\":{\"kind\":\"string\"},\"sensitivity\":{\"category\":\"pii\",\"retention\":\"delete_on_request\",\"redaction\":\"replace\"}}]}",
		"payload":{"kind":"none","json":""}
	}`
}

func decodeSubmissionCreatedHTTPResponse(t *testing.T, response *http.Response) submissionCreatedHTTPResponse {
	t.Helper()
	var body submissionCreatedHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode submission created response: %v", err)
	}
	if body.ReceiptToken == "" {
		t.Fatalf("receipt token is empty")
	}
	if body.Submission.ID == "" {
		t.Fatalf("submission id is empty")
	}
	return body
}

func decodeSubmissionHTTPResponse(t *testing.T, response *http.Response) submissionHTTPResponse {
	t.Helper()
	var body submissionHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode submission response: %v", err)
	}
	return body
}

func decodeSubmissionsHTTPResponse(t *testing.T, response *http.Response) submissionsHTTPResponse {
	t.Helper()
	var body submissionsHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode submissions response: %v", err)
	}
	return body
}
