//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// taggedPublicTaskRequestJSON builds a public task request whose title
// carries a unique tag, so listing assertions can filter with ?query= and
// stay isolated from rows other tests created in the shared database.
func taggedPublicTaskRequestJSON(userID string, title string, rewardKind string, amount int64) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"` + title + `",
		"description":"A tagged task used by list assertions.",
		"reward":{"kind":"` + rewardKind + `","credit_amount":` + strconv.FormatInt(amount, 10) + `},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func taggedReservationTaskRequestJSON(userID string, title string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"` + title + `",
		"description":"A tagged reservation task used by list assertions.",
		"reward":{"kind":"none","credit_amount":0},
		"participation":{"policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func createOpenTaggedTask(t *testing.T, server *httptest.Server, owner authHTTPResponse, requestJSON string) string {
	t.Helper()
	taskID := createUserTaskFromJSON(t, server, owner.AccessToken, requestJSON)
	openTask(t, server, owner.AccessToken, taskID)
	return taskID
}

// TestAgentCredentialPollsOwnerEventFeed covers agent access to the cursor
// feed: a personal agent credential holding notifications_read reads exactly
// the owner's recipient-scoped feed, and a credential without the scope is
// denied.
func TestAgentCredentialPollsOwnerEventFeed(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "agent-feed-owner")
	worker := registerUser(t, server, "agent-feed-worker")

	createResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createResponse.Body.Close()
	assertStatus(t, createResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	readToken := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read"})
	agentFeed := fetchEventFeedHTTP(t, server.URL, readToken, "")
	if len(agentFeed.Events) == 0 {
		t.Fatalf("agent credential feed is empty, want the owner's events")
	}
	foundReservation := false
	for _, event := range agentFeed.Events {
		if event.Kind == "reservation_requested" && event.TaskID == taskBody.ID {
			foundReservation = true
		}
	}
	if !foundReservation {
		t.Fatalf("agent credential feed is missing the reservation_requested event: %+v", agentFeed.Events)
	}

	// The credential sees exactly the owner's own feed.
	sessionFeed := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "")
	if len(sessionFeed.Events) != len(agentFeed.Events) || sessionFeed.NextCursor != agentFeed.NextCursor {
		t.Fatalf("agent feed (%d events, cursor %q) diverges from the owner session feed (%d events, cursor %q)",
			len(agentFeed.Events), agentFeed.NextCursor, len(sessionFeed.Events), sessionFeed.NextCursor)
	}

	// A credential without notifications_read is denied with 403.
	tasksOnlyToken := createAgentCredential(t, server, owner.AccessToken, []string{"tasks_read"})
	denied := getWithBearer(t, server.URL+"/api/events", tasksOnlyToken)
	defer denied.Body.Close()
	assertStatus(t, denied, http.StatusForbidden)
	assertErrorCode(t, denied, "permission_denied")
}

// TestEventFeedLongPollReturnsWhenAnEventArrives holds a ?wait= request open
// in one goroutine while the main goroutine produces an event, and asserts
// the held request returns early with that event instead of running out the
// full wait.
func TestEventFeedLongPollReturnsWhenAnEventArrives(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "long-poll-owner")
	worker := registerUser(t, server, "long-poll-worker")

	createResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createResponse.Body.Close()
	assertStatus(t, createResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	readToken := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read"})
	drained := fetchEventFeedHTTP(t, server.URL, readToken, "")

	query := "?wait=15"
	if drained.NextCursor != "" {
		query = "?after=" + drained.NextCursor + "&wait=15"
	}

	type longPollOutcome struct {
		feed    eventFeedHTTPResponse
		status  int
		failure string
		elapsed time.Duration
	}
	outcomes := make(chan longPollOutcome, 1)
	started := time.Now()
	go func() {
		request, err := http.NewRequest(http.MethodGet, server.URL+"/api/events"+query, nil)
		if err != nil {
			outcomes <- longPollOutcome{failure: "build request: " + err.Error()}
			return
		}
		request.Header.Set("Authorization", "Bearer "+readToken)
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			outcomes <- longPollOutcome{failure: "long poll request: " + err.Error()}
			return
		}
		defer response.Body.Close()
		outcome := longPollOutcome{status: response.StatusCode, elapsed: time.Since(started)}
		if err := json.NewDecoder(response.Body).Decode(&outcome.feed); err != nil {
			outcome.failure = "decode long poll response: " + err.Error()
		}
		outcomes <- outcome
	}()

	// Give the long poll time to reach its holding loop, then produce the
	// event it is waiting for.
	time.Sleep(500 * time.Millisecond)
	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	select {
	case outcome := <-outcomes:
		if outcome.failure != "" {
			t.Fatalf("long poll failed: %s", outcome.failure)
		}
		if outcome.status != http.StatusOK {
			t.Fatalf("long poll status = %d, want %d", outcome.status, http.StatusOK)
		}
		if len(outcome.feed.Events) == 0 {
			t.Fatalf("long poll returned no events, want the reservation event")
		}
		if outcome.elapsed >= 10*time.Second {
			t.Fatalf("long poll held for %v, want an early return once the event landed", outcome.elapsed)
		}
	case <-time.After(20 * time.Second):
		t.Fatalf("long poll did not return after the event landed")
	}
}

