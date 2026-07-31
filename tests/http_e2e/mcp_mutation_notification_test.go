//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestMCPMutationProducesOwnerNotification is the headline regression test for
// service-layer event emission: a mutation performed entirely over MCP (a
// worker agent calling sharecrop.submit_response) must leave an unread
// submission_created notification in the task owner's inbox, exactly as the
// same mutation over REST does.
func TestMCPMutationProducesOwnerNotification(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "mcp-mutation-owner")
	worker := registerUser(t, server, "mcp-mutation-worker")
	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	workerAgent := createAgentCredential(t, server, worker.AccessToken, []string{"tasks_read", "submissions_write"})
	workerSession := initializeMCPSession(t, server, workerAgent)
	submit := toolText(t, decodeRPC(t, mcpCall(t, server, workerAgent, workerSession, `1`, "sharecrop.submit_response", `{"task_id":"`+task.ID+`","response_json":"{\"answer\":\"done\"}"}`)))
	if !strings.Contains(submit, "receipt_token") {
		t.Fatalf("mcp submit did not return a receipt: %s", submit)
	}

	// The owner's inbox has the unread submission_created row over REST...
	unreadResponse := getWithBearer(t, server.URL+"/api/notifications?state=unread", owner.AccessToken)
	defer unreadResponse.Body.Close()
	assertStatus(t, unreadResponse, http.StatusOK)
	var unreadBody struct {
		Notifications []struct {
			Kind        string `json:"kind"`
			State       string `json:"state"`
			ActorUserID string `json:"actor_user_id"`
		} `json:"notifications"`
	}
	if err := json.NewDecoder(unreadResponse.Body).Decode(&unreadBody); err != nil {
		t.Fatalf("decode unread notifications: %v", err)
	}
	found := false
	for _, value := range unreadBody.Notifications {
		if value.Kind == "submission_created" && value.State == "unread" && value.ActorUserID == worker.SubjectID {
			found = true
		}
	}
	if !found {
		t.Fatalf("mcp submission left no unread submission_created notification for the owner: %+v", unreadBody.Notifications)
	}

	// ...and the count endpoint plus the MCP count tool both see it.
	countResponse := getWithBearer(t, server.URL+"/api/notifications/unread-count", owner.AccessToken)
	defer countResponse.Body.Close()
	assertStatus(t, countResponse, http.StatusOK)
	var countBody struct {
		UnreadCount int64 `json:"unread_count"`
	}
	if err := json.NewDecoder(countResponse.Body).Decode(&countBody); err != nil {
		t.Fatalf("decode unread count: %v", err)
	}
	if countBody.UnreadCount < 1 {
		t.Fatalf("unread count = %d, want at least 1", countBody.UnreadCount)
	}

	ownerAgent := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read"})
	ownerSession := initializeMCPSession(t, server, ownerAgent)
	counted := toolText(t, decodeRPC(t, mcpCall(t, server, ownerAgent, ownerSession, `1`, "sharecrop.get_unread_notification_count", `{}`)))
	if !strings.Contains(counted, `"unread_count"`) || strings.Contains(counted, `"unread_count":0`) {
		t.Fatalf("mcp unread count tool did not report the owner's unread notification: %s", counted)
	}
}
