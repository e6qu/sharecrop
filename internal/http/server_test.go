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
	"github.com/e6qu/sharecrop/internal/org"
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

func testHandler() http.Handler {
	return New(testStaticFiles(), testAuthService(), testVerifier{}, testOrganizationService{})
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
