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

func TestReservationRequiredTaskDiscoveryAndSubmission(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "reservation-owner")
	worker := registerUser(t, server, "reservation-worker")
	other := registerUser(t, server, "reservation-other")

	createTaskResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicReservationTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
	defer createTaskResponse.Body.Close()
	assertStatus(t, createTaskResponse, http.StatusCreated)
	taskBody := decodeTaskHTTPResponse(t, createTaskResponse)
	openTask(t, server, owner.AccessToken, taskBody.ID)

	ownerTasksResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user", owner.AccessToken)
	defer ownerTasksResponse.Body.Close()
	assertStatus(t, ownerTasksResponse, http.StatusOK)
	assertTaskPresent(t, decodeTasksHTTPResponse(t, ownerTasksResponse), taskBody.ID)

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)
	reservationBody := decodeReservationHTTPResponse(t, reserveResponse)
	if reservationBody.State != "active" {
		t.Fatalf("reservation state = %q, want active", reservationBody.State)
	}
	if reservationBody.AssigneeID != worker.SubjectID {
		t.Fatalf("reservation assignee = %q, want %q", reservationBody.AssigneeID, worker.SubjectID)
	}

	otherSubmitResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskBody.ID+"/submissions", []byte(`{"response_json":"{\"answer\":\"done\"}"}`), other.AccessToken)
	defer otherSubmitResponse.Body.Close()
	assertStatus(t, otherSubmitResponse, http.StatusForbidden)

	otherListResponse := getWithBearer(t, server.URL+"/api/tasks?scope=public", other.AccessToken)
	defer otherListResponse.Body.Close()
	assertStatus(t, otherListResponse, http.StatusOK)
	assertTaskAbsent(t, decodeTasksHTTPResponse(t, otherListResponse), taskBody.ID)

	includeReservedResponse := getWithBearer(t, server.URL+"/api/tasks?scope=public&include_reserved=true", other.AccessToken)
	defer includeReservedResponse.Body.Close()
	assertStatus(t, includeReservedResponse, http.StatusOK)
	assertTaskPresent(t, decodeTasksHTTPResponse(t, includeReservedResponse), taskBody.ID)

	workerListResponse := getWithBearer(t, server.URL+"/api/tasks?scope=public", worker.AccessToken)
	defer workerListResponse.Body.Close()
	assertStatus(t, workerListResponse, http.StatusOK)
	assertTaskPresent(t, decodeTasksHTTPResponse(t, workerListResponse), taskBody.ID)

	ownerListResponse := getWithBearer(t, server.URL+"/api/tasks?scope=public", owner.AccessToken)
	defer ownerListResponse.Body.Close()
	assertStatus(t, ownerListResponse, http.StatusOK)
	assertTaskPresent(t, decodeTasksHTTPResponse(t, ownerListResponse), taskBody.ID)

	submitAuthenticated(t, server, worker.AccessToken, taskBody.ID)
}

func TestReservationApprovalIsBoundToOwningTask(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	targetOwner := registerUser(t, server, "idor-target-owner")
	worker := registerUser(t, server, "idor-worker")
	attacker := registerUser(t, server, "idor-attacker")

	// Another requester's approval-policy task with a pending reservation from the worker.
	targetCreate := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicApprovalTaskRequestJSON(targetOwner.SubjectID)), targetOwner.AccessToken)
	defer targetCreate.Body.Close()
	assertStatus(t, targetCreate, http.StatusCreated)
	targetTask := decodeTaskHTTPResponse(t, targetCreate)
	openTask(t, server, targetOwner.AccessToken, targetTask.ID)

	reserve := postJSONWithBearer(t, server.URL+"/api/tasks/"+targetTask.ID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserve.Body.Close()
	assertStatus(t, reserve, http.StatusCreated)
	reservation := decodeReservationHTTPResponse(t, reserve)
	if reservation.State != "requested" {
		t.Fatalf("reservation state = %q, want requested", reservation.State)
	}

	// The attacker owns an unrelated task and tries to approve another requester's
	// reservation through it. Ownership of the attacker's task must not authorize
	// mutating a reservation that belongs to a different task.
	attackerCreate := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(publicApprovalTaskRequestJSON(attacker.SubjectID)), attacker.AccessToken)
	defer attackerCreate.Body.Close()
	assertStatus(t, attackerCreate, http.StatusCreated)
	attackerTask := decodeTaskHTTPResponse(t, attackerCreate)

	idorAttempt := postJSONWithBearer(t, server.URL+"/api/tasks/"+attackerTask.ID+"/reservations/"+reservation.ID+"/approve", []byte(`{}`), attacker.AccessToken)
	defer idorAttempt.Body.Close()
	if idorAttempt.StatusCode < 400 {
		t.Fatalf("cross-task reservation approval status = %d, want a client error", idorAttempt.StatusCode)
	}

	// The target reservation must still be pending, not force-approved.
	list := getWithBearer(t, server.URL+"/api/tasks/"+targetTask.ID+"/reservations", targetOwner.AccessToken)
	defer list.Body.Close()
	assertStatus(t, list, http.StatusOK)
	var listBody struct {
		Reservations []reservationHTTPResponse `json:"reservations"`
	}
	if err := json.NewDecoder(list.Body).Decode(&listBody); err != nil {
		t.Fatalf("decode reservations: %v", err)
	}
	for _, value := range listBody.Reservations {
		if value.ID == reservation.ID && value.State != "requested" {
			t.Fatalf("target reservation state = %q after IDOR attempt, want requested", value.State)
		}
	}
}

