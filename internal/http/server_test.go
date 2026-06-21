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
	request := httptest.NewRequest(http.MethodPost, "/api/tasks/"+taskIDCreated.Value.String()+"/submissions", strings.NewReader(`{"response_json":"{\"answer\":\"done\"}","wallet_address":""}`))
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

func testHandler() http.Handler {
	return New(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{}, testTaskService{}, testSubmissionService{}, testLedgerService{})
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

func (testOrganizationService) ListOrganizations(context.Context, auth.UserSubject) org.ListOrganizationsResult {
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

func (testOrganizationService) CreateOrganizationTeam(_ context.Context, actor auth.UserSubject, organizationID core.OrganizationID, name org.TeamName) org.CreateTeamResult {
	teamIDResult := core.NewTeamID()
	teamIDCreated := teamIDResult.(core.TeamIDCreated)
	return org.TeamCreated{Value: org.Team{ID: teamIDCreated.Value, OrganizationID: organizationID, Name: name, CreatedBy: actor.ID}}
}

func (testOrganizationService) ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func (testTaskService) Create(_ context.Context, command task.CreateCommand) task.CreateResult {
	idResult := core.NewTaskID()
	idCreated := idResult.(core.TaskIDCreated)
	return task.TaskCreated{Value: task.Task{
		ID:             idCreated.Value,
		Owner:          command.Owner,
		Title:          command.Title,
		Description:    command.Description,
		State:          task.StateDraft,
		Visibility:     command.Visibility,
		Placement:      command.Placement,
		ResponseSchema: command.ResponseSchema,
		Payload:        command.Payload,
		CreatedBy:      command.Actor.ID,
	}}
}

func (testTaskService) Open(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) Cancel(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
}

func (testTaskService) List(context.Context, auth.UserSubject, task.ListScope) task.ListResult {
	return task.TasksListed{Values: []task.Task{}}
}

func (testTaskService) CreateCapabilityToken(context.Context, auth.UserSubject, core.TaskID) task.CreateCapabilityTokenResult {
	return task.CreateCapabilityTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "unused test task service")}
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
			Submitter:      command.Submitter,
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

func (testSubmissionService) ListForTask(context.Context, auth.UserSubject, core.TaskID) submission.ListResult {
	return submission.SubmissionsListed{Values: []submission.Submission{}}
}

func (testLedgerService) FundTask(_ context.Context, _ core.UserID, taskID core.TaskID, amount ledger.CreditAmount, _ ledger.IdempotencyKey) ledger.FundResult {
	return ledger.TaskFunded{Escrow: ledger.TaskEscrow{TaskID: taskID, Amount: amount, State: ledger.EscrowStateHeld}}
}

func (testLedgerService) AcceptSubmission(_ context.Context, _ core.UserID, taskID core.TaskID, submissionID core.SubmissionID, _ ledger.IdempotencyKey) ledger.AcceptResult {
	return ledger.SubmissionAccepted{TaskID: taskID, SubmissionID: submissionID, Payout: ledger.NoPayout{}}
}

func (testLedgerService) RefundTask(_ context.Context, _ core.UserID, taskID core.TaskID, _ ledger.IdempotencyKey) ledger.RefundResult {
	return ledger.TaskRefunded{Escrow: ledger.TaskEscrow{TaskID: taskID, State: ledger.EscrowStateRefunded}}
}

func (testLedgerService) Balance(context.Context, core.UserID) ledger.BalanceResult {
	return ledger.BalanceFound{Value: ledger.NewBalance(100)}
}

func (testLedgerService) ListEntries(context.Context, core.UserID) ledger.ListEntriesResult {
	return ledger.EntriesListed{Values: []ledger.LedgerEntry{}}
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
