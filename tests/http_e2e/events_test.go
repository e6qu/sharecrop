//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

type eventFeedHTTPResponse struct {
	Events []struct {
		Kind          string `json:"kind"`
		ActorKind     string `json:"actor_kind"`
		ActorUserID   string `json:"actor_user_id"`
		Cursor        string `json:"cursor"`
		TaskID        string `json:"task_id"`
		ReservationID string `json:"reservation_id"`
	} `json:"events"`
	NextCursor string `json:"next_cursor"`
}

func fetchEventFeedHTTP(t *testing.T, serverURL string, accessToken string, query string) eventFeedHTTPResponse {
	t.Helper()
	response := getWithBearer(t, serverURL+"/api/events"+query, accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body eventFeedHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode event feed: %v", err)
	}
	return body
}

// TestReservationAppearsInOwnerEventFeed covers the live feed end to end: a
// worker reserving the owner's task makes a reservation_requested event
// appear in the owner's feed with an advancing cursor, and re-reading the
// feed after that cursor excludes the already-seen events.
func TestReservationAppearsInOwnerEventFeed(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "event-feed-owner")
	worker := registerUser(t, server, "event-feed-worker")

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	before := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "")
	beforeCursor := before.NextCursor

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	after := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "")
	if after.NextCursor == "" || after.NextCursor == beforeCursor {
		t.Fatalf("feed cursor did not advance: before %q after %q", beforeCursor, after.NextCursor)
	}
	foundReservation := false
	for _, value := range after.Events {
		if value.Kind == "reservation_requested" && value.TaskID == taskBody.ID {
			foundReservation = true
			if value.ActorKind != "user" || value.ActorUserID != worker.SubjectID {
				t.Fatalf("reservation event actor = %+v, want the worker", value)
			}
			if value.ReservationID == "" {
				t.Fatalf("reservation event carries no reservation reference: %+v", value)
			}
		}
	}
	if !foundReservation {
		t.Fatalf("owner feed has no reservation_requested event for task %s: %+v", taskBody.ID, after.Events)
	}

	// Resuming after the last-seen cursor excludes everything already read.
	resumed := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "?after="+after.NextCursor)
	if len(resumed.Events) != 0 || resumed.NextCursor != "" {
		t.Fatalf("feed after the newest cursor should be empty, got %+v", resumed)
	}
	if beforeCursor != "" {
		sinceBefore := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "?after="+beforeCursor)
		for _, value := range sinceBefore.Events {
			if value.Cursor <= beforeCursor && len(value.Cursor) <= len(beforeCursor) {
				t.Fatalf("feed after cursor %q returned older event %+v", beforeCursor, value)
			}
		}
	}

	// The worker's own feed also reflects their action, but the owner's
	// pre-reservation events (task_opened) stay invisible to strangers: a
	// third user sees neither.
	stranger := registerUser(t, server, "event-feed-stranger")
	strangerFeed := fetchEventFeedHTTP(t, server.URL, stranger.AccessToken, "")
	if len(strangerFeed.Events) != 0 {
		t.Fatalf("stranger's feed should be empty, got %+v", strangerFeed.Events)
	}
}
