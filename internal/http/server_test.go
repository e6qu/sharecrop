package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

func TestHealth(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestIndex(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	contentType := response.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("content type = %q, want html", contentType)
	}
}

func TestParseTaskListFiltersAcceptsSearchQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/tasks?query=queue&task_type=code_review&sort=title_asc", nil)

	result := parseTaskListFilters(request)
	accepted, matched := result.(taskListFiltersAccepted)
	if !matched {
		t.Fatalf("filters = %T, want taskListFiltersAccepted", result)
	}
	search, matched := accepted.value.Search.(task.SearchContains)
	if !matched {
		t.Fatalf("search filter = %T, want SearchContains", accepted.value.Search)
	}
	if search.Value.String() != "queue" {
		t.Fatalf("search query = %q, want queue", search.Value.String())
	}
	typeFilter, matched := accepted.value.Type.(task.TypeEquals)
	if !matched {
		t.Fatalf("type filter = %T, want TypeEquals", accepted.value.Type)
	}
	if typeFilter.Value.String() != "code_review" {
		t.Fatalf("task type = %q, want code_review", typeFilter.Value.String())
	}
	if accepted.value.Sort != task.SortTitleAsc {
		t.Fatalf("sort = %q, want title_asc", accepted.value.Sort.String())
	}
}

func TestParseTaskListFiltersRejectsBlankSearchQuery(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/tasks?query=+++", nil)

	result := parseTaskListFilters(request)
	if _, matched := result.(taskListFiltersRejected); !matched {
		t.Fatalf("filters = %T, want taskListFiltersRejected", result)
	}
}

func TestParsePageStrictRejectsInvalidLimit(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/tasks?limit=abc&offset=0", nil)

	result := parsePageStrict(request)
	if _, matched := result.(pageParseRejected); !matched {
		t.Fatalf("page = %T, want pageParseRejected", result)
	}
}

func TestParseTaskListFiltersRejectsInvalidSort(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/tasks?sort=random", nil)

	result := parseTaskListFilters(request)
	if _, matched := result.(taskListFiltersRejected); !matched {
		t.Fatalf("filters = %T, want taskListFiltersRejected", result)
	}
}

func TestRegisterEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/register", strings.NewReader(`{"email":"person@example.com","password":"correct horse battery staple"}`))
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	assertAuthResponse(t, response, "user")
	assertRefreshCookie(t, response)
}

func TestGuestEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/guest", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	assertAuthResponse(t, response, "guest")
	assertRefreshCookie(t, response)
}

func TestRefreshEndpointRequiresCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestLogoutClearsRefreshCookie(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	for _, cookie := range response.Result().Cookies() {
		if cookie.Name == "sharecrop_refresh_token" {
			if cookie.MaxAge >= 0 {
				t.Fatalf("refresh cookie max age = %d, want negative", cookie.MaxAge)
			}
			return
		}
	}
	t.Fatalf("refresh cookie was not cleared")
}

func TestCreateOrganizationEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/organizations", strings.NewReader(`{"name":"Sharecrop Labs"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body organizationResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Name != "Sharecrop Labs" {
		t.Fatalf("name = %q, want Sharecrop Labs", body.Name)
	}
}

func TestCreateOrganizationRequiresUserToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/organizations", strings.NewReader(`{"name":"Sharecrop Labs"}`))
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestPrivacyRequestEndpointReturnsQueuedResponse(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/privacy-requests", strings.NewReader(`{"kind":"data_export"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body privacyRequestResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Kind != "data_export" {
		t.Fatalf("kind = %q, want data_export", body.Kind)
	}
	if body.Status != "queued" {
		t.Fatalf("status = %q, want queued", body.Status)
	}
	if body.RequestedBy == "" {
		t.Fatalf("requested_by is empty")
	}
	if body.CreatedAt == "" {
		t.Fatalf("created_at is empty")
	}
	if body.ResolvedAt != "" {
		t.Fatalf("resolved_at = %q, want empty", body.ResolvedAt)
	}
	if body.RedactedFieldCount != 0 {
		t.Fatalf("redacted_field_count = %d, want 0", body.RedactedFieldCount)
	}
}