// TestEventFeedWaitParameterBounds pins the wait parameter's contract: a
// malformed or negative wait is rejected, and a wait beyond the 25-second cap
// is accepted (clamped) rather than rejected.
func TestEventFeedWaitParameterBounds(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "wait-bounds-owner")
	readToken := createAgentCredential(t, server, owner.AccessToken, []string{"notifications_read"})

	for _, raw := range []string{"soon", "-1", "1.5"} {
		rejected := getWithBearer(t, server.URL+"/api/events?wait="+raw, readToken)
		defer rejected.Body.Close()
		assertStatus(t, rejected, http.StatusBadRequest)
		assertErrorCode(t, rejected, "invalid_argument")
	}

	// Opening a task records a task_opened event with the owner as a
	// recipient, so an over-cap wait answers immediately with the pending
	// page instead of holding: over-cap values are clamped, not rejected.
	createOpenTaggedTask(t, server, owner, taggedPublicTaskRequestJSON(owner.SubjectID, "wait bounds "+uniqueTestSuffix(t), "none", 0))
	overCap := fetchEventFeedHTTP(t, server.URL, readToken, "?wait=9999")
	if len(overCap.Events) == 0 {
		t.Fatalf("over-cap wait returned no events, want the pending feed page")
	}
}

// TestTaskListTotalsAcrossPagesAndFilters pins the `total` field: it counts
// every row matching the filter, stays identical across pages, and ignores
// limit/offset.
func TestTaskListTotalsAcrossPagesAndFilters(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "totals-owner")
	tag := "totals-" + uniqueTestSuffix(t)
	for index := 0; index < 3; index++ {
		createOpenTaggedTask(t, server, owner, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" task "+strconv.Itoa(index), "none", 0))
	}

	firstPage := getWithBearer(t, server.URL+"/api/tasks?query="+tag+"&limit=2&offset=0", owner.AccessToken)
	defer firstPage.Body.Close()
	assertStatus(t, firstPage, http.StatusOK)
	first := decodeTasksHTTPResponse(t, firstPage)
	if len(first.Tasks) != 2 || first.NextOffset != 2 || first.Total != 3 {
		t.Fatalf("first page = %d tasks, next_offset %d, total %d; want 2 tasks, next_offset 2, total 3", len(first.Tasks), first.NextOffset, first.Total)
	}

	secondPage := getWithBearer(t, server.URL+"/api/tasks?query="+tag+"&limit=2&offset=2", owner.AccessToken)
	defer secondPage.Body.Close()
	assertStatus(t, secondPage, http.StatusOK)
	second := decodeTasksHTTPResponse(t, secondPage)
	if len(second.Tasks) != 1 || second.NextOffset != 0 || second.Total != 3 {
		t.Fatalf("second page = %d tasks, next_offset %d, total %d; want 1 task, next_offset 0, total 3", len(second.Tasks), second.NextOffset, second.Total)
	}
}

