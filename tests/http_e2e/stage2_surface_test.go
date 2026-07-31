//go:build http_e2e

package http_e2e_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// This file covers the agent-loop REST surface added in the stage-2 API
// work: the default public task listing, personal-agent-credential listing,
// created_after filtering, the enriched display-name/title read models, the
// platform-admin credit grant endpoint, webhook audiences, and the account
// profile / display-name surface.

func localPart(email string) string {
	return strings.Split(email, "@")[0]
}

func TestBareTaskListingDefaultsToPublicScope(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerEmail := "bare-list-owner-" + uniqueTestSuffix(t) + "@example.com"
	owner := registerUserWithEmail(t, server, ownerEmail)
	worker := registerUser(t, server, "bare-list-worker")

	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	// No scope parameter at all: the listing defaults to the public scope.
	response := mustGet(t, server, worker.AccessToken, "/api/tasks")
	defer response.Body.Close()
	body := decodeTasksHTTPResponse(t, response)
	listed := findTaskInListing(t, body, task.ID)
	if listed.CreatorDisplayName != localPart(ownerEmail) {
		t.Fatalf("creator_display_name = %q, want %q", listed.CreatorDisplayName, localPart(ownerEmail))
	}
	// The worker did not create the task, so its pending-review count is 0.
	if listed.PendingReviewCount != 0 {
		t.Fatalf("pending_review_count for non-owner = %d, want 0", listed.PendingReviewCount)
	}
}

func TestOwnerListingReportsPendingReviewCount(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "pending-owner")
	worker := registerUser(t, server, "pending-worker")

	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)
	submitAuthenticated(t, server, worker.AccessToken, task.ID)

	ownList := mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user")
	defer ownList.Body.Close()
	ownBody := decodeTasksHTTPResponse(t, ownList)
	if listed := findTaskInListing(t, ownBody, task.ID); listed.PendingReviewCount != 1 {
		t.Fatalf("owner pending_review_count = %d, want 1", listed.PendingReviewCount)
	}

	workerList := mustGet(t, server, worker.AccessToken, "/api/tasks")
	defer workerList.Body.Close()
	workerBody := decodeTasksHTTPResponse(t, workerList)
	if listed := findTaskInListing(t, workerBody, task.ID); listed.PendingReviewCount != 0 {
		t.Fatalf("worker-visible pending_review_count = %d, want 0", listed.PendingReviewCount)
	}
}

func TestPersonalAgentCredentialListsPublicTasksOnly(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "agent-list-owner")
	agentOperator := registerUser(t, server, "agent-list-operator")
	task := createPublicUserTask(t, server, owner)
	openTask(t, server, owner.AccessToken, task.ID)

	readToken := createAgentCredential(t, server, agentOperator.AccessToken, []string{"tasks_read"})

	// A bare listing (public by default) works with the agent credential.
	listResponse := getWithBearer(t, server.URL+"/api/tasks", readToken)
	defer listResponse.Body.Close()
	assertStatus(t, listResponse, http.StatusOK)
	assertTaskPresent(t, decodeTasksHTTPResponse(t, listResponse), task.ID)

	// Any non-public scope is denied for a personal agent credential.
	deniedResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user", readToken)
	defer deniedResponse.Body.Close()
	assertStatus(t, deniedResponse, http.StatusForbidden)
	assertErrorCode(t, deniedResponse, "permission_denied")

	// A credential without tasks_read cannot list at all.
	writeOnly := createAgentCredential(t, server, agentOperator.AccessToken, []string{"submissions_write"})
	scopeDenied := getWithBearer(t, server.URL+"/api/tasks", writeOnly)
	defer scopeDenied.Body.Close()
	assertStatus(t, scopeDenied, http.StatusForbidden)
	assertErrorCode(t, scopeDenied, "permission_denied")
}

func TestTaskListCreatedAfterFilter(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "created-after-owner")

	early := createUserTaskFromJSON(t, server, owner.AccessToken, titledUserTaskRequestJSON(owner.SubjectID, "Created-after early "+uniqueTestSuffix(t)))
	openTask(t, server, owner.AccessToken, early)

	time.Sleep(50 * time.Millisecond)
	boundary := time.Now().UTC().Format(time.RFC3339Nano)
	time.Sleep(50 * time.Millisecond)

	late := createUserTaskFromJSON(t, server, owner.AccessToken, titledUserTaskRequestJSON(owner.SubjectID, "Created-after late "+uniqueTestSuffix(t)))
	openTask(t, server, owner.AccessToken, late)

	response := mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user&created_after="+url.QueryEscape(boundary))
	defer response.Body.Close()
	body := decodeTasksHTTPResponse(t, response)
	assertTaskPresent(t, body, late)
	assertTaskAbsent(t, body, early)

	invalid := getWithBearer(t, server.URL+"/api/tasks?created_after=not-a-timestamp", owner.AccessToken)
	defer invalid.Body.Close()
	assertStatus(t, invalid, http.StatusBadRequest)
	assertErrorCode(t, invalid, "invalid_argument")
}