func TestPrivacyRequestEndpointRejectsInvalidKind(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/privacy-requests", strings.NewReader(`{"kind":"hard_delete"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAdminPrivacyRequestListAndResolve(t *testing.T) {
	t.Setenv("SHARECROP_ADMIN_USER_IDS", stableTestUserID.String())
	handler := testHandler()

	createRequest := httptest.NewRequest(http.MethodPost, "/api/privacy-requests", strings.NewReader(`{"kind":"data_export"}`))
	createRequest.Header.Set("Authorization", "Bearer test-access-token")
	createResponse := httptest.NewRecorder()
	handler.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d", createResponse.Code, http.StatusCreated)
	}
	var created privacyRequestResponse
	if err := json.NewDecoder(createResponse.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/admin/privacy-requests", nil)
	listRequest.Header.Set("Authorization", "Bearer test-access-token")
	listResponse := httptest.NewRecorder()
	handler.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}
	var listed privacyRequestsResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(listed.Requests) != 1 {
		t.Fatalf("privacy request count = %d, want 1", len(listed.Requests))
	}

	resolveRequest := httptest.NewRequest(http.MethodPost, "/api/admin/privacy-requests/"+created.ID+"/resolve", strings.NewReader(`{"resolution_note":"export generated"}`))
	resolveRequest.SetPathValue("privacy_request_id", created.ID)
	resolveRequest.Header.Set("Authorization", "Bearer test-access-token")
	resolveResponse := httptest.NewRecorder()
	handler.ServeHTTP(resolveResponse, resolveRequest)
	if resolveResponse.Code != http.StatusOK {
		t.Fatalf("resolve status = %d, want %d", resolveResponse.Code, http.StatusOK)
	}
	var resolved privacyRequestResponse
	if err := json.NewDecoder(resolveResponse.Body).Decode(&resolved); err != nil {
		t.Fatalf("decode resolve response: %v", err)
	}
	if resolved.Status != "resolved" {
		t.Fatalf("status = %q, want resolved", resolved.Status)
	}
	if resolved.ResolutionNote != "export generated" {
		t.Fatalf("resolution_note = %q, want export generated", resolved.ResolutionNote)
	}
	if resolved.ResolvedAt == "" {
		t.Fatalf("resolved_at is empty")
	}
	if !strings.Contains(resolved.ExportJSON, stableTestUserID.String()) {
		t.Fatalf("export_json = %q, want user id", resolved.ExportJSON)
	}
}

func TestSavedQueueViewsRoundTrip(t *testing.T) {
	handler := testHandler()

	saveRequest := httptest.NewRequest(http.MethodPost, "/api/saved-queue-views", strings.NewReader(`{"scope":"team_work","name":"Ready work","query":"review","state_filter":"ready","type_filter":"code_review","sort":"title_asc"}`))
	saveRequest.Header.Set("Authorization", "Bearer test-access-token")
	saveResponse := httptest.NewRecorder()

	handler.ServeHTTP(saveResponse, saveRequest)

	if saveResponse.Code != http.StatusOK {
		t.Fatalf("save status = %d, want %d", saveResponse.Code, http.StatusOK)
	}

	var saved savedQueueViewResponse
	if err := json.NewDecoder(saveResponse.Body).Decode(&saved); err != nil {
		t.Fatalf("decode saved view: %v", err)
	}
	if saved.Scope != "team_work" || saved.Name != "Ready work" || saved.Query != "review" || saved.StateFilter != "ready" || saved.TypeFilter != "code_review" || saved.Sort != "title_asc" {
		t.Fatalf("saved view = %+v, want team_work Ready work review ready code_review title_asc", saved)
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/api/saved-queue-views?scope=team_work", nil)
	listRequest.Header.Set("Authorization", "Bearer test-access-token")
	listResponse := httptest.NewRecorder()

	handler.ServeHTTP(listResponse, listRequest)

	if listResponse.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listResponse.Code, http.StatusOK)
	}

	var list savedQueueViewsResponse
	if err := json.NewDecoder(listResponse.Body).Decode(&list); err != nil {
		t.Fatalf("decode saved views: %v", err)
	}
	if len(list.Views) != 1 || list.Views[0].Name != "Ready work" {
		t.Fatalf("listed views = %+v, want saved Ready work view", list.Views)
	}
}

func TestSavedQueueViewsRejectInvalidScope(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/saved-queue-views", strings.NewReader(`{"scope":"unknown","name":"Ready work"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestOrganizationLedgerEndpoint(t *testing.T) {
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated)
	request := httptest.NewRequest(http.MethodGet, "/api/organizations/"+organizationID.Value.String()+"/credits/ledger", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body ledgerListResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode ledger response: %v", err)
	}
	if body.Entries == nil {
		t.Fatalf("entries is nil")
	}
}

