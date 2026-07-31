// Package appmux assembles the real internal/http routing table for the WASI
// app guest around the full set of live domain services, so the guest and the
// tests that check it against the native server build the exact same mux. The
// domain service graph itself is built by internal/appgraph — the same builder
// cmd/sharecrop serve and mcp-stdio use — so the guest passes the bridge-backed
// GuestStores and a test passes the real internal/db stores; the mux is
// identical either way. Only the storage adapter differs.
package appmux

import (
	"net/http"
	"testing/fstest"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/appgraph"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/event"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
)

// Stores is the full set of store interfaces the mux is built over: the
// domain stores appgraph consumes plus the RuntimeState services
// (internal/http) that are bridged so the pooled guest shares one
// Postgres-backed store instead of per-instance state.
type Stores struct {
	Auth          auth.Store
	Event         event.Store
	Webhook       webhook.Store
	Notification  notification.Store
	Organization  org.Store
	Task          task.Store
	Submission    submission.Store
	Ledger        ledger.Store
	Agent         agent.Store
	OrgCredential orgcred.Store
	Assets        assets.Store
	Audit         audit.Store
	// SavedQueueViews and the fields below are RuntimeState services
	// (internal/http), bridged so the pooled guest shares one Postgres-backed
	// store instead of per-instance state.
	SavedQueueViews    httpserver.SavedQueueViewService
	PlatformAdmins     httpserver.PlatformAdminService
	ModerationTriage   httpserver.ModerationTriageService
	Privacy            httpserver.PrivacyService
	IPRateLimiter      httpserver.RateLimiter
	SubjectRateLimiter httpserver.RateLimiter
	MCPSessions        httpserver.MCPSessionPersistence
}

// domainStores maps the flat store set onto appgraph's domain store set.
func domainStores(stores Stores) appgraph.Stores {
	return appgraph.Stores{
		Auth:          stores.Auth,
		Event:         stores.Event,
		Notification:  stores.Notification,
		Organization:  stores.Organization,
		Task:          stores.Task,
		Submission:    stores.Submission,
		Ledger:        stores.Ledger,
		Agent:         stores.Agent,
		OrgCredential: stores.OrgCredential,
		Assets:        stores.Assets,
		Audit:         stores.Audit,
	}
}

// New builds the full app mux over the given access-token secret and stores.
// A rejected graph build (an unusable access-token secret) yields the same
// unusable zero services the previous inline wiring produced, so behavior is
// unchanged; the serve and mcp-stdio commands check the same rejection
// explicitly before starting.
func New(secret auth.AccessTokenSecret, stores Stores) http.Handler {
	built, _ := appgraph.Build(secret, domainStores(stores)).(appgraph.GraphBuilt)
	graph := built.Value

	runtime := httpserver.DefaultRuntimeState(map[string]bool{})
	runtime.NotificationService = graph.NotificationService
	runtime.AuditService = graph.AuditService
	runtime.EventStore = stores.Event
	runtime.WebhookStore = stores.Webhook
	runtime.SavedQueueViews = stores.SavedQueueViews
	runtime.PlatformAdmins = stores.PlatformAdmins
	runtime.ModerationTriage = stores.ModerationTriage
	runtime.PrivacyService = stores.Privacy
	runtime.IPRateLimiter = stores.IPRateLimiter
	runtime.SubjectRateLimiter = stores.SubjectRateLimiter
	runtime.MCPSessions = httpserver.NewPersistedMCPHTTPSessionStore(stores.MCPSessions)

	return httpserver.NewWithRuntimeState(
		fstest.MapFS{},
		graph.AuthService,
		graph.TokenVerifier,
		graph.OrganizationService,
		graph.TaskService,
		graph.SubmissionService,
		graph.LedgerService,
		graph.AgentService,
		graph.OrgCredentialService,
		graph.AssetService,
		runtime,
	)
}
