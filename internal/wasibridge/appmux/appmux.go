// Package appmux assembles the real internal/http routing table for the WASI
// app guest around the full set of live domain services, so the guest and the
// tests that check it against the native server build the exact same mux. Every
// domain service is bound to a store interface, so the guest passes the
// bridge-backed GuestStores and a test passes the real internal/db stores - the
// mux is identical either way. This mirrors the service graph cmd/sharecrop
// serve builds; only the storage adapter differs.
package appmux

import (
	"net/http"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// Stores is the full set of domain store interfaces the mux is built over. The
// guest fills these with bridge GuestStores; a test fills them with the real
// internal/db stores. Because each is an interface, the assembled mux is
// identical either way.
type Stores struct {
	Auth          auth.Store
	Notification  notification.Store
	Organization  org.Store
	Task          task.Store
	Submission    submission.Store
	Ledger        ledger.Store
	Agent         agent.Store
	OrgCredential orgcred.Store
	Assets        assets.Store
	Audit         audit.Store
	// SavedQueueViews and PlatformAdmins are RuntimeState services (internal/http),
	// bridged so the pooled guest shares one Postgres-backed store instead of
	// per-instance state.
	SavedQueueViews  httpserver.SavedQueueViewService
	PlatformAdmins   httpserver.PlatformAdminService
	ModerationTriage httpserver.ModerationTriageService
}

// New builds the full app mux over the given access-token secret and stores.
// Access-token verification stays stateless (signature + clock, no store); the
// domain services are wired in the same dependency order cmd/sharecrop uses -
// org and agent services feed the task service; the task store and org service
// feed the submission service. The RuntimeState services that have no dedicated
// domain store (rate limiters, MCP sessions, saved queue views, privacy,
// platform admins, moderation triage) keep their in-memory defaults; the audit
// and notification services are overridden to run over the bridged stores.
func New(secret auth.AccessTokenSecret, stores Stores) http.Handler {
	verifier := auth.NewAccessTokenVerifier(secret, auth.SystemClock{})
	authService, _ := auth.NewService(stores.Auth, secret, auth.SystemClock{}).(auth.ServiceCreated)

	agentService := agent.NewService(stores.Agent)
	orgCredentialService := orgcred.NewService(stores.OrgCredential)
	organizationService := org.NewService(stores.Organization)
	taskService := task.NewService(stores.Task, organizationService, agentService)
	submissionService := submission.NewService(stores.Submission, stores.Task, organizationService)
	ledgerService := ledger.NewService(stores.Ledger)
	assetService := assets.NewService(stores.Assets)

	runtime := httpserver.DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = notification.NewService(stores.Notification)
	runtime.AuditService = audit.NewService(stores.Audit)
	runtime.SavedQueueViews = stores.SavedQueueViews
	runtime.PlatformAdmins = stores.PlatformAdmins
	runtime.ModerationTriage = stores.ModerationTriage

	return httpserver.NewWithRuntimeState(
		fstest.MapFS{},
		authService.Value,
		verifier,
		organizationService,
		taskService,
		submissionService,
		ledgerService,
		agentService,
		orgCredentialService,
		assetService,
		runtime,
	)
}