func TestOrganizationAuditEndpoint(t *testing.T) {
	organizationID := core.NewOrganizationID().(core.OrganizationIDCreated)
	request := httptest.NewRequest(http.MethodGet, "/api/organizations/"+organizationID.Value.String()+"/audit-events", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body auditEventsResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode audit response: %v", err)
	}
	if body.Events == nil {
		t.Fatalf("events is nil")
	}
}

func TestCreateTaskEndpointUsesDefaultUserVisibility(t *testing.T) {
	userIDResult := core.NewUserID()
	userIDCreated := userIDResult.(core.UserIDCreated)
	requestBody := `{
		"owner":{"kind":"user","user_id":"` + userIDCreated.Value.String() + `","team_id":"","organization_id":""},
		"title":"Collect examples",
		"description":"Find small examples for schema tests.",
		"reward":{"kind":"none","credit_amount":0},
		"visibility":{"kind":"default","user_id":"","team_id":"","organization_id":""},
		"placement":{"kind":"standalone","series_id":"","series_title":"","series_position":0},
		"response_schema_json":"{\"kind\":\"freeform\"}",
		"payload":{"kind":"none","json":""}
	}`
	request := httptest.NewRequest(http.MethodPost, "/api/tasks", strings.NewReader(requestBody))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body taskResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.VisibilityKind != task.VisibilityKindUser.String() {
		t.Fatalf("visibility kind = %q, want user", body.VisibilityKind)
	}
}