func TestTaskListPagination(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "pagination-owner")

	for index := 0; index < 5; index++ {
		createResponse := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(userTaskRequestJSON(owner.SubjectID)), owner.AccessToken)
		assertStatus(t, createResponse, http.StatusCreated)
		_ = decodeTaskHTTPResponse(t, createResponse)
		createResponse.Body.Close()
	}

	// The full ordered list establishes the deterministic order the pages slice into.
	fullResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user", owner.AccessToken)
	defer fullResponse.Body.Close()
	assertStatus(t, fullResponse, http.StatusOK)
	fullPage := decodeTasksHTTPResponse(t, fullResponse)
	if len(fullPage.Tasks) != 5 {
		t.Fatalf("full list count = %d, want 5", len(fullPage.Tasks))
	}

	firstPageResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user&limit=2", owner.AccessToken)
	defer firstPageResponse.Body.Close()
	assertStatus(t, firstPageResponse, http.StatusOK)
	firstPage := decodeTasksHTTPResponse(t, firstPageResponse)
	if len(firstPage.Tasks) != 2 {
		t.Fatalf("first page count = %d, want 2", len(firstPage.Tasks))
	}
	if firstPage.Tasks[0].ID != fullPage.Tasks[0].ID || firstPage.Tasks[1].ID != fullPage.Tasks[1].ID {
		t.Fatalf("first page = [%q %q], want [%q %q]", firstPage.Tasks[0].ID, firstPage.Tasks[1].ID, fullPage.Tasks[0].ID, fullPage.Tasks[1].ID)
	}

	secondPageResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user&limit=2&offset=2", owner.AccessToken)
	defer secondPageResponse.Body.Close()
	assertStatus(t, secondPageResponse, http.StatusOK)
	secondPage := decodeTasksHTTPResponse(t, secondPageResponse)
	if len(secondPage.Tasks) != 2 {
		t.Fatalf("second page count = %d, want 2", len(secondPage.Tasks))
	}
	if secondPage.Tasks[0].ID != fullPage.Tasks[2].ID || secondPage.Tasks[1].ID != fullPage.Tasks[3].ID {
		t.Fatalf("second page = [%q %q], want [%q %q]", secondPage.Tasks[0].ID, secondPage.Tasks[1].ID, fullPage.Tasks[2].ID, fullPage.Tasks[3].ID)
	}
}

