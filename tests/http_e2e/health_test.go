//go:build http_e2e

package http_e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/web"
)

func TestHealthEndpoint(t *testing.T) {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		t.Fatalf("static files: %v", err)
	}

	server := httptest.NewServer(httpserver.New(staticFiles, testAuthService(), testVerifier{}, testOrganizationService{}))
	defer server.Close()

	response, err := http.Get(server.URL + "/healthz")
	if err != nil {
		t.Fatalf("get health: %v", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
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
	return auth.RegisterAccepted{Subject: auth.UserSubject{ID: idCreated.Value}, AccessToken: testAccessToken(), RefreshToken: testRefreshToken()}
}

func (testAuth) Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.LoginAccepted{Subject: auth.UserSubject{ID: idCreated.Value}, AccessToken: testAccessToken(), RefreshToken: testRefreshToken()}
}

func (testAuth) Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.RefreshAccepted{Subject: auth.UserSubject{ID: idCreated.Value}, AccessToken: testAccessToken(), RefreshToken: testRefreshToken()}
}

func (testAuth) CreateGuest(context.Context) auth.GuestResult {
	idResult := core.NewGuestID()
	idCreated := idResult.(core.GuestIDCreated)
	return auth.GuestAccepted{Subject: auth.GuestSubject{ID: idCreated.Value}, AccessToken: testAccessToken(), RefreshToken: testRefreshToken()}
}

func (testVerifier) Verify(auth.AccessToken) auth.SubjectVerifyResult {
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	return auth.SubjectVerified{Value: auth.UserSubject{ID: idCreated.Value}}
}

func (testOrganizationService) CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult {
	return org.CreateOrganizationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testOrganizationService) ListOrganizations(context.Context, auth.UserSubject) org.ListOrganizationsResult {
	return org.OrganizationsListed{Values: []org.Organization{}}
}

func (testOrganizationService) ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult {
	return org.ProvisionMemberRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testOrganizationService) DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult {
	return org.DeactivateMemberRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testOrganizationService) CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult {
	return org.CreateTeamRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (testOrganizationService) ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID) org.ListTeamsResult {
	return org.OrganizationTeamsListed{Values: []org.Team{}}
}

func testAccessToken() auth.AccessToken {
	secretResult := auth.NewAccessTokenSecret("01234567890123456789012345678901")
	secretAccepted := secretResult.(auth.AccessTokenSecretAccepted)
	idResult := core.NewUserID()
	idCreated := idResult.(core.UserIDCreated)
	tokenResult := auth.SignAccessToken(secretAccepted.Value, auth.UserSubject{ID: idCreated.Value}, time.Unix(1_700_000_000, 0).UTC())
	tokenAccepted := tokenResult.(auth.AccessTokenAccepted)
	return tokenAccepted.Value
}

func testRefreshToken() auth.RefreshTokenPlain {
	tokenResult := auth.ParseRefreshTokenPlain("test-refresh-token")
	tokenAccepted := tokenResult.(auth.RefreshTokenPlainAccepted)
	return tokenAccepted.Value
}