func TestAuthenticatedSubmissionEndpointReturnsReceipt(t *testing.T) {
	taskIDResult := core.NewTaskID()
	taskIDCreated := taskIDResult.(core.TaskIDCreated)
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskIDCreated.Value.String()+"/submissions", strings.NewReader(`{"response_json":"{\"answer\":\"done\"}"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body submissionCreatedResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.ReceiptToken == "" {
		t.Fatalf("receipt token is empty")
	}
}

func TestCreditsBalanceEndpoint(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/api/credits/balance", nil)
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}

	var body balanceResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.SpendableCredits != 100 {
		t.Fatalf("spendable_credits = %d, want 100", body.SpendableCredits)
	}
	if body.AllocatedCredits != 0 {
		t.Fatalf("allocated_credits = %d, want 0", body.AllocatedCredits)
	}
}

func TestFundTaskEndpointReturnsTaskFund(t *testing.T) {
	taskIDCreated := core.NewTaskID().(core.TaskIDCreated)
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskIDCreated.Value.String()+"/funding", strings.NewReader(`{"amount":40,"idempotency_key":"fund-1"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body taskFundResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.CreditAmount != 40 {
		t.Fatalf("credit_amount = %d, want 40", body.CreditAmount)
	}
}

func TestFundTaskEndpointRejectsNonPositiveAmount(t *testing.T) {
	taskIDCreated := core.NewTaskID().(core.TaskIDCreated)
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskIDCreated.Value.String()+"/funding", strings.NewReader(`{"amount":0,"idempotency_key":"fund-1"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestCreateAgentCredentialReturnsSecret(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/agent-credentials", strings.NewReader(`{"label":"Local agent","scopes":["tasks_read","submissions_write"]}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}
	var body agentCredentialCreatedResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.HasPrefix(body.Secret, "scrop_agent_") {
		t.Fatalf("secret = %q, want scrop_agent_ prefix", body.Secret)
	}
	if len(body.Credential.Scopes) != 2 {
		t.Fatalf("scope count = %d, want 2", len(body.Credential.Scopes))
	}
}

func TestCreateAgentCredentialRejectsUnknownScope(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/api/agent-credentials", strings.NewReader(`{"label":"Local agent","scopes":["everything"]}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMCPEndpointRequiresAgentCredential(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func testHandler() http.Handler {
	return New(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testOrgCredentialService{}, testAssetService{})
}

func testStaticFiles() fs.FS {
	return fstest.MapFS{
		"index.html": {
			Data: []byte("<!doctype html><title>Sharecrop</title>"),
		},
	}
}

func assertAuthResponse(t *testing.T, response *httptest.ResponseRecorder, subjectKind string) {
	t.Helper()

	var body authResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if body.SubjectKind != subjectKind {
		t.Fatalf("subject kind = %q, want %q", body.SubjectKind, subjectKind)
	}

	if body.SubjectID == "" {
		t.Fatalf("subject id is empty")
	}

	if body.AccessToken == "" {
		t.Fatalf("access token is empty")
	}
}

func assertRefreshCookie(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	cookies := response.Result().Cookies()
	for _, cookie := range cookies {
		if cookie.Name == "sharecrop_refresh_token" && cookie.HttpOnly {
			return
		}
	}

	t.Fatalf("refresh cookie was not set")
}

type testAuth struct{}

type testVerifier struct{}

type testOrganizationService struct{}

type testTaskService struct{}

type testSubmissionService struct{}

type testLedgerService struct{}

var stableTestUserID = core.NewUserID().(core.UserIDCreated).Value

func testAuthService() testAuth {
	return testAuth{}
}

func (testAuth) Register(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.RegisterResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.RegisterAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.LoginAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) LoginExternal(context.Context, string, string, auth.EmailAddress) auth.ExternalLoginResult {
	id := core.NewUserID().(core.UserIDCreated).Value
	return auth.ExternalLoginAccepted{Subject: auth.UserSubject{ID: id}, AccessToken: testAccessToken(), RefreshToken: testRefreshToken()}
}

func (testAuth) Logout(context.Context, auth.RefreshTokenPlain) auth.LogoutResult {
	return auth.LogoutDone{}
}

func (testAuth) Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.RefreshAccepted{
		Subject:      auth.UserSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) ValidateSession(context.Context, auth.RefreshTokenPlain) auth.ValidateRefreshTokenResult {
	return auth.RefreshTokenActive{}
}

func (testAuth) CreateGuest(context.Context) auth.GuestResult {
	idResult := core.NewGuestID()
	idCreated := idResult.(core.GuestIDCreated)
	return auth.GuestAccepted{
		Subject:      auth.GuestSubject{ID: idCreated.Value},
		AccessToken:  testAccessToken(),
		RefreshToken: testRefreshToken(),
	}
}

func (testAuth) ListUsers(context.Context, string, core.Page) auth.UserDirectoryResult {
	return auth.UsersListed{Values: []auth.UserDirectoryEntry{}}
}

func (testAuth) RequestEmailVerification(context.Context, core.UserID) auth.AccountTokenIssueResult {
	return auth.AccountTokenIssueRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) VerifyEmail(context.Context, auth.AccountTokenPlain) auth.AccountActionResult {
	return auth.AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) RequestPasswordReset(context.Context, auth.EmailAddress) auth.AccountTokenIssueResult {
	return auth.AccountTokenIssueRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) ResetPassword(context.Context, auth.AccountTokenPlain, auth.PasswordSecret) auth.AccountActionResult {
	return auth.AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) ChangePassword(context.Context, core.UserID, auth.PasswordSecret, auth.PasswordSecret) auth.AccountActionResult {
	return auth.AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) UpdateProfile(context.Context, core.UserID, auth.EmailAddress) auth.AccountActionResult {
	return auth.AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testAuth) DeactivateAccount(context.Context, core.UserID) auth.AccountActionResult {
	return auth.AccountActionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testVerifier) Verify(auth.AccessToken) auth.SubjectVerifyResult {
	return auth.SubjectVerified{Value: auth.UserSubject{ID: stableTestUserID}}
}

func TestShauthProtectedAPIsRequireTheActiveBrowserSession(t *testing.T) {
	server := Server{authService: testAuth{}, subjectVerifier: testVerifier{}, requireBrowserSession: true}
	request := httptest.NewRequest(http.MethodGet, "/api/credits/balance", nil)
	request.Header.Set("Authorization", "Bearer "+testAccessToken().String())
	if _, accepted := server.requireUserSubject(request).(userSubjectAccepted); accepted {
		t.Fatal("bearer token without a Sharecrop browser session was accepted")
	}
	request.AddCookie(&http.Cookie{Name: "sharecrop_refresh_token", Value: testRefreshToken().String()})
	if _, accepted := server.requireUserSubject(request).(userSubjectAccepted); !accepted {
		t.Fatal("active browser session with a valid bearer token was rejected")
	}
}

func (testOrganizationService) CreateOrganization(_ context.Context, actor auth.UserSubject, name org.OrganizationName) org.CreateOrganizationResult {
	idResult := core.NewOrganizationID()
	idCreated := idResult.(core.OrganizationIDCreated)
	return org.OrganizationCreated{Value: org.Organization{ID: idCreated.Value, Name: name, CreatedBy: actor.ID}}
}

func (testOrganizationService) ListOrganizations(context.Context, auth.UserSubject, string, core.Page) org.ListOrganizationsResult {
	return org.OrganizationsListed{Values: []org.Organization{}}
}

func (testOrganizationService) ProvisionMember(context.Context, auth.Subject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult {
	membershipIDResult := core.NewOrganizationMembershipID()
	membershipIDCreated := membershipIDResult.(core.OrganizationMembershipIDCreated)
	userIDResult := core.NewUserID()
	userIDCreated := userIDResult.(core.UserIDCreated)
	organizationIDResult := core.NewOrganizationID()
	organizationIDCreated := organizationIDResult.(core.OrganizationIDCreated)
	return org.MemberProvisioned{Value: org.OrganizationMember{ID: membershipIDCreated.Value, OrganizationID: organizationIDCreated.Value, UserID: userIDCreated.Value, Status: org.MembershipStatusActive, Roles: []org.Role{org.RoleMember}}}
}

func (testOrganizationService) DeactivateMember(context.Context, auth.Subject, core.OrganizationID, core.UserID) org.DeactivateMemberResult {
	return org.MemberDeactivationAccepted{}
}

func (testOrganizationService) UpdateMemberRoles(_ context.Context, _ auth.Subject, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesResult {
	membershipIDResult := core.NewOrganizationMembershipID()
	membershipIDCreated := membershipIDResult.(core.OrganizationMembershipIDCreated)
	return org.MemberRolesUpdatedResult{Value: org.OrganizationMember{ID: membershipIDCreated.Value, OrganizationID: organizationID, UserID: userID, Status: org.MembershipStatusActive, Roles: roles}}
}

func (testOrganizationService) CreateOrganizationTeam(_ context.Context, actor auth.UserSubject, organizationID core.OrganizationID, name org.TeamName) org.CreateTeamResult {
	teamIDResult := core.NewTeamID()
	teamIDCreated := teamIDResult.(core.TeamIDCreated)
	return org.TeamCreated{Value: org.Team{ID: teamIDCreated.Value, Owner: org.OrganizationOwnedTeam{OrganizationID: organizationID}, Name: name, CreatedBy: actor.ID}}
}

func (testOrganizationService) ListMembers(context.Context, auth.UserSubject, core.OrganizationID, core.Page) org.ListMembersResult {
	return org.MembersListed{Values: []org.OrganizationMember{}}
}

func (testOrganizationService) CreateStandaloneTeam(_ context.Context, actor auth.UserSubject, name org.TeamName) org.CreateTeamResult {
	teamIDResult := core.NewTeamID()
	teamIDCreated := teamIDResult.(core.TeamIDCreated)
	return org.TeamCreated{Value: org.Team{ID: teamIDCreated.Value, Owner: org.UserOwnedTeam{OwnerUserID: actor.ID}, Name: name, CreatedBy: actor.ID}}
}

func (testOrganizationService) ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID, string, core.Page) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func (testOrganizationService) ListStandaloneTeams(context.Context, auth.UserSubject, string, core.Page) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func (testOrganizationService) GetTeam(context.Context, auth.Subject, core.TeamID) org.GetTeamResult {
	return org.GetTeamRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "team not found")}
}

func (testOrganizationService) AddTeamMember(context.Context, auth.Subject, core.TeamID, auth.EmailAddress) org.AddTeamMemberResult {
	return org.TeamMemberAddedResult{}
}

func (testOrganizationService) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (testOrganizationService) CheckOrganizationTeamMembership(context.Context, core.OrganizationID, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (testOrganizationService) CheckTeamMembership(context.Context, core.TeamID, core.UserID) org.PermissionCheck {
	return org.PermissionGranted{}
}

func (testTaskService) Create(_ context.Context, command task.CreateCommand) task.CreateResult {
	idResult := core.NewTaskID()
	idCreated := idResult.(core.TaskIDCreated)
	return task.TaskCreated{Value: task.Task{
		ID:             idCreated.Value,
		Owner:          command.Owner,
		Title:          command.Title,
		Description:    command.Description,
		Reward:         command.Reward,
		Participation:  command.Participation,
		AssigneeScope:  command.AssigneeScope,
		ReservationTTL: command.ReservationTTL,
		State:          task.StateDraft,
		Visibility:     command.Visibility,
		Placement:      command.Placement,
		ResponseSchema: command.ResponseSchema,
		Payload:        command.Payload,
		Attachments:    command.Attachments,
		CreatedBy:      command.Actor.ID,
	}}
}

func (testTaskService) Get(_ context.Context, actor auth.Subject, taskID core.TaskID) task.GetResult {
	userActor, _ := actor.(auth.UserSubject)
	return task.TaskGot{Value: task.Task{
		ID:             taskID,
		Owner:          task.UserOwner{UserID: userActor.ID},
		Participation:  task.ParticipationPolicyOpen,
		AssigneeScope:  task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(),
		State:          task.StateOpen,
		Visibility:     task.UserVisibility{UserID: userActor.ID},
		Placement:      task.StandalonePlacement{},
		Payload:        task.NoDataPayload{},
		CreatedBy:      userActor.ID,
	}}
}

func (testTaskService) Open(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) Cancel(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) Unpublish(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) CreateSeries(context.Context, auth.UserSubject, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) UpdateSeries(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ChangeSeriesState(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesStateTransition) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) AddTaskToSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) RemoveTaskFromSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ReorderSeries(context.Context, auth.UserSubject, core.TaskSeriesID, []core.TaskID) task.SeriesMutationResult {
	return task.SeriesMutationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) AddSeriesComment(context.Context, auth.UserSubject, core.TaskSeriesID, task.CommentBody) task.SeriesCommentResult {
	return task.SeriesCommentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ListSeriesComments(context.Context, auth.UserSubject, core.TaskSeriesID) task.SeriesCommentsResult {
	return task.SeriesCommentsListed{Values: nil}
}

func (testTaskService) AddTaskComment(context.Context, auth.UserSubject, core.TaskID, task.CommentBody) task.TaskCommentResult {
	return task.TaskCommentRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ListTaskComments(context.Context, auth.UserSubject, core.TaskID) task.TaskCommentsResult {
	return task.TaskCommentsListed{Values: nil}
}

func (testTaskService) List(context.Context, auth.Subject, task.ListScope, task.ListFilters, core.Page) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (testTaskService) Reserve(_ context.Context, actor auth.UserSubject, taskID core.TaskID) task.ReservationResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationCreated{Value: task.Reservation{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.UserAssignee{UserID: actor.ID},
		State:       task.ReservationStateActive,
		RequestedBy: actor.ID,
	}}
}

func (testTaskService) ReserveForOrganizationTeam(_ context.Context, actor auth.UserSubject, taskID core.TaskID, organizationID core.OrganizationID, teamID core.TeamID) task.ReservationResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationCreated{Value: task.Reservation{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.OrganizationTeamAssignee{OrganizationID: organizationID, TeamID: teamID},
		State:       task.ReservationStateActive,
		RequestedBy: actor.ID,
	}}
}

func (testTaskService) ReserveForTeam(_ context.Context, actor auth.UserSubject, taskID core.TaskID, teamID core.TeamID) task.ReservationResult {
	reservationID := core.NewTaskReservationID().(core.TaskReservationIDCreated)
	return task.ReservationCreated{Value: task.Reservation{
		ID:          reservationID.Value,
		TaskID:      taskID,
		Assignee:    task.TeamAssignee{TeamID: teamID},
		State:       task.ReservationStateActive,
		RequestedBy: actor.ID,
	}}
}

func (testTaskService) ApproveReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) DeclineReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) CancelReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ListReservations(context.Context, auth.Subject, core.TaskID) task.ReservationsListResult {
	return task.ReservationsListed{Values: []task.Reservation{}}
}