func TestAdminCreditGrants(t *testing.T) {
	t.Setenv("SHARECROP_ADMIN_USER_IDS", "")
	bootstrap := newAuthHTTPServer(t, t.Context())
	admin := registerUser(t, bootstrap, "grant-admin")
	beneficiary := registerUser(t, bootstrap, "grant-beneficiary")
	bootstrap.Close()

	t.Setenv("SHARECROP_ADMIN_USER_IDS", admin.SubjectID)
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	grantKey := "grant-" + uniqueTestSuffix(t)
	grantBody := []byte(`{"target_kind":"user","target_id":"` + beneficiary.SubjectID + `","amount":15,"note":"Reimbursed the cancelled batch.","idempotency_key":"` + grantKey + `"}`)

	// A non-admin is denied.
	denied := postJSONWithBearer(t, server.URL+"/api/admin/credits/grants", grantBody, beneficiary.AccessToken)
	defer denied.Body.Close()
	assertStatus(t, denied, http.StatusForbidden)
	assertErrorCode(t, denied, "permission_denied")

	granted := postCreditGrant(t, server, admin.AccessToken, grantBody)
	if granted.Amount != 15 || granted.EntryID == "" {
		t.Fatalf("grant response = %+v, want amount 15 and an entry id", granted)
	}
	if balance := getBalance(t, server, beneficiary.AccessToken); balance.SpendableCredits != 115 {
		t.Fatalf("beneficiary balance after grant = %d, want 115", balance.SpendableCredits)
	}

	// Replaying the same idempotency key does not double-credit.
	replayed := postCreditGrant(t, server, admin.AccessToken, grantBody)
	if replayed.EntryID != granted.EntryID {
		t.Fatalf("replayed entry id = %q, want %q", replayed.EntryID, granted.EntryID)
	}
	if balance := getBalance(t, server, beneficiary.AccessToken); balance.SpendableCredits != 115 {
		t.Fatalf("beneficiary balance after replay = %d, want 115", balance.SpendableCredits)
	}

	// The note is visible in the beneficiary's ledger.
	ledgerBody := getLedger(t, server, beneficiary.AccessToken)
	foundNote := false
	for _, entry := range ledgerBody.Entries {
		if entry.Kind == "manual_adjustment" && entry.Note == "Reimbursed the cancelled batch." && entry.Amount == 15 {
			foundNote = true
		}
	}
	if !foundNote {
		t.Fatalf("ledger entries = %+v, want a manual_adjustment carrying the grant note", ledgerBody.Entries)
	}

	// The beneficiary is notified.
	if !hasNotificationKind(t, server, beneficiary.AccessToken, "credit_granted") {
		t.Fatalf("beneficiary inbox is missing the credit_granted notification")
	}

	// An organization account can be granted to as well.
	organizationID := createOrganization(t, server, admin, "Grant Org "+uniqueTestSuffix(t))
	balanceBeforeGrant := organizationBalance(t, server, admin.AccessToken, organizationID)
	organizationGrant := postCreditGrant(t, server, admin.AccessToken, []byte(`{"target_kind":"organization","target_id":"`+organizationID+`","amount":30,"note":"Seeded the org wallet.","idempotency_key":"org-`+grantKey+`"}`))
	if organizationGrant.Amount != 30 {
		t.Fatalf("organization grant amount = %d, want 30", organizationGrant.Amount)
	}
	if balance := organizationBalance(t, server, admin.AccessToken, organizationID); balance != balanceBeforeGrant+30 {
		t.Fatalf("organization balance after grant = %d, want %d", balance, balanceBeforeGrant+30)
	}

	// An unknown target kind is an enum error.
	badKind := postJSONWithBearer(t, server.URL+"/api/admin/credits/grants", []byte(`{"target_kind":"team","target_id":"`+beneficiary.SubjectID+`","amount":5,"note":"n","idempotency_key":"bad-`+grantKey+`"}`), admin.AccessToken)
	defer badKind.Body.Close()
	assertStatus(t, badKind, http.StatusBadRequest)
}

type creditGrantHTTPResponse struct {
	EntryID string `json:"entry_id"`
	Amount  int64  `json:"amount"`
}

