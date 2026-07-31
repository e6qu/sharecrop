// Package appgraph builds the domain service graph over a set of store
// interfaces. Four programs consume the same graph — cmd/sharecrop serve,
// cmd/sharecrop mcp-stdio, the WASI app guest (through appmux), and the
// lifecycle runner — so the wiring lives in exactly one place. The package is
// guest-safe: it depends only on domain packages, never on pgx, wazero, or
// net/http serving concerns.
package appgraph

import (
	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// Stores is the full set of domain store interfaces the graph is built over.
// Production fills these with the real internal/db stores; the WASI guest
// fills them with bridge GuestStores. Because each is an interface, the
// assembled graph is identical either way.
type Stores struct {
	Auth          auth.Store
	Event         event.Store
	Notification  notification.Store
	Organization  org.Store
	Task          task.Store
	Submission    submission.Store
	Ledger        ledger.Store
	Agent         agent.Store
	OrgCredential orgcred.Store
	Assets        assets.Store
	Audit         audit.Store
}

// Graph is the assembled service layer.
type Graph struct {
	AuthService          auth.Service
	TokenVerifier        auth.AccessTokenVerifier
	OrganizationService  org.Service
	TaskService          task.Service
	SubmissionService    submission.Service
	LedgerService        ledger.Service
	AgentService         agent.Service
	OrgCredentialService orgcred.Service
	AssetService         assets.Service
	NotificationService  notification.Service
	AuditService         audit.Service
	EventRecorder        event.Recorder
}

type BuildResult interface {
	buildResult()
}

type GraphBuilt struct {
	Value Graph
}

type BuildRejected struct {
	Reason core.DomainError
}

func (GraphBuilt) buildResult() {}

func (BuildRejected) buildResult() {}

// Build wires the domain services in dependency order: org and agent services
// feed the task service; the task store and org service feed the submission
// service; the event recorder (over the event store and notification service)
// feeds every mutating service.
func Build(secret auth.AccessTokenSecret, stores Stores) BuildResult {
	authServiceResult := auth.NewService(stores.Auth, secret, auth.SystemClock{})
	authService, matched := authServiceResult.(auth.ServiceCreated)
	if !matched {
		return BuildRejected{Reason: authServiceResult.(auth.ServiceRejected).Reason}
	}

	notificationService := notification.NewService(stores.Notification)
	recorder := event.NewRecorder(stores.Event, notificationService)

	agentService := agent.NewService(stores.Agent)
	orgCredentialService := orgcred.NewService(stores.OrgCredential)
	organizationService := org.NewService(stores.Organization)
	taskService := task.NewService(stores.Task, organizationService, agentService, recorder)
	submissionService := submission.NewService(stores.Submission, stores.Task, organizationService, recorder)
	ledgerService := ledger.NewService(stores.Ledger, recorder)
	assetService := assets.NewService(stores.Assets, recorder)

	return GraphBuilt{Value: Graph{
		AuthService:          authService.Value,
		TokenVerifier:        auth.NewAccessTokenVerifier(secret, auth.SystemClock{}),
		OrganizationService:  organizationService,
		TaskService:          taskService,
		SubmissionService:    submissionService,
		LedgerService:        ledgerService,
		AgentService:         agentService,
		OrgCredentialService: orgCredentialService,
		AssetService:         assetService,
		NotificationService:  notificationService,
		AuditService:         audit.NewService(stores.Audit),
		EventRecorder:        recorder,
	}}
}