func (testTaskService) ListSeries(context.Context, auth.UserSubject, core.Page) task.ListSeriesResult {
	return task.SeriesListed{Values: []task.Series{}}
}

func (testTaskService) GetSeries(context.Context, auth.Subject, core.TaskSeriesID) task.GetSeriesResult {
	return task.GetSeriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testSubmissionService) Submit(_ context.Context, command submission.SubmitCommand) submission.SubmitResult {
	idResult := core.NewSubmissionID()
	idCreated := idResult.(core.SubmissionIDCreated)
	tokenResult := submission.NewReceiptTokenPlain()
	tokenCreated := tokenResult.(submission.ReceiptTokenPlainAccepted)
	return submission.SubmissionCreated{
		Value: submission.Submission{
			ID:             idCreated.Value,
			TaskID:         command.TaskID,
			SubmitterID:    command.SubmitterID,
			State:          submission.StateSubmitted,
			ResponseSource: command.ResponseSource,
			Attachments:    command.Attachments,
			Validation:     submission.ValidationPassed{},
		},
		ReceiptToken: tokenCreated.Value,
	}
}

func (testSubmissionService) FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return submission.ReceiptStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test submission service")}
}

func (testSubmissionService) Get(_ context.Context, actor auth.Subject, submissionID core.SubmissionID) submission.GetResult {
	userActor, _ := actor.(auth.UserSubject)
	taskIDCreated := core.NewTaskID().(core.TaskIDCreated)
	sourceResult := submission.NewResponseSource("{}")
	sourceAccepted := sourceResult.(submission.ResponseSourceAccepted)
	return submission.SubmissionGot{Value: submission.Submission{
		ID:             submissionID,
		TaskID:         taskIDCreated.Value,
		SubmitterID:    userActor.ID,
		State:          submission.StateSubmitted,
		ResponseSource: sourceAccepted.Value,
		Validation:     submission.ValidationPassed{},
	}}
}