func TestTaskListFiltersByStateAndParticipation(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "filter-owner")

	// One open task with open participation, one draft task with reservation participation.
	openTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, userTaskRequestJSON(owner.SubjectID))
	openTask(t, server, owner.AccessToken, openTaskID)
	draftTaskID := createUserTaskFromJSON(t, server, owner.AccessToken, publicReservationTaskRequestJSON(owner.SubjectID))

	openListing := decodeTasksHTTPResponse(t, mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user&state=open"))
	assertTaskPresent(t, openListing, openTaskID)
	assertTaskAbsent(t, openListing, draftTaskID)

	draftListing := decodeTasksHTTPResponse(t, mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user&state=draft"))
	assertTaskPresent(t, draftListing, draftTaskID)
	assertTaskAbsent(t, draftListing, openTaskID)

	reservationListing := decodeTasksHTTPResponse(t, mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user&participation_policy=reservation_required"))
	assertTaskPresent(t, reservationListing, draftTaskID)
	assertTaskAbsent(t, reservationListing, openTaskID)

	invalidResponse := getWithBearer(t, server.URL+"/api/tasks?scope=user&state=bogus", owner.AccessToken)
	defer invalidResponse.Body.Close()
	assertStatus(t, invalidResponse, http.StatusBadRequest)
}

func TestTaskListItemExposesActiveAssignee(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "active-assignee-owner")
	worker := registerUser(t, server, "active-assignee-worker")

	taskID := createUserTaskFromJSON(t, server, owner.AccessToken, publicReservationTaskRequestJSON(owner.SubjectID))
	openTask(t, server, owner.AccessToken, taskID)

	beforeReserve := findTaskInListing(t, decodeTasksHTTPResponse(t, mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user")), taskID)
	if beforeReserve.ActiveAssigneeKind != "" || beforeReserve.ActiveAssigneeID != "" {
		t.Fatalf("active assignee before reserve = (%q, %q), want empty", beforeReserve.ActiveAssigneeKind, beforeReserve.ActiveAssigneeID)
	}

	reserveResponse := postJSONWithBearer(t, server.URL+"/api/tasks/"+taskID+"/reservations", []byte(`{}`), worker.AccessToken)
	defer reserveResponse.Body.Close()
	assertStatus(t, reserveResponse, http.StatusCreated)

	afterReserve := findTaskInListing(t, decodeTasksHTTPResponse(t, mustGet(t, server, owner.AccessToken, "/api/tasks?scope=user")), taskID)
	if afterReserve.ActiveAssigneeKind != "user" {
		t.Fatalf("active assignee kind = %q, want user", afterReserve.ActiveAssigneeKind)
	}
	if afterReserve.ActiveAssigneeID != worker.SubjectID {
		t.Fatalf("active assignee id = %q, want %q", afterReserve.ActiveAssigneeID, worker.SubjectID)
	}
}

func TestPrivateTaskDoesNotLeakToOtherUsers(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "leak-owner")
	taskID := createUserTaskFromJSON(t, server, owner.AccessToken, userTaskRequestJSON(owner.SubjectID))
	outsider := registerUser(t, server, "leak-outsider")

	// The task detail endpoint denies a viewer without permission.
	detailResponse := getWithBearer(t, server.URL+"/api/tasks/"+taskID, outsider.AccessToken)
	defer detailResponse.Body.Close()
	assertStatus(t, detailResponse, http.StatusForbidden)

	// Public discovery never lists the private task.
	discoveryResponse := getWithBearer(t, server.URL+"/api/tasks?scope=public", outsider.AccessToken)
	defer discoveryResponse.Body.Close()
	assertStatus(t, discoveryResponse, http.StatusOK)
	for _, item := range decodeTasksHTTPResponse(t, discoveryResponse).Tasks {
		if item.ID == taskID {
			t.Fatalf("private task %s leaked into public discovery", taskID)
		}
	}

	// The owner's public profile never lists the private task.
	profileResponse := getWithBearer(t, server.URL+"/api/users/"+owner.SubjectID, outsider.AccessToken)
	defer profileResponse.Body.Close()
	assertStatus(t, profileResponse, http.StatusOK)
	var profile struct {
		Tasks []struct {
			ID string `json:"id"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(profileResponse.Body).Decode(&profile); err != nil {
		t.Fatalf("decode profile: %v", err)
	}
	for _, item := range profile.Tasks {
		if item.ID == taskID {
			t.Fatalf("private task %s leaked into the owner's public profile", taskID)
		}
	}
}

func TestUserProfileShowsOnlyPublicTasks(t *testing.T) {
	server := newAuthHTTPServer(t, t.Context())
	defer server.Close()

	owner := registerUser(t, server, "profile-owner")
	createPublicUserTask(t, server, owner)
	createUserTaskFromJSON(t, server, owner.AccessToken, userTaskRequestJSON(owner.SubjectID))

	// A different viewer reads the owner's profile and must see only the owner's
	// public task, never the private (default-visibility) one.
	viewer := registerUser(t, server, "profile-viewer")
	response := getWithBearer(t, server.URL+"/api/users/"+owner.SubjectID, viewer.AccessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusOK)

	var body struct {
		ID    string `json:"id"`
		Tasks []struct {
			Title string `json:"title"`
		} `json:"tasks"`
	}
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode user profile: %v", err)
	}
	if body.ID != owner.SubjectID {
		t.Fatalf("profile id = %q, want %q", body.ID, owner.SubjectID)
	}
	if len(body.Tasks) != 1 {
		t.Fatalf("profile task count = %d, want 1 (public only, no private leak)", len(body.Tasks))
	}
	if body.Tasks[0].Title != "Public agent task" {
		t.Fatalf("profile task = %q, want the public task", body.Tasks[0].Title)
	}
}

func createUserTaskFromJSON(t *testing.T, server *httptest.Server, accessToken string, requestJSON string) string {
	t.Helper()
	response := postJSONWithBearer(t, server.URL+"/api/tasks", []byte(requestJSON), accessToken)
	defer response.Body.Close()
	assertStatus(t, response, http.StatusCreated)
	return decodeTaskHTTPResponse(t, response).ID
}

func mustGet(t *testing.T, server *httptest.Server, accessToken string, path string) *http.Response {
	t.Helper()
	response := getWithBearer(t, server.URL+path, accessToken)
	t.Cleanup(func() { response.Body.Close() })
	assertStatus(t, response, http.StatusOK)
	return response
}

func findTaskInListing(t *testing.T, body tasksHTTPResponse, taskID string) taskHTTPResponse {
	t.Helper()
	for _, value := range body.Tasks {
		if value.ID == taskID {
			return value
		}
	}
	t.Fatalf("task %s was not present", taskID)
	return taskHTTPResponse{}
}

type taskHTTPResponse struct {
	ID                     string `json:"id"`
	State                  string `json:"state"`
	VisibilityKind         string `json:"visibility_kind"`
	RewardKind             string `json:"reward_kind"`
	RewardCreditAmount     int64  `json:"reward_credit_amount"`
	RewardCollectibleCount int    `json:"reward_collectible_count"`
	ParticipationPolicy    string `json:"participation_policy"`
	ActiveAssigneeKind     string `json:"active_assignee_kind"`
	ActiveAssigneeID       string `json:"active_assignee_id"`
}

type tasksHTTPResponse struct {
	Tasks []taskHTTPResponse `json:"tasks"`
}

type taskCapabilityTokenHTTPResponse struct {
	Token string `json:"token"`
}

type reservationHTTPResponse struct {
	ID           string `json:"id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
}

func userTaskRequestJSON(userID string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Review schema samples",
		"description":"Review response examples against the local schema parser.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func userCreditTaskRequestJSON(userID string, amount int64) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Credit reward task",
		"description":"Review response examples for a credit reward.",
		"reward":{"kind":"credit","credit_amount":` + strconv.FormatInt(amount, 10) + `},
		"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func userBundleTaskRequestJSON(userID string, amount int64) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Bundle reward task",
		"description":"Review response examples for a bundled reward.",
		"reward":{"kind":"bundle","credit_amount":` + strconv.FormatInt(amount, 10) + `},
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
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func publicReservationTaskRequestJSON(userID string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Reserve public task",
		"description":"Reserve before submitting a response.",
		"reward":{"kind":"none","credit_amount":0},
		"participation":{"policy":"reservation_required","assignee_scope":"user","reservation_expiry_hours":48},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func publicApprovalTaskRequestJSON(userID string) string {
	return `{
		"owner":{"kind":"user","user_id":"` + userID + `","team_id":"","organization_id":""},
		"title":"Approval public task",
		"description":"Request approval before submitting a response.",
		"reward":{"kind":"none","credit_amount":0},
		"participation":{"policy":"approval_required","assignee_scope":"user","reservation_expiry_hours":48},
		"visibility":{"kind":"public","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
}

func assertTaskPresent(t *testing.T, body tasksHTTPResponse, taskID string) {
	t.Helper()
	for _, value := range body.Tasks {
		if value.ID == taskID {
			return
		}
	}
	t.Fatalf("task %s was not present", taskID)
}

func assertTaskAbsent(t *testing.T, body tasksHTTPResponse, taskID string) {
	t.Helper()
	for _, value := range body.Tasks {
		if value.ID == taskID {
			t.Fatalf("task %s was present", taskID)
		}
	}
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

func decodeReservationHTTPResponse(t *testing.T, response *http.Response) reservationHTTPResponse {
	t.Helper()
	var body reservationHTTPResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode reservation response: %v", err)
	}
	if body.ID == "" {
		t.Fatalf("reservation id is empty")
	}
	return body
}
