package wasmdemo

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

type seedTestEnv struct {
	auth         auth.Service
	organization org.Service
	task         task.Service
	ledger       ledger.Service
	submission   submission.Service
	asset        assets.Service
	notification notification.Service
}

func newSeedTestEnv(t *testing.T) seedTestEnv {
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
	taskStore := NewTaskBrowserStore(storage, ids, systemTestClock{})
	taskService := task.NewService(taskStore, organizationService, agentService)
	ledgerService := ledger.NewService(NewLedgerBrowserStore(storage, ids))
	submissionService := submission.NewService(NewSubmissionBrowserStore(storage, ids), taskStore, organizationService)
	assetService := assets.NewService(NewAssetBrowserStore(storage, ids))
	notificationService := notification.NewService(NewNotificationBrowserStore(storage))

	return seedTestEnv{
		auth:         authService.Value,
		organization: organizationService,
		task:         taskService,
		ledger:       ledgerService,
		submission:   submissionService,
		asset:        assetService,
		notification: notificationService,
	}
}

func (env seedTestEnv) seed(ctx context.Context) SeedResult {
	return SeedDemoScenario(ctx, env.auth, env.organization, env.task, env.ledger, env.submission, env.asset, env.notification)
}

func TestSeedDemoScenarioSeedsFixedDemoCast(t *testing.T) {
	env := newSeedTestEnv(t)
	ctx := context.Background()

	result := env.seed(ctx)
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
	balanceResult := env.ledger.Balance(ctx, result.AdminUserID)
	balance, matched := balanceResult.(ledger.BalanceFound)
	if !matched {
		t.Fatalf("admin balance: want BalanceFound, got %#v", balanceResult)
	}
	// 100 signup grant - 30 escrowed for the fraud-signals task.
	if balance.Value.Spendable() != 70 {
		t.Fatalf("admin balance = %d, want 70", balance.Value.Spendable())
	}

	tasksResult := env.task.List(ctx, adminSubject, task.PublicListScope{ViewerID: result.AdminUserID}, task.ListFilters{}, testPage(t, 20, 0))
	listed, matched := tasksResult.(task.TasksListed)
	if !matched {
		t.Fatalf("list public tasks: want TasksListed, got %#v", tasksResult)
	}
	// 4 discovery tasks + mara's own approval ("Proofread") and review
	// ("Review 5 pull request diffs") tasks.
	if len(listed.Values) != 6 {
		t.Fatalf("public task count = %d, want 6", len(listed.Values))
	}

	organizationsResult := env.organization.ListOrganizations(ctx, adminSubject, "", testPage(t, 20, 0))
	organizations, matched := organizationsResult.(org.OrganizationsListed)
	if !matched {
		t.Fatalf("list organizations: want OrganizationsListed, got %#v", organizationsResult)
	}
	if len(organizations.Values) != 1 || organizations.Values[0].Name.String() != "Field Operations" {
		t.Fatalf("organizations = %+v, want one named Field Operations", organizations.Values)
	}

	// The review-side seed gives mara a pending submission to review, an inbox
	// notification, and a collectible in her holdings - so the single-actor
	// demo can exercise the review/approve/collectible journeys.
	notificationsResult := env.notification.List(ctx, result.AdminUserID, testPage(t, 20, 0))
	notifications, matched := notificationsResult.(notification.NotificationsListed)
	if !matched {
		t.Fatalf("list notifications: want Listed, got %#v", notificationsResult)
	}
	if len(notifications.Values) != 1 {
		t.Fatalf("notification count = %d, want 1", len(notifications.Values))
	}

	collectiblesResult := env.asset.ListCollectibles(ctx, result.AdminUserID, testPage(t, 20, 0))
	collectibles, matched := collectiblesResult.(assets.CollectiblesListed)
	if !matched {
		t.Fatalf("list collectibles: want CollectiblesListed, got %#v", collectiblesResult)
	}
	if len(collectibles.Values) != 1 {
		t.Fatalf("collectible count = %d, want 1", len(collectibles.Values))
	}
}

func TestSeedDemoScenarioSecondRunLogsInInsteadOfReseeding(t *testing.T) {
	env := newSeedTestEnv(t)
	ctx := context.Background()

	first := env.seed(ctx)
	if first.Err != "" {
		t.Fatalf("first seed: %s", first.Err)
	}
	second := env.seed(ctx)
	if second.Err != "" {
		t.Fatalf("second seed: %s", second.Err)
	}
	if second.AdminUserID != first.AdminUserID {
		t.Fatalf("second seed admin id = %v, want same as first %v (login, not re-register)", second.AdminUserID, first.AdminUserID)
	}
	if second.AdminRefreshToken == nil || second.AdminRefreshToken.Value == "" {
		t.Fatalf("second seed result missing a usable refresh cookie: %+v", second)
	}

	// Re-seeding would have duplicated the fixed task cast; still exactly 5.
	adminSubject := auth.UserSubject{ID: first.AdminUserID}
	tasksResult := env.task.List(ctx, adminSubject, task.PublicListScope{ViewerID: first.AdminUserID}, task.ListFilters{}, testPage(t, 20, 0))
	listed, matched := tasksResult.(task.TasksListed)
	if !matched {
		t.Fatalf("list public tasks: want TasksListed, got %#v", tasksResult)
	}
	if len(listed.Values) != 6 {
		t.Fatalf("public task count after second seed = %d, want 6 (no duplication)", len(listed.Values))
	}
}