func (testSubmissionService) ListForTask(context.Context, auth.Subject, core.TaskID, core.Page) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (testSubmissionService) ListForSubmitter(context.Context, auth.UserSubject, core.UserID, core.Page) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (testSubmissionService) AddSubmissionComment(_ context.Context, actor auth.UserSubject, submissionID core.SubmissionID, body task.CommentBody) submission.SubmissionCommentResult {
	commentID := core.NewSubmissionCommentID().(core.SubmissionCommentIDCreated)
	taskID := core.NewTaskID().(core.TaskIDCreated)
	return submission.SubmissionCommentAdded{
		Value:         submission.SubmissionComment{ID: commentID.Value, SubmissionID: submissionID, AuthorID: actor.ID, Body: body},
		TaskID:        taskID.Value,
		SubmitterID:   actor.ID,
		TaskCreatorID: actor.ID,
	}
}

func (testSubmissionService) ListSubmissionComments(context.Context, auth.Subject, core.SubmissionID) submission.SubmissionCommentsResult {
	return submission.SubmissionCommentsListed{Values: []submission.SubmissionComment{}}
}

func (testLedgerService) FundTask(_ context.Context, _ core.UserID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Fund: ledger.TaskFund{TaskID: taskID, CreditAmount: amount}}
}

func (testLedgerService) FundTaskFromOrganization(_ context.Context, _ core.OrganizationID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Fund: ledger.TaskFund{TaskID: taskID, CreditAmount: amount}}
}

