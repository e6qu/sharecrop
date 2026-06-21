//go:build http_e2e

package http_e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/web"
)

func TestHealthEndpoint(t *testing.T) {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		t.Fatalf("static files: %v", err)
	}

	server := httptest.NewServer(httpserver.New(staticFiles, healthAuthService{}, healthVerifier{}, healthOrganizationService{}, healthTaskService{}))
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

type healthTaskService struct{}

type healthAuthService struct{}

type healthVerifier struct{}

type healthOrganizationService struct{}

func (healthAuthService) Register(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.RegisterResult {
	return auth.RegisterRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAuthService) Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult {
	return auth.LoginRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAuthService) Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult {
	return auth.RefreshRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAuthService) CreateGuest(context.Context) auth.GuestResult {
	return auth.GuestRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthVerifier) Verify(auth.AccessToken) auth.SubjectVerifyResult {
	return auth.SubjectVerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult {
	return org.CreateOrganizationRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) ListOrganizations(context.Context, auth.UserSubject) org.ListOrganizationsResult {
	return org.ListOrganizationsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult {
	return org.ProvisionMemberRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult {
	return org.DeactivateMemberRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult {
	return org.CreateTeamRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthOrganizationService) ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID) org.ListTeamsResult {
	return org.ListTeamsRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) Create(context.Context, task.CreateCommand) task.CreateResult {
	return task.CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) Open(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) Cancel(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult {
	return task.ChangeStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) List(context.Context, auth.UserSubject, task.ListScope) task.ListResult {
	return task.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) CreateCapabilityToken(context.Context, auth.UserSubject, core.TaskID) task.CreateCapabilityTokenResult {
	return task.CreateCapabilityTokenRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}
