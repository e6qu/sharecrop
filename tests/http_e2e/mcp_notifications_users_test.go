//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestMCPNotificationsLifecycle(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-notify-owner")
	worker := registerUser(t, server, "mcp-notify-worker")
	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	// A worker submitting a response notifies the task owner.
	submitResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", []byte(`{"response_json":"{\"answer\":\"done\"}"}`), worker.AccessToken)
	defer submitResponse.Body.Close()
	assertStatus(t, submitResponse, http.StatusCreated)

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read", "notifications_manage"})
	session := initializeMCPSession(t, server, ownerAgent)

	listed := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `1`, "sharecrop.list_notifications", `{}`)))
	var notificationsPayload struct {
		Notifications []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"notifications"`
	}
	if err := json.Unmarshal([]byte(listed), &notificationsPayload); err != nil {
		t.Fatalf("decode list_notifications: %v (%s)", err, listed)
	}
	if len(notificationsPayload.Notifications) == 0 {
		t.Fatalf("expected at least one notification for the task owner")
	}
	notificationID := notificationsPayload.Notifications[0].ID

	read := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, session, `2`, "sharecrop.mark_notification_read", `{"notification_id":"`+notificationID+`"}`)))
	if !strings.Contains(read, `"state":"read"`) {
		t.Fatalf("mark_notification_read did not report read state: %s", read)
	}
}

func TestMCPUsersLifecycle(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerEmail := "mcp-users-owner-" + uniqueTestSuffix(t) + "@example.com"
	owner := registerUserWithEmail(t, server, ownerEmail)
	workerEmail := "mcp-users-worker-" + uniqueTestSuffix(t) + "@example.com"
	worker := registerUserWithEmail(t, server, workerEmail)
	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	submitResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", []byte(`{"response_json":"{\"answer\":\"done\"}"}`), worker.AccessToken)
	defer submitResponse.Body.Close()
	assertStatus(t, submitResponse, http.StatusCreated)

	workerAgent := createAgentCredential(t, server, worker.AccessToken, []string{"users_read"})
	session := initializeMCPSession(t, server, workerAgent)

	directory := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, session, `1`, "sharecrop.list_users", `{"query":"`+workerEmail+`"}`)))
	if !strings.Contains(directory, worker.SubjectID) {
		t.Fatalf("list_users missing the current user: %s", directory)
	}

	ownerProfile := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, session, `2`, "sharecrop.get_user_profile", `{"user_id":"`+owner.SubjectID+`"}`)))
	if !strings.Contains(ownerProfile, task.ID) {
		t.Fatalf("get_user_profile missing the owner's created task: %s", ownerProfile)
	}

	workerWork := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, session, `3`, "sharecrop.get_user_work", `{"user_id":"`+worker.SubjectID+`"}`)))
	_ = workerWork // work list may be empty once the task is closed by submission; presence of the tasks key is enough to confirm wiring
	if !strings.Contains(workerWork, `"tasks"`) {
		t.Fatalf("get_user_work missing tasks key: %s", workerWork)
	}

	workerSubmissions := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, session, `4`, "sharecrop.get_user_submissions", `{"user_id":"`+worker.SubjectID+`"}`)))
	if !strings.Contains(workerSubmissions, task.ID) {
		t.Fatalf("get_user_submissions missing the worker's submission: %s", workerSubmissions)
	}
}