func (testLedgerService) OrganizationBalance(context.Context, core.OrganizationID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(100, 0)}
}

func (testLedgerService) AcceptSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (testLedgerService) ReviewAcceptSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.CollectibleTipSelection) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (testLedgerService) RequestChanges(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, note submission.ReviewNote) ledger.RequestChangesResult {
	return ledger.ChangesRequested{TaskID: taskID, SubmissionID: submissionID, ReviewNote: note.String()}
}

func (testLedgerService) RejectSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey, _ submission.ReviewNote, _ ledger.CreditReviewSelection, _ ledger.TipSelection, _ ledger.BanSelection) ledger.RejectResult {
	return ledger.SubmissionRejected{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}, Tip: ledger.NoTip{}}
}

func (testLedgerService) RefundTask(_ context.Context, _ core.UserID, taskID core.TaskID, _ ledger.IdempotencyKey) ledger.RefundResult {
	return ledger.TaskRefunded{Fund: ledger.TaskFund{TaskID: taskID, CreditAmount: ledger.NewCreditAmount(1).(ledger.CreditAmountAccepted).Value}}
}

func (testLedgerService) TaskAllocatedCredits(context.Context, core.TaskID) ledger.TaskAllocatedResult {
	return ledger.TaskAllocatedFound{Amount: 0}
}

func (testLedgerService) Balance(context.Context, core.UserID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(100, 0)}
}

func (testLedgerService) ListEntries(context.Context, core.UserID, core.Page) ledger.ListEntriesResult {
	return ledger.EntriesListed{Values: []ledger.LedgerEntry{}}
}

func (testLedgerService) ListOrganizationEntries(context.Context, core.OrganizationID, core.Page) ledger.ListEntriesResult {
	return ledger.EntriesListed{Values: []ledger.LedgerEntry{}}
}

type testAgentService struct{}

func (testAgentService) Create(_ context.Context, owner core.UserID, label agent.Label, scopes agent.ScopeSet, expiresAt *time.Time, taskID *core.TaskID) agent.CreateResult {
	idCreated := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	secretCreated := agent.NewSecretPlain().(agent.SecretPlainAccepted)
	return agent.CredentialCreated{
		Value:  agent.Credential{ID: idCreated.Value, UserID: owner, Label: label, Scopes: scopes, State: agent.StateActive, ExpiresAt: expiresAt, TaskID: taskID},
		Secret: secretCreated.Value,
	}
}

func (testAgentService) Verify(context.Context, agent.SecretPlain) agent.VerifyResult {
	idCreated := core.NewUserID().(core.UserIDCreated)
	return agent.VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test agent service for user "+idCreated.Value.String())}
}

