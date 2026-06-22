//go:build http_e2e

package http_e2e_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/web"
)

func TestHealthEndpoint(t *testing.T) {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		t.Fatalf("static files: %v", err)
	}

	server := httptest.NewServer(httpserver.New(staticFiles, healthAuthService{}, healthVerifier{}, healthOrganizationService{}, healthTaskService{}, healthSubmissionService{}, healthLedgerService{}, healthAgentService{}, healthAssetService{}))
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

type healthSubmissionService struct{}

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

func (healthOrganizationService) CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck {
	return org.PermissionDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) Create(context.Context, task.CreateCommand) task.CreateResult {
	return task.CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) Get(context.Context, auth.UserSubject, core.TaskID) task.GetResult {
	return task.GetRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
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

func (healthTaskService) ListSeries(context.Context, auth.UserSubject) task.ListSeriesResult {
	return task.ListSeriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthTaskService) GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult {
	return task.GetSeriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthSubmissionService) Submit(context.Context, submission.SubmitCommand) submission.SubmitResult {
	return submission.SubmitRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthSubmissionService) FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return submission.ReceiptStatusRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthSubmissionService) ListForTask(context.Context, auth.UserSubject, core.TaskID) submission.ListResult {
	return submission.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

type healthLedgerService struct{}

func (healthLedgerService) FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult {
	return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) FundTaskFromOrganization(context.Context, core.OrganizationID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult {
	return ledger.FundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) OrganizationBalance(context.Context, core.OrganizationID) ledger.BalanceResult {
	return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) AcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey) ledger.AcceptResult {
	return ledger.AcceptRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) RefundTask(context.Context, core.UserID, core.TaskID, ledger.IdempotencyKey) ledger.RefundResult {
	return ledger.RefundRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) Balance(context.Context, core.UserID) ledger.BalanceResult {
	return ledger.BalanceRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthLedgerService) ListEntries(context.Context, core.UserID) ledger.ListEntriesResult {
	return ledger.ListEntriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

type healthAgentService struct{}

func (healthAgentService) Create(context.Context, core.UserID, agent.Label, agent.ScopeSet) agent.CreateResult {
	return agent.CreateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAgentService) Verify(context.Context, agent.SecretPlain) agent.VerifyResult {
	return agent.VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAgentService) List(context.Context, core.UserID) agent.ListResult {
	return agent.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

type healthAssetService struct{}

func (healthAssetService) Mint(context.Context, core.UserID, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy) assets.MintResult {
	return assets.MintRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAssetService) ListCollectibles(context.Context, core.UserID) assets.ListResult {
	return assets.ListRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAssetService) FundReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult {
	return assets.FundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAssetService) RefundReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult {
	return assets.RefundRewardRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}

func (healthAgentService) Revoke(context.Context, core.UserID, core.AgentCredentialID) agent.RevokeResult {
	return agent.RevokeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "not used")}
}