// TestLedgerAndSubmissionTotals pins `total` on the credits ledger and the
// task submissions listing.
func TestLedgerAndSubmissionTotals(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "ledger-totals-owner")
	worker := registerUser(t, server, "ledger-totals-worker")
	task := createPublicCreditUserTask(t, server, owner, 10)
	fundTask(t, server, owner.AccessToken, task.ID, 10, "totals-fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)

	// The signup grant and the funding entry are the owner's whole ledger.
	for offset := 0; offset < 2; offset++ {
		page := getWithBearer(t, server.URL+"/api/credits/ledger?limit=1&offset="+strconv.Itoa(offset), owner.AccessToken)
		defer page.Body.Close()
		assertStatus(t, page, http.StatusOK)
		var body ledgerHTTPResponse
		if err := json.NewDecoder(page.Body).Decode(&body); err != nil {
			t.Fatalf("decode ledger page %d: %v", offset, err)
		}
		if len(body.Entries) != 1 || body.Total != 2 {
			t.Fatalf("ledger page %d = %d entries, total %d; want 1 entry, total 2", offset, len(body.Entries), body.Total)
		}
	}

	submitAuthenticated(t, server, worker.AccessToken, task.ID)
	submitAuthenticated(t, server, owner.AccessToken, task.ID)

	listResponse := getWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions?limit=1&offset=0", owner.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	submissions := decodeSubmissionsHTTPResponse(t, listResponse)
	if len(submissions.Submissions) != 1 || submissions.Total != 2 || submissions.NextOffset != 1 {
		t.Fatalf("submissions page = %d rows, next_offset %d, total %d; want 1 row, next_offset 1, total 2", len(submissions.Submissions), submissions.NextOffset, submissions.Total)
	}
}

// TestTaskListFundedFilterMatrix drives the ?funded= filter across the three
// funded states and checks each list item's funded field.
func TestTaskListFundedFilterMatrix(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "funded-owner")
	tag := "funded-" + uniqueTestSuffix(t)

	fundedTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" funded", "credit", 10))
	fundTask(t, server, owner.AccessToken, fundedTaskID, 10, "matrix-fund-"+fundedTaskID)
	openTask(t, server, owner.AccessToken, fundedTaskID)

	// A credit-reward task cannot open unfunded, so the reward_unfunded row
	// is a draft; the matrix therefore lists scope=user (the owner's own
	// listing), which includes drafts alongside the open tasks.
	unfundedTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" unfunded", "credit", 10))
	noCreditTaskID := createOpenTaggedTask(t, server, owner, taggedPublicTaskRequestJSON(owner.SubjectID, tag+" no credit", "none", 0))

	cases := []struct {
		filter     string
		wantTaskID string
		wantFunded string
	}{
		{filter: "reward_funded", wantTaskID: fundedTaskID, wantFunded: "reward_funded"},
		{filter: "reward_unfunded", wantTaskID: unfundedTaskID, wantFunded: "reward_unfunded"},
		{filter: "no_credit_reward", wantTaskID: noCreditTaskID, wantFunded: "no_credit_reward"},
	}
	for _, testCase := range cases {
		response := getWithBearer(t, server.URL+"/api/tasks?scope=user&query="+tag+"&funded="+testCase.filter, owner.AccessToken)
		defer response.Body.Close()
		assertStatus(t, response, http.StatusOK)
		body := decodeTasksHTTPResponse(t, response)
		if len(body.Tasks) != 1 || body.Tasks[0].ID != testCase.wantTaskID {
			t.Fatalf("funded=%s returned %+v, want exactly task %s", testCase.filter, body.Tasks, testCase.wantTaskID)
		}
		if body.Tasks[0].Funded != testCase.wantFunded {
			t.Fatalf("funded=%s item funded = %q, want %q", testCase.filter, body.Tasks[0].Funded, testCase.wantFunded)
		}
		if body.Total != 1 {
			t.Fatalf("funded=%s total = %d, want 1", testCase.filter, body.Total)
		}
	}

	all := getWithBearer(t, server.URL+"/api/tasks?scope=user&query="+tag, owner.AccessToken)
	defer all.Body.Close()
	assertStatus(t, all, http.StatusOK)
	allBody := decodeTasksHTTPResponse(t, all)
	if len(allBody.Tasks) != 3 || allBody.Total != 3 {
		t.Fatalf("unfiltered listing = %d tasks, total %d; want all 3", len(allBody.Tasks), allBody.Total)
	}

	invalid := getWithBearer(t, server.URL+"/api/tasks?funded=bogus", owner.AccessToken)
	defer invalid.Body.Close()
	assertStatus(t, invalid, http.StatusBadRequest)
	assertErrorCode(t, invalid, "invalid_enum")
}