func (testAgentService) List(context.Context, core.UserID, core.Page) agent.ListResult {
	return agent.CredentialsListed{Values: []agent.Credential{}}
}

type testOrgCredentialService struct{}

func (testOrgCredentialService) Create(_ context.Context, organizationID core.OrganizationID, label agent.Label, scopes agent.ScopeSet, expiresAt *time.Time) orgcred.CreateResult {
	idCreated := core.NewOrgCredentialID().(core.OrgCredentialIDCreated)
	secretCreated := orgcred.NewSecretPlain().(orgcred.SecretPlainAccepted)
	return orgcred.CredentialCreated{
		Value:  orgcred.Credential{ID: idCreated.Value, OrganizationID: organizationID, Label: label, Scopes: scopes, State: agent.StateActive, ExpiresAt: expiresAt},
		Secret: secretCreated.Value,
	}
}

func (testOrgCredentialService) Verify(context.Context, orgcred.SecretPlain) orgcred.VerifyResult {
	return orgcred.VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test org credential service")}
}

func (testOrgCredentialService) List(context.Context, core.OrganizationID, core.Page) orgcred.ListResult {
	return orgcred.CredentialsListed{Values: []orgcred.Credential{}}
}

func (testOrgCredentialService) Revoke(_ context.Context, organizationID core.OrganizationID, id core.OrgCredentialID) orgcred.RevokeResult {
	labelAccepted := agent.NewLabel("Test org").(agent.LabelAccepted)
	return orgcred.CredentialRevoked{Value: orgcred.Credential{ID: id, OrganizationID: organizationID, Label: labelAccepted.Value, Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), State: agent.StateRevoked}}
}

type testAssetService struct{}

func (testAssetService) Mint(_ context.Context, ownerKind string, ownerID string, organizationID string, name assets.CollectibleName, kind assets.CollectibleKind, policy assets.TransferPolicy, art string) assets.MintResult {
	idCreated := core.NewCollectibleID().(core.CollectibleIDCreated)
	return assets.CollectibleMinted{Value: assets.Collectible{ID: idCreated.Value, Name: name, Kind: kind, State: assets.CollectibleStateMinted, Policy: policy, OwnerKind: ownerKind, OwnerID: ownerID, OrganizationID: organizationID, Art: art}}
}

func (testAssetService) ListCollectibles(context.Context, core.UserID, core.Page) assets.ListResult {
	return assets.CollectiblesListed{Values: []assets.Collectible{}}
}

func (testAssetService) ListByOwner(context.Context, string, string, core.Page) assets.ListResult {
	return assets.CollectiblesListed{Values: []assets.Collectible{}}
}

func (testAssetService) FundReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult {
	return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test asset service")}
}

func (testAssetService) RefundReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult {
	return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test asset service")}
}

func (testAssetService) GiftCollectible(context.Context, core.UserID, core.UserID, core.CollectibleID) assets.GiftResult {
	return assets.CollectibleGifted{}
}

func (testAssetService) TaskHeldCollectibles(context.Context, core.TaskID) assets.TaskHeldCollectiblesResult {
	return assets.TaskHeldCollectiblesFound{IDs: nil}
}

func (testAssetService) AwardOrganizationCollectible(context.Context, core.OrganizationID, core.CollectibleID, core.UserID) assets.GiftResult {
	return assets.CollectibleGifted{}
}

func (testAgentService) Revoke(_ context.Context, owner core.UserID, id core.AgentCredentialID) agent.RevokeResult {
	labelAccepted := agent.NewLabel("Test agent").(agent.LabelAccepted)
	return agent.CredentialRevoked{Value: agent.Credential{ID: id, UserID: owner, Label: labelAccepted.Value, Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), State: agent.StateRevoked}}
}

func testAccessToken() auth.AccessToken {
	secretResult := auth.NewAccessTokenSecret("01234567890123456789012345678901")
	secretAccepted := secretResult.(auth.AccessTokenSecretAccepted)
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	tokenResult := auth.SignAccessToken(secretAccepted.Value, auth.UserSubject{ID: idCreated.Value}, timeForTest())
	tokenAccepted := tokenResult.(auth.AccessTokenAccepted)
	return tokenAccepted.Value
}

func testRefreshToken() auth.RefreshTokenPlain {
	tokenResult := auth.ParseRefreshTokenPlain("test-refresh-token")
	tokenAccepted := tokenResult.(auth.RefreshTokenPlainAccepted)
	return tokenAccepted.Value
}

func timeForTest() time.Time {
	return time.Unix(1_700_000_000, 0).UTC()
}
