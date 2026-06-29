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
	request := httptest.NewRequest(http.MethodGet, "/api/tasks?query=queue", nil)

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

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
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
	if body.Amount != 100 {
		t.Fatalf("amount = %d, want 100", body.Amount)
	}
}

func TestFundTaskEndpointReturnsHeldEscrow(t *testing.T) {
	taskIDCreated := core.NewTaskID().(core.TaskIDCreated)
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskIDCreated.Value.String()+"/funding", strings.NewReader(`{"amount":40,"idempotency_key":"fund-1"}`))
	request.Header.Set("Authorization", "Bearer test-access-token")
	response := httptest.NewRecorder()

	testHandler().ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusCreated)
	}

	var body taskEscrowResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Amount != 40 {
		t.Fatalf("amount = %d, want 40", body.Amount)
	}
	if body.State != "held" {
		t.Fatalf("state = %q, want held", body.State)
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
	return New(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{}, testAgentService{}, testAssetService{})
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
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.SubjectVerified{Value: auth.UserSubject{ID: idCreated.Value}}
}

func (testOrganizationService) CreateOrganization(_ context.Context, actor auth.UserSubject, name org.OrganizationName) org.CreateOrganizationResult {
	idResult := core.NewOrganizationID()
	idCreated := idResult.(core.OrganizationIDCreated)
	return org.OrganizationCreated{Value: org.Organization{ID: idCreated.Value, Name: name, CreatedBy: actor.ID}}
}

func (testOrganizationService) ListOrganizations(context.Context, auth.UserSubject, string, core.Page) org.ListOrganizationsResult {
	return org.OrganizationsListed{Values: []org.Organization{}}
}

func (testOrganizationService) ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult {
	membershipIDResult := core.NewOrganizationMembershipID()
	membershipIDCreated := membershipIDResult.(core.OrganizationMembershipIDCreated)
	userIDResult := core.NewUserID()
	userIDCreated := userIDResult.(core.UserIDCreated)
	organizationIDResult := core.NewOrganizationID()
	organizationIDCreated := organizationIDResult.(core.OrganizationIDCreated)
	return org.MemberProvisioned{Value: org.OrganizationMember{ID: membershipIDCreated.Value, OrganizationID: organizationIDCreated.Value, UserID: userIDCreated.Value, Status: org.MembershipStatusActive, Roles: []org.Role{org.RoleMember}}}
}

func (testOrganizationService) DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult {
	return org.MemberDeactivationAccepted{}
}

func (testOrganizationService) UpdateMemberRoles(_ context.Context, _ auth.UserSubject, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesResult {
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

func (testOrganizationService) GetTeam(context.Context, auth.UserSubject, core.TeamID) org.GetTeamResult {
	return org.GetTeamRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "team not found")}
}

func (testOrganizationService) AddTeamMember(context.Context, auth.UserSubject, core.TeamID, auth.EmailAddress) org.AddTeamMemberResult {
	return org.TeamMemberAddedResult{}
}

func (testOrganizationService) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
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
		CreatedBy:      command.Actor.ID,
	}}
}

func (testTaskService) Get(_ context.Context, actor auth.UserSubject, taskID core.TaskID) task.GetResult {
	return task.TaskGot{Value: task.Task{
		ID:             taskID,
		Owner:          task.UserOwner{UserID: actor.ID},
		Participation:  task.ParticipationPolicyOpen,
		AssigneeScope:  task.AssigneeScopeUser,
		ReservationTTL: task.DefaultReservationTTL(),
		State:          task.StateOpen,
		Visibility:     task.UserVisibility{UserID: actor.ID},
		Placement:      task.StandalonePlacement{},
		Payload:        task.NoDataPayload{},
		CreatedBy:      actor.ID,
	}}
}

func (testTaskService) Open(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) Cancel(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) Unpublish(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
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

func (testTaskService) List(context.Context, auth.UserSubject, task.ListScope, task.ListFilters, core.Page) task.ListResult {
	return task.TasksListed{Values: []task.ListItem{}}
}

func (testTaskService) CreateCapabilityToken(context.Context, auth.UserSubject, core.TaskID) task.CreateCapabilityTokenResult {
	return task.CreateCapabilityTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
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

func (testTaskService) ApproveReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) DeclineReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) CancelReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult {
	return task.ReservationStateChangeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) ListReservations(context.Context, auth.UserSubject, core.TaskID) task.ReservationsListResult {
	return task.ReservationsListed{Values: []task.Reservation{}}
}

func (testTaskService) ListSeries(context.Context, auth.UserSubject, core.Page) task.ListSeriesResult {
	return task.SeriesListed{Values: []task.Series{}}
}

func (testTaskService) GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult {
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
			Validation:     submission.ValidationPassed{},
		},
		ReceiptToken: tokenCreated.Value,
	}
}

func (testSubmissionService) FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return submission.ReceiptStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test submission service")}
}

func (testSubmissionService) Get(_ context.Context, actor auth.UserSubject, submissionID core.SubmissionID) submission.GetResult {
	taskIDCreated := core.NewTaskID().(core.TaskIDCreated)
	sourceResult := submission.NewResponseSource("{}")
	sourceAccepted := sourceResult.(submission.ResponseSourceAccepted)
	return submission.SubmissionGot{Value: submission.Submission{
		ID:             submissionID,
		TaskID:         taskIDCreated.Value,
		SubmitterID:    actor.ID,
		State:          submission.StateSubmitted,
		ResponseSource: sourceAccepted.Value,
		Validation:     submission.ValidationPassed{},
	}}
}

func (testSubmissionService) ListForTask(context.Context, auth.UserSubject, core.TaskID, core.Page) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (testSubmissionService) ListForSubmitter(context.Context, auth.UserSubject, core.UserID) submission.ListResult {
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

func (testSubmissionService) ListSubmissionComments(context.Context, auth.UserSubject, core.SubmissionID) submission.SubmissionCommentsResult {
	return submission.SubmissionCommentsListed{Values: []submission.SubmissionComment{}}
}

func (testLedgerService) FundTask(_ context.Context, _ core.UserID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{TaskID: taskID, Amount: amount, State: ledger.EscrowStateHeld}}
}

func (testLedgerService) FundTaskFromOrganization(_ context.Context, _ core.OrganizationID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{TaskID: taskID, Amount: amount, State: ledger.EscrowStateHeld}}
}

func (testLedgerService) OrganizationBalance(context.Context, core.OrganizationID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(100)}
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
	return ledger.TaskRefunded{Escrow: ledger.TaskEscrow{TaskID: taskID, State: ledger.EscrowStateRefunded}}
}

func (testLedgerService) Balance(context.Context, core.UserID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(100)}
}

func (testLedgerService) ListEntries(context.Context, core.UserID, core.Page) ledger.ListEntriesResult {
	return ledger.EntriesListed{Values: []ledger.LedgerEntry{}}
}

type testAgentService struct{}

func (testAgentService) Create(_ context.Context, owner core.UserID, label agent.Label, scopes agent.ScopeSet) agent.CreateResult {
	idCreated := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	secretCreated := agent.NewSecretPlain().(agent.SecretPlainAccepted)
	return agent.CredentialCreated{
		Value:  agent.Credential{ID: idCreated.Value, UserID: owner, Label: label, Scopes: scopes, State: agent.StateActive},
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