func postCreditGrant(t *testing.T, server *httptest.Server, accessToken string, body []byte) creditGrantHTTPResponse {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/admin/credits/grants", body, accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var decoded creditGrantHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		t.Fatalf("decode credit grant response: %v", err)
	}
	return decoded
}

func hasNotificationKind(t *testing.T, server *httptest.Server, accessToken string, kind string) bool {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/notifications", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var body struct {
		Notifications []struct {
			Kind             string `json:"kind"`
			ActorDisplayName string `json:"actor_display_name"`
			SubjectTitle     string `json:"subject_title"`
		} `json:"notifications"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode notifications: %v", err)
	}
	for _, notification := range body.Notifications {
		if notification.Kind == kind {
			return true
		}
	}
	return false
}

func TestWebhookMarketplaceAudience(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	operator := registerUser(t, server, "marketplace-hooks")

	// A marketplace subscription with both filters is accepted and echoed.
	created := createWebhookSubscriptionHTTP(t, server, operator.AccessToken,
		`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"marketplace","filter_task_type":"code_review","filter_min_credit_reward":25}`)
	if created.Subscription.Audience != "marketplace" || created.Subscription.FilterTaskType != "code_review" || created.Subscription.FilterMinCreditReward != 25 {
		t.Fatalf("marketplace subscription = %+v, want audience/filters echoed", created.Subscription)
	}

	// Marketplace subscriptions may listen only for task_opened.
	mixed := postJSONWithBearer(t, server.URL+"/api/webhook-subscriptions",
		[]byte(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened","payout_received"],"audience":"marketplace"}`), operator.AccessToken)
	defer mixed.Body.Close()
	assertStatus(t, mixed, http.StatusBadRequest)
	assertErrorCode(t, mixed, "invalid_argument")

	// The filters are valid only with the marketplace audience.
	recipientWithFilter := postJSONWithBearer(t, server.URL+"/api/webhook-subscriptions",
		[]byte(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"filter_task_type":"code_review"}`), operator.AccessToken)
	defer recipientWithFilter.Body.Close()
	assertStatus(t, recipientWithFilter, http.StatusBadRequest)
	assertErrorCode(t, recipientWithFilter, "invalid_argument")

	// The minimum reward must be positive.
	negativeReward := postJSONWithBearer(t, server.URL+"/api/webhook-subscriptions",
		[]byte(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"marketplace","filter_min_credit_reward":-5}`), operator.AccessToken)
	defer negativeReward.Body.Close()
	assertStatus(t, negativeReward, http.StatusBadRequest)

	// An unknown audience is an enum error.
	badAudience := postJSONWithBearer(t, server.URL+"/api/webhook-subscriptions",
		[]byte(`{"url":"https://receiver.example.com/hooks","kinds":["task_opened"],"audience":"broadcast"}`), operator.AccessToken)
	defer badAudience.Body.Close()
	assertStatus(t, badAudience, http.StatusBadRequest)
	assertErrorCode(t, badAudience, "invalid_enum")

	// Absent audience keeps the recipient default.
	recipient := createWebhookSubscriptionHTTP(t, server, operator.AccessToken,
		`{"url":"https://receiver.example.com/hooks","kinds":["payout_received"]}`)
	if recipient.Subscription.Audience != "recipient" || recipient.Subscription.FilterTaskType != "" || recipient.Subscription.FilterMinCreditReward != 0 {
		t.Fatalf("default subscription = %+v, want the recipient audience without filters", recipient.Subscription)
	}
}

type webhookSubscriptionHTTP struct {
	ID                    string `json:"id"`
	Audience              string `json:"audience"`
	FilterTaskType        string `json:"filter_task_type"`
	FilterMinCreditReward int64  `json:"filter_min_credit_reward"`
	State                 string `json:"state"`
}

type webhookCreatedHTTP struct {
	Subscription webhookSubscriptionHTTP `json:"subscription"`
	Secret       string                  `json:"secret"`
}

func createWebhookSubscriptionHTTP(t *testing.T, server *httptest.Server, accessToken string, body string) webhookCreatedHTTP {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/webhook-subscriptions", []byte(body), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	var created webhookCreatedHTTP
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatalf("decode webhook subscription: %v", err)
	}
	return created
}