// TestTaskListExposesReservationHolderName pins holder_display_name: empty
// while the task is unreserved, and the reserving worker's display name once
// an active user reservation exists.
func TestTaskListExposesReservationHolderName(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "holder-owner")
	worker := registerUser(t, server, "holder-worker")
	tag := "holder-" + uniqueTestSuffix(t)
	taskID := createOpenTaggedTask(t, server, owner, taggedReservationTaskRequestJSON(owner.SubjectID, tag+" reservable"))

	before := getWithBearer(t, server.URL+"/api/tasks?query="+tag, owner.AccessToken)
	defer before.Body.Close()
	assertStatus(t, before, http.StatusOK)
	beforeItem := findTaskInListing(t, decodeTasksHTTPResponse(t, before), taskID)
	if beforeItem.HolderDisplayName != "" {
		t.Fatalf("unreserved holder_display_name = %q, want empty", beforeItem.HolderDisplayName)
	}

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	after := getWithBearer(t, server.URL+"/api/tasks?query="+tag+"&include_reserved=true", owner.AccessToken)
	defer after.Body.Close()
	assertStatus(t, after, http.StatusOK)
	afterItem := findTaskInListing(t, decodeTasksHTTPResponse(t, after), taskID)
	if afterItem.HolderDisplayName != worker.DisplayName {
		t.Fatalf("reserved holder_display_name = %q, want %q", afterItem.HolderDisplayName, worker.DisplayName)
	}
}

// TestCompetingSubmissionBecomesSuperseded covers the superseded lifecycle
// over the REST API: accepting one submission closes the task and moves the
// competing submitted row to the terminal superseded state, visible in the
// task listing, the loser's own submission listing, and the loser's
// notification inbox.
func TestCompetingSubmissionBecomesSuperseded(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "supersede-owner")
	winner := registerUser(t, server, "supersede-winner")
	loser := registerUser(t, server, "supersede-loser")

	task := createPublicCreditUserTask(t, server, owner, 10)
	fundTask(t, server, owner.AccessToken, task.ID, 10, "supersede-fund-"+task.ID)
	openTask(t, server, owner.AccessToken, task.ID)

	winning := submitAuthenticated(t, server, winner.AccessToken, task.ID)
	losing := submitAuthenticated(t, server, loser.AccessToken, task.ID)

	acceptSubmission(t, server, owner.AccessToken, task.ID, winning.Submission.ID, "supersede-accept-"+task.ID)

	listResponse := getWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/submissions", owner.AccessToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	listed := decodeSubmissionsHTTPResponse(t, listResponse)
	states := map[string]string{}
	for _, value := range listed.Submissions {
		states[value.ID] = value.State
	}
	if states[winning.Submission.ID] != "accepted" {
		t.Fatalf("winning submission state = %q, want accepted", states[winning.Submission.ID])
	}
	if states[losing.Submission.ID] != "superseded" {
		t.Fatalf("losing submission state = %q, want superseded", states[losing.Submission.ID])
	}

	ownListing := getWithBearer(t, server.URL+"/api/users/"+loser.SubjectID+"/submissions", loser.AccessToken)
	defer ownListing.Body.Close()
	assertStatus(t, ownListing, http.StatusOK)
	own := decodeSubmissionsHTTPResponse(t, ownListing)
	if len(own.Submissions) != 1 || own.Submissions[0].State != "superseded" {
		t.Fatalf("loser's own submissions = %+v, want the one superseded row", own.Submissions)
	}

	inboxResponse := getWithBearer(t, server.URL+"/api/notifications", loser.AccessToken)
	defer inboxResponse.Body.Close()
	assertStatus(t, inboxResponse, http.StatusOK)
	var inbox struct {
		Notifications []struct {
			Kind         string `json:"kind"`
			SubjectID    string `json:"subject_id"`
			SubjectTitle string `json:"subject_title"`
		} `json:"notifications"`
		Total int64 `json:"total"`
	}
	if err := json.NewDecoder(inboxResponse.Body).Decode(&inbox); err != nil {
		t.Fatalf("decode loser inbox: %v", err)
	}
	foundSuperseded := false
	for _, value := range inbox.Notifications {
		if value.Kind == "submission_superseded" && value.SubjectID == losing.Submission.ID {
			foundSuperseded = true
			if value.SubjectTitle == "" {
				t.Fatalf("submission_superseded notification has no subject title")
			}
		}
	}
	if !foundSuperseded {
		t.Fatalf("loser inbox is missing the submission_superseded notification: %+v", inbox.Notifications)
	}
	if inbox.Total != int64(len(inbox.Notifications)) {
		t.Fatalf("inbox total = %d, want %d (single page)", inbox.Total, len(inbox.Notifications))
	}
}
