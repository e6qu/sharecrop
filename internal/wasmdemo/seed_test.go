package wasmdemo

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
)

func newSeedTestEnv(t *testing.T) (auth.Service, org.Service, task.Service, ledger.Service) {
	t.Helper()
	storage := newTestBrowserStorage()
	ids := &counterLedgerIDs{}

	authResult := auth.NewService(NewAuthBrowserStore(storage, ids), testAccessTokenSecret(t), fixedTestClock{now: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)})
	authService, matched := authResult.(auth.ServiceCreated)
	if !matched {
		t.Fatalf("new auth service failed: %#v", authResult)
	}

	organizationService := org.NewService(NewOrgBrowserStore(storage, ids))
	agentService := agent.NewService(NewAgentBrowserStore(storage))
	taskStore := NewTaskBrowserStore(storage, ids)
	taskService := task.NewService(taskStore, organizationService, agentService)
	ledgerService := ledger.NewService(NewLedgerBrowserStore(storage, ids))

	return authService.Value, organizationService, taskService, ledgerService
}

func TestSeedDemoScenarioSeedsFixedDemoCast(t *testing.T) {
	authService, organizationService, taskService, ledgerService := newSeedTestEnv(t)
	ctx := context.Background()

	result := SeedDemoScenario(ctx, authService, organizationService, taskService, ledgerService)
	if result.Err != "" {
		t.Fatalf("seed demo scenario: %s", result.Err)
	}
	if result.AdminRefreshToken == nil || result.AdminRefreshToken.Value == "" {
		t.Fatalf("seed result missing a usable refresh cookie: %+v", result)
	}
	if result.AdminRefreshToken.Name != "sharecrop_refresh_token" {
		t.Fatalf("refresh cookie name = %q, want sharecrop_refresh_token", result.AdminRefreshToken.Name)
	}

	adminSubject := auth.UserSubject{ID: result.AdminUserID}
	balanceResult := ledgerService.Balance(ctx, result.AdminUserID)
	balance, matched := balanceResult.(ledger.BalanceFound)
	if !matched {
		t.Fatalf("admin balance: want BalanceFound, got %#v", balanceResult)
	}
	// 100 signup grant - 30 escrowed for the fraud-signals task.
	if balance.Value.Int64() != 70 {
		t.Fatalf("admin balance = %d, want 70", balance.Value.Int64())
	}

	tasksResult := taskService.List(ctx, adminSubject, task.PublicListScope{ViewerID: result.AdminUserID}, task.ListFilters{}, testPage(t, 20, 0))
	listed, matched := tasksResult.(task.TasksListed)
	if !matched {
		t.Fatalf("list public tasks: want TasksListed, got %#v", tasksResult)
	}
	if len(listed.Values) != 4 {
		t.Fatalf("public task count = %d, want 4", len(listed.Values))
	}

	organizationsResult := organizationService.ListOrganizations(ctx, adminSubject, "", testPage(t, 20, 0))
	organizations, matched := organizationsResult.(org.OrganizationsListed)
	if !matched {
		t.Fatalf("list organizations: want OrganizationsListed, got %#v", organizationsResult)
	}
	if len(organizations.Values) != 1 || organizations.Values[0].Name.String() != "Field Operations" {
		t.Fatalf("organizations = %+v, want one named Field Operations", organizations.Values)
	}
}

func TestSeedDemoScenarioSecondRunLogsInInsteadOfReseeding(t *testing.T) {
	authService, organizationService, taskService, ledgerService := newSeedTestEnv(t)
	ctx := context.Background()

	first := SeedDemoScenario(ctx, authService, organizationService, taskService, ledgerService)
	if first.Err != "" {
		t.Fatalf("first seed: %s", first.Err)
	}
	second := SeedDemoScenario(ctx, authService, organizationService, taskService, ledgerService)
	if second.Err != "" {
		t.Fatalf("second seed: %s", second.Err)
	}
	if second.AdminUserID != first.AdminUserID {
		t.Fatalf("second seed admin id = %v, want same as first %v (login, not re-register)", second.AdminUserID, first.AdminUserID)
	}
	if second.AdminRefreshToken == nil || second.AdminRefreshToken.Value == "" {
		t.Fatalf("second seed result missing a usable refresh cookie: %+v", second)
	}

	// Re-seeding would have duplicated the fixed task cast; still exactly 4.
	adminSubject := auth.UserSubject{ID: first.AdminUserID}
	tasksResult := taskService.List(ctx, adminSubject, task.PublicListScope{ViewerID: first.AdminUserID}, task.ListFilters{}, testPage(t, 20, 0))
	listed, matched := tasksResult.(task.TasksListed)
	if !matched {
		t.Fatalf("list public tasks: want TasksListed, got %#v", tasksResult)
	}
	if len(listed.Values) != 4 {
		t.Fatalf("public task count after second seed = %d, want 4 (no duplication)", len(listed.Values))
	}
}