func TestAccountProfileAndDisplayName(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	email := "profile-" + uniqueTestSuffix(t) + "@example.com"

	// Registration accepts an explicit display name and echoes it.
	registerResponse := postEncodedJSON(t, server.URL+"/api/auth/register",
		[]byte(`{"email":"`+email+`","password":"correct horse battery staple","display_name":"Mara Fields"}`), nil)
	assertStatus(t, registerResponse, http.StatusCreated)
	registered := decodeAuthHTTPResponse(t, registerResponse)
	refreshCookie := findRefreshCookie(t, registerResponse)
	registerResponse.Body.Close()
	if registered.DisplayName != "Mara Fields" {
		t.Fatalf("register display_name = %q, want Mara Fields", registered.DisplayName)
	}

	// The refreshed session names the signed-in user too.
	refreshResponse := postRefresh(t, server, refreshCookie)
	assertStatus(t, refreshResponse, http.StatusOK)
	refreshed := decodeAuthHTTPResponse(t, refreshResponse)
	refreshResponse.Body.Close()
	if refreshed.DisplayName != "Mara Fields" {
		t.Fatalf("refresh display_name = %q, want Mara Fields", refreshed.DisplayName)
	}

	// GET /api/account/profile reports id, email, and display name.
	profile := fetchAccountProfile(t, server, registered.AccessToken)
	if profile.ID != registered.SubjectID || profile.Email != email || profile.DisplayName != "Mara Fields" {
		t.Fatalf("profile = %+v, want id/email/display name of the registered account", profile)
	}

	// The display name can be replaced, and the change is visible.
	updateResponse := patchJSONWithBearer(t, server.URL+"/api/account/display-name", []byte(`{"display_name":"Mara F."}`), registered.AccessToken)
	defer updateResponse.Body.Close()
	assertStatus(t, updateResponse, http.StatusOK)
	if updated := fetchAccountProfile(t, server, registered.AccessToken); updated.DisplayName != "Mara F." {
		t.Fatalf("profile display_name after update = %q, want Mara F.", updated.DisplayName)
	}

	// A blank display name is rejected.
	blank := patchJSONWithBearer(t, server.URL+"/api/account/display-name", []byte(`{"display_name":"   "}`), registered.AccessToken)
	defer blank.Body.Close()
	assertStatus(t, blank, http.StatusBadRequest)

	// Login reports the updated name.
	loginResponse := postAuthJSON(t, server.URL+"/api/auth/login", authHTTPRequest{Email: email, Password: "correct horse battery staple"}, nil)
	assertStatus(t, loginResponse, http.StatusOK)
	loggedIn := decodeAuthHTTPResponse(t, loginResponse)
	loginResponse.Body.Close()
	if loggedIn.DisplayName != "Mara F." {
		t.Fatalf("login display_name = %q, want Mara F.", loggedIn.DisplayName)
	}

	// Registration without a display name derives it from the email.
	derivedEmail := "derived-" + uniqueTestSuffix(t) + "@example.com"
	derived := registerUserWithEmail(t, server, derivedEmail)
	if derived.DisplayName != localPart(derivedEmail) {
		t.Fatalf("derived display_name = %q, want %q", derived.DisplayName, localPart(derivedEmail))
	}

	// The user directory exposes display names.
	directoryResponse := getWithBearer(t, server.URL+"/api/users?query="+url.QueryEscape(derivedEmail), registered.AccessToken)
	defer directoryResponse.Body.Close()
	assertStatus(t, directoryResponse, http.StatusOK)
	var directory struct {
		Users []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"users"`
	}
	if err := json.NewDecoder(directoryResponse.Body).Decode(&directory); err != nil {
		t.Fatalf("decode directory: %v", err)
	}
	if len(directory.Users) != 1 || directory.Users[0].DisplayName != localPart(derivedEmail) {
		t.Fatalf("directory = %+v, want one entry naming %q", directory.Users, localPart(derivedEmail))
	}
}

type accountProfileHTTPResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func fetchAccountProfile(t *testing.T, server *httptest.Server, accessToken string) accountProfileHTTPResponse {
	t.Helper()
	response := getWithBearer(t, server.URL+"/api/account/profile", accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)
	var profile accountProfileHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&profile); err != nil {
		t.Fatalf("decode account profile: %v", err)
	}
	return profile
}

func TestReadModelsExposeDisplayNamesAndTitles(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	ownerEmail := "names-owner-" + uniqueTestSuffix(t) + "@example.com"
	workerEmail := "names-worker-" + uniqueTestSuffix(t) + "@example.com"
	owner := registerUserWithEmail(t, server, ownerEmail)
	worker := registerUserWithEmail(t, server, workerEmail)

	createResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	assertStatus(t, createResponse, http.StatusCreated)
	task := decodeTaskHTTPResponse(t, createResponse)
	createResponse.Body.Close()
	openTask(t, server, owner.AccessToken, task.ID)

	// Task detail names its creator.
	detailResponse := mustGet(t, server, worker.AccessToken, "/api/tasks/"+task.ID)
	detail := decodeTaskHTTPResponse(t, detailResponse)
	detailResponse.Body.Close()
	if detail.CreatorDisplayName != localPart(ownerEmail) {
		t.Fatalf("detail creator_display_name = %q, want %q", detail.CreatorDisplayName, localPart(ownerEmail))
	}

	// A reservation names its holder.
	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	assertStatus(t, reserveResponse, http.StatusCreated)
	reservation := decodeReservationHTTPResponse(t, reserveResponse)
	reserveResponse.Body.Close()
	if reservation.HolderDisplayName != localPart(workerEmail) {
		t.Fatalf("holder_display_name = %q, want %q", reservation.HolderDisplayName, localPart(workerEmail))
	}

	// A submission names its submitter in the owner's review listing.
	submitAuthenticated(t, server, worker.AccessToken, task.ID)
	submissionsResponse := mustGet(t, server, owner.AccessToken, "/api/tasks/"+task.ID+"/submissions")
	var submissions submissionsHTTPResponse
	if err := json.NewDecoder(submissionsResponse.Body).Decode(&submissions); err != nil {
		t.Fatalf("decode submissions: %v", err)
	}
	submissionsResponse.Body.Close()
	if len(submissions.Submissions) != 1 || submissions.Submissions[0].SubmitterDisplayName != localPart(workerEmail) {
		t.Fatalf("submissions = %+v, want one row naming %q", submissions.Submissions, localPart(workerEmail))
	}

	// A task comment names its author.
	commentResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+task.ID+"/comments", []byte(`{"body":"On it."}`), worker.AccessToken)
	assertStatus(t, commentResponse, http.StatusCreated)
	var comment struct {
		AuthorDisplayName string `json:"author_display_name"`
	}
	if err := json.NewDecoder(commentResponse.Body).Decode(&comment); err != nil {
		t.Fatalf("decode comment: %v", err)
	}
	commentResponse.Body.Close()
	if comment.AuthorDisplayName != localPart(workerEmail) {
		t.Fatalf("comment author_display_name = %q, want %q", comment.AuthorDisplayName, localPart(workerEmail))
	}

	// The owner's event feed names the acting worker and the task.
	feed := fetchEventFeedHTTP(t, server.URL, owner.AccessToken, "")
	foundEnrichedEvent := false
	for _, event := range feed.Events {
		if event.TaskID == task.ID && event.ActorUserID == worker.SubjectID {
			if event.ActorDisplayName != localPart(workerEmail) {
				t.Fatalf("event actor_display_name = %q, want %q", event.ActorDisplayName, localPart(workerEmail))
			}
			if event.TaskTitle == "" {
				t.Fatalf("event task_title is empty for task %s", task.ID)
			}
			foundEnrichedEvent = true
		}
	}
	if !foundEnrichedEvent {
		t.Fatalf("owner feed %+v has no enriched event for task %s", feed.Events, task.ID)
	}

	// The owner's notifications name the actor and the subject task.
	notificationsResponse := getWithBearer(t, server.URL+"/api/notifications", owner.AccessToken)
	defer notificationsResponse.Body.Close()
	assertStatus(t, notificationsResponse, http.StatusOK)
	var inbox struct {
		Notifications []struct {
			Kind             string `json:"kind"`
			SubjectKind      string `json:"subject_kind"`
			ActorDisplayName string `json:"actor_display_name"`
			SubjectTitle     string `json:"subject_title"`
		} `json:"notifications"`
	}
	if err := json.NewDecoder(notificationsResponse.Body).Decode(&inbox); err != nil {
		t.Fatalf("decode inbox: %v", err)
	}
	foundNamedNotification := false
	for _, notification := range inbox.Notifications {
		if notification.ActorDisplayName == localPart(workerEmail) {
			foundNamedNotification = true
			if notification.SubjectKind == "task" && notification.SubjectTitle == "" {
				t.Fatalf("task-subject notification is missing subject_title: %+v", notification)
			}
		}
	}
	if !foundNamedNotification {
		t.Fatalf("inbox %+v has no notification naming %q", inbox.Notifications, localPart(workerEmail))
	}
}

func assertErrorCode(t *testing.T, response *http.Response, wantCode string) {
	t.Helper()
	var body errorHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode error body: %v", err)
	}
	if body.Code != wantCode {
		t.Fatalf("error code = %q (%s), want %q", body.Code, body.Error, wantCode)
	}
}
