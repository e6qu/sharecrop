package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/mcp"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/web"
)

type healthResponse struct {
	Status string `json:"status"`
}

type AuthService interface {
	Register(context.Context, auth.EmailAddress, auth.PasswordSecret, auth.DisplayNameChoice) auth.RegisterResult
	Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult
	LoginExternal(context.Context, string, string, auth.EmailAddress) auth.ExternalLoginResult
	Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult
	ValidateSession(context.Context, auth.RefreshTokenPlain) auth.ValidateRefreshTokenResult
	Logout(context.Context, auth.RefreshTokenPlain) auth.LogoutResult
	CreateGuest(context.Context) auth.GuestResult
	ListUsers(context.Context, string, core.Page) auth.UserDirectoryResult
	RequestEmailVerification(context.Context, core.UserID) auth.AccountTokenIssueResult
	VerifyEmail(context.Context, auth.AccountTokenPlain) auth.AccountActionResult
	RequestPasswordReset(context.Context, auth.EmailAddress) auth.AccountTokenIssueResult
	ResetPassword(context.Context, auth.AccountTokenPlain, auth.PasswordSecret) auth.AccountActionResult
	ChangePassword(context.Context, core.UserID, auth.PasswordSecret, auth.PasswordSecret) auth.AccountActionResult
	UpdateProfile(context.Context, core.UserID, auth.EmailAddress) auth.AccountActionResult
	UpdateDisplayName(context.Context, core.UserID, auth.DisplayName) auth.AccountActionResult
	DeactivateAccount(context.Context, core.UserID) auth.AccountActionResult
}

type SubjectVerifier interface {
	Verify(auth.AccessToken) auth.SubjectVerifyResult
}

type OrganizationService interface {
	CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult
	ListOrganizations(context.Context, auth.UserSubject, string, core.Page) org.ListOrganizationsResult
	ListMembers(context.Context, auth.UserSubject, core.OrganizationID, core.Page) org.ListMembersResult
	ProvisionMember(context.Context, auth.Subject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult
	DeactivateMember(context.Context, auth.Subject, core.OrganizationID, core.UserID) org.DeactivateMemberResult
	UpdateMemberRoles(context.Context, auth.Subject, core.OrganizationID, core.UserID, []org.Role) org.UpdateMemberRolesResult
	CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult
	CreateStandaloneTeam(context.Context, auth.UserSubject, org.TeamName) org.CreateTeamResult
	ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID, string, core.Page) org.ListTeamsResult
	ListStandaloneTeams(context.Context, auth.UserSubject, string, core.Page) org.ListTeamsResult
	GetTeam(context.Context, auth.Subject, core.TeamID) org.GetTeamResult
	AddTeamMember(context.Context, auth.Subject, core.TeamID, auth.EmailAddress) org.AddTeamMemberResult
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
}

type TaskService interface {
	Create(context.Context, task.CreateCommand) task.CreateResult
	Get(context.Context, auth.Subject, core.TaskID) task.GetResult
	Open(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	Cancel(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	Unpublish(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	List(context.Context, auth.Subject, task.ListScope, task.ListFilters, core.Page) task.ListResult
	ListSeries(context.Context, auth.UserSubject, core.Page) task.ListSeriesResult
	GetSeries(context.Context, auth.Subject, core.TaskSeriesID) task.GetSeriesResult
	CreateSeries(context.Context, auth.UserSubject, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult
	UpdateSeries(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult
	ChangeSeriesState(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesStateTransition) task.SeriesMutationResult
	AddTaskToSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	RemoveTaskFromSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	ReorderSeries(context.Context, auth.UserSubject, core.TaskSeriesID, []core.TaskID) task.SeriesMutationResult
	AddSeriesComment(context.Context, auth.UserSubject, core.TaskSeriesID, task.CommentBody) task.SeriesCommentResult
	ListSeriesComments(context.Context, auth.UserSubject, core.TaskSeriesID, core.Page) task.SeriesCommentsResult
	AddTaskComment(context.Context, auth.UserSubject, core.TaskID, task.CommentBody) task.TaskCommentResult
	ListTaskComments(context.Context, auth.UserSubject, core.TaskID, core.Page) task.TaskCommentsResult
	Reserve(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ReserveForOrganizationTeam(context.Context, auth.UserSubject, core.TaskID, core.OrganizationID, core.TeamID) task.ReservationResult
	ReserveForTeam(context.Context, auth.UserSubject, core.TaskID, core.TeamID) task.ReservationResult
	CancelReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	ListReservations(context.Context, auth.Subject, core.TaskID, core.Page) task.ReservationsListResult
}

type AgentService interface {
	Create(context.Context, core.UserID, agent.Label, agent.ScopeSet, *time.Time, *core.TaskID) agent.CreateResult
	Verify(context.Context, agent.SecretPlain) agent.VerifyResult
	List(context.Context, core.UserID, core.Page) agent.ListResult
	Revoke(context.Context, core.UserID, core.AgentCredentialID) agent.RevokeResult
}

// OrgCredentialService mints and verifies org-wide credentials (see
// internal/orgcred), which act with full parity to an org-admin member
// wherever a Server handler accepts auth.Subject.
type OrgCredentialService interface {
	Create(context.Context, core.OrganizationID, agent.Label, agent.ScopeSet, *time.Time) orgcred.CreateResult
	Verify(context.Context, orgcred.SecretPlain) orgcred.VerifyResult
	List(context.Context, core.OrganizationID, core.Page) orgcred.ListResult
	Revoke(context.Context, core.OrganizationID, core.OrgCredentialID) orgcred.RevokeResult
}

type AssetService interface {
	Mint(context.Context, core.UserID, string, string, string, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy, string) assets.MintResult
	ListCatalog(context.Context) assets.CatalogListResult
	AddCatalogEntry(context.Context, assets.CatalogEntry) assets.CatalogMutationResult
	WithdrawCatalogEntry(context.Context, assets.CatalogSlug) assets.CatalogMutationResult
	DeleteCatalogEntry(context.Context, assets.CatalogSlug) assets.CatalogMutationResult
	AwardFromCatalog(context.Context, core.UserID, string, string, string, string) assets.MintResult
	WithdrawCollectible(context.Context, core.UserID, core.CollectibleID) assets.WithdrawResult
	DeleteWithdrawnCollectible(context.Context, core.CollectibleID) assets.DeleteCollectibleResult
	ListCollectibles(context.Context, core.UserID, core.Page) assets.ListResult
	ListByOwner(context.Context, string, string, core.Page) assets.ListResult
	FundReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult
	RefundReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult
	GiftCollectible(context.Context, core.UserID, core.UserID, core.CollectibleID) assets.GiftResult
	AwardOrganizationCollectible(context.Context, core.OrganizationID, core.CollectibleID, core.UserID) assets.GiftResult
	TransferCollectibleToOrganization(context.Context, core.UserID, core.OrganizationID, core.CollectibleID) assets.GiftResult
	TransferCollectibleFromOrganization(context.Context, core.UserID, core.OrganizationID, core.UserID, core.CollectibleID) assets.GiftResult
	TaskHeldCollectibles(context.Context, core.TaskID) assets.TaskHeldCollectiblesResult
}

type SubmissionService interface {
	Submit(context.Context, submission.SubmitCommand) submission.SubmitResult
	Get(context.Context, auth.Subject, core.SubmissionID) submission.GetResult
	FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	ListForTask(context.Context, auth.Subject, core.TaskID, core.Page) submission.ListResult
	ListForSubmitter(context.Context, auth.UserSubject, core.UserID, core.Page) submission.ListResult
	AddSubmissionComment(context.Context, auth.UserSubject, core.SubmissionID, task.CommentBody) submission.SubmissionCommentResult
	ListSubmissionComments(context.Context, auth.Subject, core.SubmissionID, core.Page) submission.SubmissionCommentsResult
}

type LedgerService interface {
	FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	FundTaskFromOrganization(context.Context, core.UserID, core.OrganizationID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	AcceptSubmission(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey) ledger.AcceptResult
	ReviewAcceptSubmission(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, ledger.CreditReviewSelection, ledger.TipSelection, ledger.CollectibleTipSelection) ledger.AcceptResult
	RequestChanges(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote) ledger.RequestChangesResult
	RejectSubmission(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote, ledger.CreditReviewSelection, ledger.TipSelection, ledger.BanSelection) ledger.RejectResult
	RefundTask(context.Context, core.UserID, core.TaskID, ledger.IdempotencyKey) ledger.RefundResult
	TaskAllocatedCredits(context.Context, core.TaskID) ledger.TaskAllocatedResult
	Balance(context.Context, core.UserID) ledger.BalanceResult
	OrganizationBalance(context.Context, core.OrganizationID) ledger.BalanceResult
	ListEntries(context.Context, core.UserID, core.Page) ledger.ListEntriesResult
	ListOrganizationEntries(context.Context, core.OrganizationID, core.Page) ledger.ListEntriesResult
	GrantCredits(context.Context, core.UserID, ledger.GrantTarget, ledger.CreditAmount, ledger.GrantNote, ledger.IdempotencyKey) ledger.GrantResult
	SendCredits(context.Context, core.UserID, ledger.TransferSource, ledger.TransferTarget, ledger.CreditAmount, ledger.TransferNote, ledger.IdempotencyKey) ledger.SendResult
}

type AuditService interface {
	Record(context.Context, core.UserID, audit.Action, audit.Subject, audit.Metadata) audit.RecordResult
	Get(context.Context, core.AuditEventID) audit.GetResult
	List(context.Context, audit.ListFilters, core.Page) audit.ListResult
}

type NotificationService interface {
	List(context.Context, core.UserID, notification.StateFilter, core.Page) notification.ListResult
	CountUnread(context.Context, core.UserID) notification.CountResult
	MarkRead(context.Context, core.UserID, core.NotificationID) notification.MarkReadResult
}

type PrivacyService interface {
	Create(context.Context, core.UserID, string) PrivacyMutationResult
	ListForRequester(context.Context, core.UserID, core.Page) PrivacyListResult
	ListAll(context.Context, core.Page) PrivacyListResult
	Resolve(context.Context, string, string) PrivacyMutationResult
	RecordSensitiveFieldAccess(context.Context, core.UserID, submission.Submission) PrivacyMutationResult
	RecordSensitiveFieldAccessBatch(context.Context, core.UserID, []submission.Submission) PrivacyMutationResult
	RunRetention(context.Context, core.UserID) PrivacyRetentionResult
}

type ModerationTriageService interface {
	RecordOpen(context.Context, audit.Event) ModerationTriageMutationResult
	List(context.Context, ModerationTriageStateFilter, core.Page) ModerationTriageListResult
	Update(context.Context, core.UserID, core.AuditEventID, ModerationTriageState, string) ModerationTriageMutationResult
}

type Server struct {
	staticFiles           fs.FS
	authService           AuthService
	subjectVerifier       SubjectVerifier
	organizationService   OrganizationService
	taskService           TaskService
	submissionService     SubmissionService
	ledgerService         LedgerService
	agentService          AgentService
	orgCredentialService  OrgCredentialService
	assetService          AssetService
	mcpServer             mcp.Server
	mcpSessions           *mcpHTTPSessionStore
	secureCookies         bool
	ipRateLimiter         RateLimiter
	subjectRateLimiter    RateLimiter
	registrationLimiter   RateLimiter
	platformAdmins        PlatformAdminService
	accountTokens         accountTokenDelivery
	auditService          AuditService
	notificationService   NotificationService
	eventStore            event.Store
	webhookService        webhook.Service
	savedQueueViews       SavedQueueViewService
	privacyService        PrivacyService
	moderationTriage      ModerationTriageService
	shauth                shauthConfig
	requireBrowserSession bool
	oidcSessions          auth.OpenIDConnectSessionStore
}

type RuntimeState struct {
	IPRateLimiter      RateLimiter
	SubjectRateLimiter RateLimiter
	// RegistrationRateLimiter is the dedicated, tighter per-IP budget applied
	// to POST /api/auth/register on top of the generic unauthenticated limiter.
	RegistrationRateLimiter RateLimiter
	MCPSessions             *mcpHTTPSessionStore
	AuditService            AuditService
	NotificationService     NotificationService
	EventStore              event.Store
	WebhookStore            webhook.Store
	SavedQueueViews         SavedQueueViewService
	PrivacyService          PrivacyService
	PlatformAdmins          PlatformAdminService
	ModerationTriage        ModerationTriageService
	OIDCSessions            auth.OpenIDConnectSessionStore
}

// Rate-limit budgets (burst capacity + steady refill per second): bound abusive
// volume on unauthenticated endpoints (by client IP) and MCP tool calls (by agent
// subject) without impeding normal use. Registration gets its own much tighter
// per-IP budget (a burst of 5, one new attempt every 12 minutes) because account
// creation is the one unauthenticated endpoint where volume directly mints
// resources (accounts and signup credit grants).
const (
	IPRateCapacity               = 20
	IPRateRefillPerSec           = 5
	MCPRateCapacity              = 60
	MCPRateRefillPerSec          = 10
	RegistrationRateCapacity     = 5
	RegistrationRateRefillPerSec = 1.0 / 720
)

// DefaultRuntimeState builds the same in-memory RuntimeState New() uses,
// exported so a caller that needs the real domain services + real mux but
// wants to override just one or two RuntimeState fields (e.g. a persistent
// NotificationService) doesn't have to reimplement the other in-memory
// defaults - cmd/sharecrop-wasm does exactly this for the browser demo.
func DefaultRuntimeState(bootstrapAdmins map[string]bool) RuntimeState {
	return RuntimeState{
		IPRateLimiter:           newRateLimiter(IPRateCapacity, IPRateRefillPerSec),
		SubjectRateLimiter:      newRateLimiter(MCPRateCapacity, MCPRateRefillPerSec),
		RegistrationRateLimiter: newRateLimiter(RegistrationRateCapacity, RegistrationRateRefillPerSec),
		MCPSessions:             newMCPHTTPSessionStore(),
		AuditService:            newMemoryAuditService(),
		NotificationService:     notification.NewService(notification.NewMemoryStore()),
		EventStore:              event.NewMemoryStore(),
		WebhookStore:            webhook.NewMemoryStore(),
		SavedQueueViews:         newMemorySavedQueueViewService(),
		PrivacyService:          newMemoryPrivacyService(),
		PlatformAdmins:          newMemoryPlatformAdminService(bootstrapAdmins),
		ModerationTriage:        newMemoryModerationTriageService(),
		OIDCSessions:            newMemoryOpenIDConnectSessionStore(),
	}
}

func New(staticFiles fs.FS, authService AuthService, subjectVerifier SubjectVerifier, organizationService OrganizationService, taskService TaskService, submissionService SubmissionService, ledgerService LedgerService, agentService AgentService, orgCredentialService OrgCredentialService, assetService AssetService) http.Handler {
	bootstrapAdmins := parseAdminUserIDs(os.Getenv("SHARECROP_ADMIN_USER_IDS"))
	return newServer(staticFiles, authService, subjectVerifier, organizationService, taskService, submissionService, ledgerService, agentService, orgCredentialService, assetService, DefaultRuntimeState(bootstrapAdmins))
}

func NewWithRuntimeState(staticFiles fs.FS, authService AuthService, subjectVerifier SubjectVerifier, organizationService OrganizationService, taskService TaskService, submissionService SubmissionService, ledgerService LedgerService, agentService AgentService, orgCredentialService OrgCredentialService, assetService AssetService, runtime RuntimeState) http.Handler {
	if runtime.IPRateLimiter == nil || runtime.SubjectRateLimiter == nil || runtime.RegistrationRateLimiter == nil || runtime.MCPSessions == nil || runtime.AuditService == nil || runtime.NotificationService == nil || runtime.EventStore == nil || runtime.WebhookStore == nil || runtime.SavedQueueViews == nil || runtime.PrivacyService == nil || runtime.PlatformAdmins == nil || runtime.ModerationTriage == nil || runtime.OIDCSessions == nil {
		panic("runtime state requires explicit rate limiters (including the registration limiter), MCP sessions, audit service, notification service, event store, webhook store, saved queue views, privacy service, platform admin service, moderation triage service, and OpenID Connect session storage")
	}
	return newServer(staticFiles, authService, subjectVerifier, organizationService, taskService, submissionService, ledgerService, agentService, orgCredentialService, assetService, runtime)
}

func newServer(staticFiles fs.FS, authService AuthService, subjectVerifier SubjectVerifier, organizationService OrganizationService, taskService TaskService, submissionService SubmissionService, ledgerService LedgerService, agentService AgentService, orgCredentialService OrgCredentialService, assetService AssetService, runtime RuntimeState) http.Handler {
	shauth := shauthConfigFromEnv()
	if err := shauth.validate(); err != nil {
		panic(err)
	}
	server := Server{
		staticFiles:          staticFiles,
		authService:          authService,
		subjectVerifier:      subjectVerifier,
		organizationService:  organizationService,
		taskService:          taskService,
		submissionService:    submissionService,
		ledgerService:        ledgerService,
		agentService:         agentService,
		orgCredentialService: orgCredentialService,
		assetService:         assetService,
		mcpServer:            mcp.NewServer(mcpServices{taskService: taskService, submissionService: submissionService, ledgerService: ledgerService, organizationService: organizationService, orgCredentialService: orgCredentialService, assetService: assetService, notificationService: runtime.NotificationService, authService: authService, platformAdmins: runtime.PlatformAdmins, moderationTriage: runtime.ModerationTriage, privacyService: runtime.PrivacyService, auditService: runtime.AuditService, webhookService: webhook.NewService(runtime.WebhookStore), eventStore: runtime.EventStore}),
		mcpSessions:          runtime.MCPSessions,
		// The refresh-token cookie is Secure by default; local plain-HTTP dev can
		// opt out explicitly with SHARECROP_INSECURE_COOKIES=true.
		secureCookies:         os.Getenv("SHARECROP_INSECURE_COOKIES") != "true",
		ipRateLimiter:         runtime.IPRateLimiter,
		subjectRateLimiter:    runtime.SubjectRateLimiter,
		registrationLimiter:   runtime.RegistrationRateLimiter,
		accountTokens:         newAccountTokenDeliveryFromEnv(),
		auditService:          runtime.AuditService,
		notificationService:   runtime.NotificationService,
		eventStore:            runtime.EventStore,
		webhookService:        webhook.NewService(runtime.WebhookStore),
		savedQueueViews:       runtime.SavedQueueViews,
		privacyService:        runtime.PrivacyService,
		platformAdmins:        runtime.PlatformAdmins,
		moderationTriage:      runtime.ModerationTriage,
		oidcSessions:          runtime.OIDCSessions,
		shauth:                shauth,
		requireBrowserSession: shauth.enabled() || os.Getenv("SHARECROP_REQUIRE_BROWSER_SESSION") == "true",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /api/auth/register", server.register)
	mux.HandleFunc("POST /api/auth/login", server.login)
	mux.HandleFunc("GET /api/auth/shauth", server.shauthLogin)
	mux.HandleFunc("GET /api/auth/shauth/callback", server.shauthCallback)
	mux.HandleFunc("GET /api/auth/shauth/frontchannel-logout", server.shauthFrontchannelLogout)
	mux.HandleFunc("POST /api/auth/shauth/backchannel-logout", server.shauthBackchannelLogout)
	mux.HandleFunc("GET /auth/shauth/logout/complete", server.shauthLogoutComplete)
	mux.HandleFunc("GET /auth/validation", server.shauthValidation)
	mux.HandleFunc("POST /auth/shauth/logout", server.shauthValidationLogout)
	mux.HandleFunc("GET /api/auth/signed-out", server.shauthSignedOut)
	mux.HandleFunc("POST /api/auth/refresh", server.refresh)
	mux.HandleFunc("POST /api/auth/logout", server.logout)
	mux.HandleFunc("POST /api/auth/guest", server.guest)
	mux.HandleFunc("POST /api/auth/email-verification/confirm", server.confirmEmailVerification)
	mux.HandleFunc("POST /api/auth/password-reset/request", server.requestPasswordReset)
	mux.HandleFunc("POST /api/auth/password-reset/confirm", server.confirmPasswordReset)
	mux.HandleFunc("POST /api/account/email-verification", server.requestEmailVerification)
	mux.HandleFunc("PATCH /api/account/password", server.changePassword)
	mux.HandleFunc("GET /api/account/profile", server.accountProfile)
	mux.HandleFunc("PATCH /api/account/profile", server.updateAccountProfile)
	mux.HandleFunc("PATCH /api/account/display-name", server.updateAccountDisplayName)
	mux.HandleFunc("DELETE /api/account", server.deactivateAccount)
	mux.HandleFunc("POST /api/privacy-requests", server.createPrivacyRequest)
	mux.HandleFunc("GET /api/privacy-requests", server.listPrivacyRequests)
	mux.HandleFunc("POST /api/moderation/reports", server.createModerationReport)
	mux.HandleFunc("GET /api/saved-queue-views", server.listSavedQueueViews)
	mux.HandleFunc("POST /api/saved-queue-views", server.upsertSavedQueueView)
	mux.HandleFunc("GET /api/organizations", server.listOrganizations)
	mux.HandleFunc("POST /api/organizations", server.createOrganization)
	mux.HandleFunc("GET /api/organizations/{organization_id}/members", server.listOrganizationMembers)
	mux.HandleFunc("POST /api/organizations/{organization_id}/members", server.provisionOrganizationMember)
	mux.HandleFunc("PATCH /api/organizations/{organization_id}/members/{user_id}/roles", server.updateOrganizationMemberRoles)
	mux.HandleFunc("PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate", server.deactivateOrganizationMember)
	mux.HandleFunc("GET /api/organizations/{organization_id}/teams", server.listOrganizationTeams)
	mux.HandleFunc("POST /api/organizations/{organization_id}/teams", server.createOrganizationTeam)
	mux.HandleFunc("GET /api/organizations/{organization_id}/credits/ledger", server.organizationCreditsLedger)
	mux.HandleFunc("GET /api/organizations/{organization_id}/audit-events", server.listOrganizationAuditEvents)
	mux.HandleFunc("GET /api/teams", server.listStandaloneTeams)
	mux.HandleFunc("POST /api/teams", server.createStandaloneTeam)
	mux.HandleFunc("GET /api/teams/{team_id}", server.getTeam)
	mux.HandleFunc("GET /api/teams/{team_id}/work", server.getTeamWork)
	mux.HandleFunc("POST /api/teams/{team_id}/members", server.addTeamMember)
	mux.HandleFunc("GET /api/users", server.listUsers)
	mux.HandleFunc("GET /api/users/{user_id}", server.getUserProfile)
	mux.HandleFunc("GET /api/users/{user_id}/work", server.getUserWork)
	mux.HandleFunc("GET /api/users/{user_id}/submissions", server.getUserSubmissions)
	mux.HandleFunc("GET /api/tasks", server.listTasks)
	mux.HandleFunc("POST /api/tasks", server.createTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/open", server.openTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/cancel", server.cancelTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions", server.createAuthenticatedSubmission)
	mux.HandleFunc("GET /api/tasks/{task_id}/submissions", server.listTaskSubmissions)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations", server.reserveTask)
	mux.HandleFunc("GET /api/tasks/{task_id}/reservations", server.listTaskReservations)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel", server.cancelTaskReservation)
	mux.HandleFunc("GET /api/submission-receipts/{receipt_token}", server.findSubmissionReceipt)
	mux.HandleFunc("GET /api/submissions/{submission_id}/comments", server.listSubmissionComments)
	mux.HandleFunc("POST /api/submissions/{submission_id}/comments", server.addSubmissionComment)
	mux.HandleFunc("GET /api/organizations/{organization_id}/credits/balance", server.organizationCreditsBalance)
	mux.HandleFunc("GET /api/credits/balance", server.creditsBalance)
	mux.HandleFunc("GET /api/credits/ledger", server.creditsLedger)
	mux.HandleFunc("POST /api/credits/transfers", server.sendCredits)
	mux.HandleFunc("POST /api/tasks/{task_id}/funding", server.fundTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/refund", server.refundTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/accept", server.acceptSubmission)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/request-changes", server.requestSubmissionChanges)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/reject", server.rejectSubmission)
	mux.HandleFunc("GET /api/tasks/{task_id}", server.getTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/unpublish", server.unpublishTask)
	mux.HandleFunc("GET /api/tasks/{task_id}/comments", server.listTaskComments)
	mux.HandleFunc("POST /api/tasks/{task_id}/comments", server.addTaskComment)
	mux.HandleFunc("GET /api/task-series", server.listTaskSeries)
	mux.HandleFunc("POST /api/task-series", server.createTaskSeries)
	mux.HandleFunc("GET /api/task-series/{series_id}", server.getTaskSeries)
	mux.HandleFunc("PATCH /api/task-series/{series_id}", server.updateTaskSeries)
	mux.HandleFunc("POST /api/task-series/{series_id}/publish", server.publishTaskSeries)
	mux.HandleFunc("POST /api/task-series/{series_id}/unpublish", server.unpublishTaskSeries)
	mux.HandleFunc("POST /api/task-series/{series_id}/close", server.closeTaskSeries)
	mux.HandleFunc("POST /api/task-series/{series_id}/reopen", server.reopenTaskSeries)
	mux.HandleFunc("POST /api/task-series/{series_id}/tasks", server.addTaskToSeriesHandler)
	mux.HandleFunc("DELETE /api/task-series/{series_id}/tasks/{task_id}", server.removeTaskFromSeriesHandler)
	mux.HandleFunc("POST /api/task-series/{series_id}/reorder", server.reorderTaskSeries)
	mux.HandleFunc("GET /api/task-series/{series_id}/comments", server.listTaskSeriesComments)
	mux.HandleFunc("POST /api/task-series/{series_id}/comments", server.addTaskSeriesComment)
	mux.HandleFunc("POST /api/collectibles", server.mintCollectible)
	mux.HandleFunc("GET /api/collectibles", server.listCollectibles)
	mux.HandleFunc("GET /api/collectibles/catalog", server.collectibleCatalog)
	mux.HandleFunc("POST /api/collectibles/award", server.awardCollectible)
	mux.HandleFunc("POST /api/collectibles/{collectible_id}/transfer", server.transferCollectible)
	mux.HandleFunc("POST /api/admin/collectible-catalog", server.addCatalogEntry)
	mux.HandleFunc("POST /api/admin/collectible-catalog/{slug}/withdraw", server.withdrawCatalogEntry)
	mux.HandleFunc("DELETE /api/admin/collectible-catalog/{slug}", server.deleteCatalogEntry)
	mux.HandleFunc("POST /api/admin/collectibles/{collectible_id}/withdraw", server.withdrawCollectible)
	mux.HandleFunc("DELETE /api/admin/collectibles/{collectible_id}", server.deleteCollectible)
	mux.HandleFunc("GET /api/admin/operations", server.operationsStatus)
	mux.HandleFunc("GET /api/admin/platform-admins", server.listPlatformAdmins)
	mux.HandleFunc("POST /api/admin/platform-admins", server.grantPlatformAdmin)
	mux.HandleFunc("POST /api/admin/platform-admins/{user_id}/revoke", server.revokePlatformAdmin)
	mux.HandleFunc("GET /api/admin/audit-events", server.listAuditEvents)
	mux.HandleFunc("POST /api/admin/credits/grants", server.grantCredits)
	mux.HandleFunc("GET /api/admin/moderation/reports", server.listAdminModerationReports)
	mux.HandleFunc("POST /api/admin/moderation/reports/{report_id}/triage", server.triageModerationReport)
	mux.HandleFunc("GET /api/admin/privacy-requests", server.listAdminPrivacyRequests)
	mux.HandleFunc("POST /api/admin/privacy-requests/{privacy_request_id}/resolve", server.resolveAdminPrivacyRequest)
	mux.HandleFunc("POST /api/admin/privacy-retention/run", server.runPrivacyRetention)
	mux.HandleFunc("GET /api/events", server.listEvents)
	mux.HandleFunc("GET /api/events/stream", server.streamEvents)
	mux.HandleFunc("POST /api/webhook-subscriptions", server.createWebhookSubscription)
	mux.HandleFunc("GET /api/webhook-subscriptions", server.listWebhookSubscriptions)
	mux.HandleFunc("DELETE /api/webhook-subscriptions/{subscription_id}", server.revokeWebhookSubscription)
	mux.HandleFunc("GET /api/webhook-subscriptions/{subscription_id}/deliveries", server.listWebhookDeliveries)
	mux.HandleFunc("GET /api/notifications", server.listNotifications)
	mux.HandleFunc("GET /api/notifications/unread-count", server.unreadNotificationCount)
	mux.HandleFunc("POST /api/notifications/{notification_id}/read", server.markNotificationRead)
	mux.HandleFunc("GET /api/organizations/{organization_id}/collectibles", server.listOrganizationCollectibles)
	mux.HandleFunc("POST /api/organizations/{organization_id}/collectibles/{collectible_id}/award", server.awardOrganizationCollectible)
	mux.HandleFunc("POST /api/organizations/{organization_id}/collectibles/{collectible_id}/transfer", server.transferOrganizationCollectible)
	mux.HandleFunc("GET /api/teams/{team_id}/collectibles", server.listTeamCollectibles)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-reward", server.fundCollectibleReward)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-refund", server.refundCollectibleReward)
	mux.HandleFunc("POST /api/agent-credentials", server.createAgentCredential)
	mux.HandleFunc("GET /api/agent-credentials", server.listAgentCredentials)
	mux.HandleFunc("POST /api/agent-credentials/{credential_id}/revoke", server.revokeAgentCredential)
	mux.HandleFunc("POST /api/organizations/{organization_id}/credentials", server.createOrgCredential)
	mux.HandleFunc("GET /api/organizations/{organization_id}/credentials", server.listOrgCredentials)
	mux.HandleFunc("POST /api/organizations/{organization_id}/credentials/{credential_id}/revoke", server.revokeOrgCredential)
	mux.HandleFunc("POST /mcp", server.mcpEndpoint)
	mux.HandleFunc("GET /mcp", server.mcpStream)
	mux.HandleFunc("DELETE /mcp", server.mcpDeleteSession)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /", server.index)
	return withRequestBodyLimit(mux)
}

// maxRequestBodyBytes bounds the size of each request body decoded by the API
// so a large upload cannot exhaust memory. The MCP endpoint applies its own
// stricter limit, which takes effect before this one.
const maxRequestBodyBytes = 2 << 20

func withRequestBodyLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}

// NewMCPServer builds an MCP server backed by the given domain services so the
// stdio transport can reuse the same tool surface as the HTTP endpoint.
func NewMCPServer(taskService TaskService, submissionService SubmissionService, ledgerService LedgerService, organizationService OrganizationService, orgCredentialService OrgCredentialService, assetService AssetService, notificationService NotificationService, authService AuthService, platformAdmins PlatformAdminService, moderationTriage ModerationTriageService, privacyService PrivacyService, auditService AuditService, webhookService webhook.Service, eventStore event.Store) mcp.Server {
	return mcp.NewServer(mcpServices{taskService: taskService, submissionService: submissionService, ledgerService: ledgerService, organizationService: organizationService, orgCredentialService: orgCredentialService, assetService: assetService, notificationService: notificationService, authService: authService, platformAdmins: platformAdmins, moderationTriage: moderationTriage, privacyService: privacyService, auditService: auditService, webhookService: webhookService, eventStore: eventStore})
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

func (server Server) index(w http.ResponseWriter, r *http.Request) {
	// The browser app is a single-page application served from the same
	// shell for every in-app route, so deep links and refreshes load the
	// app. Unmatched API paths still return 404 rather than the shell.
	if strings.HasPrefix(r.URL.Path, "/api/") {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "no such API route")
		return
	}
	if server.requireBrowserSession && !server.requireActiveBrowserSession(w, r) {
		return
	}
	data, err := web.ApplicationShell(server.staticFiles, server.shauth.enabled())
	if err != nil {
		http.Error(w, "index not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (balanceResponse) writableResponse() {}

func (ledgerListResponse) writableResponse() {}

func (taskFundResponse) writableResponse() {}

func (acceptSubmissionResponse) writableResponse() {}

func (reviewSubmissionResponse) writableResponse() {}

func (reservationResponse) writableResponse() {}

func (reservationsResponse) writableResponse() {}

type authRequestResult interface {
	authRequestResult()
}

type authRequestAccepted struct {
	email    auth.EmailAddress
	password auth.PasswordSecret
}

type authRequestRejected struct {
	reason string
}

func (authRequestAccepted) authRequestResult() {}

func (authRequestRejected) authRequestResult() {}

type taskReservationChanger func(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult

type taskStateChanger func(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult

type reviewPathResult interface {
	reviewPathResult()
}

type reviewPathAccepted struct {
	actor        auth.Subject
	reviewer     ledger.Reviewer
	taskID       core.TaskID
	submissionID core.SubmissionID
}

type reviewPathRejected struct{}

func (reviewPathAccepted) reviewPathResult() {}

func (reviewPathRejected) reviewPathResult() {}

type creditSelectionResult interface {
	creditSelectionResult()
}

type creditSelectionAccepted struct {
	value ledger.CreditReviewSelection
}

type creditSelectionRejected struct {
	reason string
}

func (creditSelectionAccepted) creditSelectionResult() {}

func (creditSelectionRejected) creditSelectionResult() {}

type tipSelectionResult interface {
	tipSelectionResult()
}

type tipSelectionAccepted struct {
	value ledger.TipSelection
}

type tipSelectionRejected struct {
	reason string
}

func (tipSelectionAccepted) tipSelectionResult() {}

func (tipSelectionRejected) tipSelectionResult() {}

func writeJSON(w http.ResponseWriter, status int, value writableResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeAuthRequest(r *http.Request) authRequestResult {
	var request authRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return authRequestRejected{reason: "request body is invalid"}
	}
	return parseAuthCredentials(request.Email, request.Password)
}

func parseAuthCredentials(rawEmail string, rawPassword string) authRequestResult {
	emailResult := auth.NewEmailAddress(rawEmail)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return authRequestRejected{reason: rejected.Reason.Description()}
	}

	passwordResult := auth.NewPasswordSecret(rawPassword)
	passwordAccepted, passwordMatched := passwordResult.(auth.PasswordSecretAccepted)
	if !passwordMatched {
		rejected := passwordResult.(auth.PasswordSecretRejected)
		return authRequestRejected{reason: rejected.Reason.Description()}
	}

	return authRequestAccepted{email: emailAccepted.Value, password: passwordAccepted.Value}
}

type userSubjectResult interface {
	userSubjectResult()
}

type userSubjectAccepted struct {
	subject auth.UserSubject
}

type userSubjectRejected struct {
	reason string
}

func (userSubjectAccepted) userSubjectResult() {}

func (userSubjectRejected) userSubjectResult() {}

// parseAdminUserIDs builds the set of platform-admin user ids from a
// comma-separated env value, ignoring blank entries.
func parseAdminUserIDs(raw string) map[string]bool {
	admins := map[string]bool{}
	for _, part := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			admins[trimmed] = true
		}
	}
	return admins
}

func ParseAdminUserIDsForRuntime(raw string) map[string]bool {
	return parseAdminUserIDs(raw)
}

// ParseRegistrationRateCapacityForRuntime maps the optional
// SHARECROP_REGISTRATION_RATE_CAPACITY value onto the registration
// limiter's burst capacity. Blank keeps the production default. A value
// that does not parse to a positive integer also keeps the default: the
// default is the tighter (fail-closed) setting, so a typo can only make the
// limiter stricter than intended, never looser. The knob exists for test
// harnesses that mint many accounts from one client address (the browser
// E2E suite registers a fresh account per test from 127.0.0.1).
func ParseRegistrationRateCapacityForRuntime(raw string) int {
	capacity, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || capacity < 1 {
		return RegistrationRateCapacity
	}
	return capacity
}

// ParseRegistrationRateRefillForRuntime maps the optional
// SHARECROP_REGISTRATION_RATE_REFILL value (tokens per second) onto the
// registration limiter's refill rate, with the same fail-closed rule as the
// capacity knob: blank or unparsable keeps the production default. The knob
// exists because registration buckets persist in the rate-limit store, so a
// test harness that raises only the capacity still inherits a drained bucket
// from an earlier suite run against the same database; raising the refill
// rate makes the bucket recover immediately.
func ParseRegistrationRateRefillForRuntime(raw string) float64 {
	refill, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || refill <= 0 {
		return RegistrationRateRefillPerSec
	}
	return refill
}

// requireWorkerSubject resolves a request to an acting user subject from either
// a user access token or an agent credential that holds the required scope. This
// lets a single agent token drive the worker REST endpoints as well as MCP (an
// agent credential always acts as its owning user, exactly as it does over MCP).
// taskID is the task this specific request acts on: a task-scoped credential
// (Credential.TaskID != nil, e.g. one auto-issued on reservation) is rejected
// outright if it doesn't match, regardless of what scopes it holds.
func (server Server) requireWorkerSubject(r *http.Request, scope agent.Scope, taskID core.TaskID) userSubjectResult {
	if accepted, matched := server.requireUserSubject(r).(userSubjectAccepted); matched {
		return accepted
	}
	verifyResult := server.verifyAgent(r)
	verified, matched := verifyResult.(agent.CredentialVerified)
	if !matched {
		return userSubjectRejected{reason: "a user access token or an agent credential is required"}
	}
	if _, granted := verified.Credential.Scopes.Allows(scope).(agent.ScopeGranted); !granted {
		return userSubjectRejected{reason: "the agent credential is missing the " + scope.String() + " scope"}
	}
	if !verified.Credential.MatchesTask(taskID) {
		return userSubjectRejected{reason: "the agent credential is not valid for this task"}
	}
	return userSubjectAccepted{subject: verified.Subject}
}

func (server Server) requireUserSubject(r *http.Request) userSubjectResult {
	if server.requireBrowserSession {
		cookie, err := r.Cookie("sharecrop_refresh_token")
		if err != nil || cookie.Value == "" {
			return userSubjectRejected{reason: "active Sharecrop browser session is required"}
		}
		parsed, ok := auth.ParseRefreshTokenPlain(cookie.Value).(auth.RefreshTokenPlainAccepted)
		if !ok {
			return userSubjectRejected{reason: "active Sharecrop browser session is required"}
		}
		if _, active := server.authService.ValidateSession(r.Context(), parsed.Value).(auth.RefreshTokenActive); !active {
			return userSubjectRejected{reason: "active Sharecrop browser session is required"}
		}
	}
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched {
		return userSubjectRejected{reason: "bearer access token is required"}
	}

	tokenResult := auth.ParseAccessToken(rawToken)
	tokenAccepted, tokenMatched := tokenResult.(auth.AccessTokenParsed)
	if !tokenMatched {
		rejected := tokenResult.(auth.AccessTokenParseRejected)
		return userSubjectRejected{reason: rejected.Reason.Description()}
	}

	verifyResult := server.subjectVerifier.Verify(tokenAccepted.Value)
	verified, verifyMatched := verifyResult.(auth.SubjectVerified)
	if !verifyMatched {
		rejected := verifyResult.(auth.SubjectVerifyRejected)
		return userSubjectRejected{reason: rejected.Reason.Description()}
	}

	subject, subjectMatched := verified.Value.(auth.UserSubject)
	if !subjectMatched {
		return userSubjectRejected{reason: "user access token is required"}
	}

	return userSubjectAccepted{subject: subject}
}

type actorResult interface {
	actorResult()
}

type actorAccepted struct {
	actor auth.Subject
}

type actorRejected struct {
	reason string
}

type actorScopeDenied struct {
	reason string
}

func (actorAccepted) actorResult() {}

func (actorRejected) actorResult() {}

func (actorScopeDenied) actorResult() {}

// requireUserOrOrgSubject resolves a request to either a user session
// subject or an org-wide credential's OrgSubject, for handlers on methods
// widened to accept auth.Subject (full org-token parity). Handlers that stay
// user-only keep using requireUserSubject and never see an OrgSubject.
//
// A user session carries no scope model and is never scope-checked; an
// organization credential must have been minted with the required scope —
// the same scope MCP enforces for the equivalent tool — so a read-only org
// credential cannot mutate over REST.
func (server Server) requireUserOrOrgSubject(r *http.Request, required agent.Scope) actorResult {
	if accepted, matched := server.requireUserSubject(r).(userSubjectAccepted); matched {
		return actorAccepted{actor: accepted.subject}
	}
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched || !orgcred.HasSecretPrefix(rawToken) {
		return actorRejected{reason: "a user access token or an organization credential is required"}
	}
	secretResult := orgcred.ParseSecretPlain(rawToken)
	secret, secretMatched := secretResult.(orgcred.SecretPlainAccepted)
	if !secretMatched {
		return actorRejected{reason: secretResult.(orgcred.SecretPlainRejected).Reason.Description()}
	}
	verifyResult := server.orgCredentialService.Verify(r.Context(), secret.Value)
	verified, verifyMatched := verifyResult.(orgcred.CredentialVerified)
	if !verifyMatched {
		return actorRejected{reason: verifyResult.(orgcred.VerifyRejected).Reason.Description()}
	}
	if _, granted := verified.Credential.Scopes.Allows(required).(agent.ScopeGranted); !granted {
		return actorScopeDenied{reason: "organization credential lacks the " + required.String() + " scope"}
	}
	return actorAccepted{actor: verified.Subject}
}

// writeActorRejection maps a failed requireUserOrOrgSubject result to its
// response: missing/invalid credentials are 401, a valid credential without
// the required scope is 403.
func writeActorRejection(w http.ResponseWriter, result actorResult) {
	if denied, matched := result.(actorScopeDenied); matched {
		writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, denied.reason)
		return
	}
	writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, result.(actorRejected).reason)
}

type organizationIDResult interface {
	organizationIDResult()
}

type organizationIDAccepted struct {
	value core.OrganizationID
}

type organizationIDRejected struct {
	reason string
}

func (organizationIDAccepted) organizationIDResult() {}

func (organizationIDRejected) organizationIDResult() {}

func parseOrganizationPathValue(r *http.Request) organizationIDResult {
	result := core.ParseOrganizationID(r.PathValue("organization_id"))
	accepted, matched := result.(core.OrganizationIDCreated)
	if !matched {
		rejected := result.(core.OrganizationIDRejected)
		return organizationIDRejected{reason: rejected.Reason.Description()}
	}
	return organizationIDAccepted{value: accepted.Value}
}

type provisionMemberResult interface {
	provisionMemberResult()
}

type provisionMemberAccepted struct {
	email auth.EmailAddress
	roles []org.Role
}

type provisionMemberRejected struct {
	reason string
}

func (provisionMemberAccepted) provisionMemberResult() {}

func (provisionMemberRejected) provisionMemberResult() {}

func decodeProvisionMemberRequest(r *http.Request) provisionMemberResult {
	var request provisionMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return provisionMemberRejected{reason: "request body is invalid"}
	}

	emailResult := auth.NewEmailAddress(request.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return provisionMemberRejected{reason: rejected.Reason.Description()}
	}

	rolesResult := parseOrganizationRoles(request.Roles)
	roles, rolesMatched := rolesResult.(organizationRolesAccepted)
	if !rolesMatched {
		return provisionMemberRejected{reason: rolesResult.(organizationRolesRejected).reason}
	}

	return provisionMemberAccepted{email: emailAccepted.Value, roles: roles.values}
}

type organizationRolesResult interface {
	organizationRolesResult()
}

type organizationRolesAccepted struct {
	values []org.Role
}

type organizationRolesRejected struct {
	reason string
}

func (organizationRolesAccepted) organizationRolesResult() {}

func (organizationRolesRejected) organizationRolesResult() {}

func parseOrganizationRoles(rawRoles []string) organizationRolesResult {
	roles := make([]org.Role, 0, len(rawRoles))
	for _, rawRole := range rawRoles {
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			rejected := roleResult.(org.RoleRejected)
			return organizationRolesRejected{reason: rejected.Reason.Description()}
		}
		roles = append(roles, roleAccepted.Value)
	}

	if len(roles) == 0 {
		return organizationRolesRejected{reason: "at least one organization role is required"}
	}

	return organizationRolesAccepted{values: roles}
}

type taskRequestResult interface {
	taskRequestResult()
}

type taskRequestAccepted struct {
	command task.CreateCommand
}

type taskRequestRejected struct {
	reason string
}

func (taskRequestAccepted) taskRequestResult() {}

func (taskRequestRejected) taskRequestResult() {}

type taskParticipationResult interface {
	taskParticipationResult()
}

type taskParticipationAccepted struct {
	policy        task.ParticipationPolicy
	assigneeScope task.AssigneeScope
	ttl           task.ReservationTTL
}

type taskParticipationRejected struct {
	reason string
}

func (taskParticipationAccepted) taskParticipationResult() {}

func (taskParticipationRejected) taskParticipationResult() {}

type taskRewardResult interface {
	taskRewardResult()
}

type taskRewardAccepted struct {
	value          task.RewardSpec
	collectibleIDs []core.CollectibleID
}

type taskRewardRejected struct {
	reason string
}

func (taskRewardAccepted) taskRewardResult() {}

func (taskRewardRejected) taskRewardResult() {}

type taskOwnerResult interface {
	taskOwnerResult()
}

type taskOwnerAccepted struct {
	value task.Owner
}

type taskOwnerRejected struct {
	reason string
}

func (taskOwnerAccepted) taskOwnerResult() {}

func (taskOwnerRejected) taskOwnerResult() {}

type taskVisibilityResult interface {
	taskVisibilityResult()
}

type taskVisibilityAccepted struct {
	value task.Visibility
}

type taskVisibilityRejected struct {
	reason string
}

func (taskVisibilityAccepted) taskVisibilityResult() {}

func (taskVisibilityRejected) taskVisibilityResult() {}

type taskPlacementResult interface {
	taskPlacementResult()
}

type taskPlacementAccepted struct {
	value task.SeriesPlacement
}

type taskPlacementRejected struct {
	reason string
}

func (taskPlacementAccepted) taskPlacementResult() {}

func (taskPlacementRejected) taskPlacementResult() {}

type taskPayloadResult interface {
	taskPayloadResult()
}

type taskPayloadAccepted struct {
	value task.DataPayload
}

type taskPayloadRejected struct {
	reason string
}

func (taskPayloadAccepted) taskPayloadResult() {}

func (taskPayloadRejected) taskPayloadResult() {}

type taskIDResult interface {
	taskIDResult()
}

type taskIDAccepted struct {
	value core.TaskID
}

type taskIDRejected struct {
	reason string
}

func (taskIDAccepted) taskIDResult() {}

func (taskIDRejected) taskIDResult() {}

type reservationIDResult interface {
	reservationIDResult()
}

type reservationIDAccepted struct {
	value core.TaskReservationID
}

type reservationIDRejected struct {
	reason string
}

func (reservationIDAccepted) reservationIDResult() {}

func (reservationIDRejected) reservationIDResult() {}

type taskListScopeResult interface {
	taskListScopeResult()
}

type taskListScopeAccepted struct {
	value task.ListScope
}

type taskListScopeRejected struct {
	reason string
}

func (taskListScopeAccepted) taskListScopeResult() {}

func (taskListScopeRejected) taskListScopeResult() {}

type taskListFiltersResult interface {
	taskListFiltersResult()
}

type taskListFiltersAccepted struct {
	value task.ListFilters
}

type taskListFiltersRejected struct {
	reason core.DomainError
}

func (taskListFiltersAccepted) taskListFiltersResult() {}

func (taskListFiltersRejected) taskListFiltersResult() {}

// parsePageOrReject parses the limit/offset query parameters strictly,
// writing a 400 response and reporting false when they are invalid. Every
// list endpoint uses this; there is no lenient variant that silently coerces
// malformed paging input to defaults.
func parsePageOrReject(w http.ResponseWriter, r *http.Request) (core.Page, bool) {
	pageResult := parsePageStrict(r)
	accepted, matched := pageResult.(pageParseAccepted)
	if !matched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, pageResult.(pageParseRejected).reason)
		return core.Page{}, false
	}
	return accepted.value, true
}

// probeListWindow sizes a list page fetched with page.Probe() (limit+1 rows),
// delegating to core.ProbeListWindow so REST and MCP share one definition of
// the next_offset semantics.
func probeListWindow(fetched int, page core.Page) (visible int, nextOffset int) {
	return core.ProbeListWindow(fetched, page)
}

type pageParseResult interface {
	pageParseResult()
}

type pageParseAccepted struct {
	value core.Page
}

type pageParseRejected struct {
	reason string
}

func (pageParseAccepted) pageParseResult() {}

func (pageParseRejected) pageParseResult() {}

func parsePageStrict(r *http.Request) pageParseResult {
	query := r.URL.Query()
	rawLimit := query.Get("limit")
	rawOffset := query.Get("offset")
	if rawLimit == "" && rawOffset == "" {
		return pageParseAccepted{value: core.DefaultPage()}
	}
	limit := core.DefaultPage().Limit()
	if rawLimit != "" {
		parsed, limitErr := strconv.Atoi(rawLimit)
		if limitErr != nil {
			return pageParseRejected{reason: "limit query parameter is invalid"}
		}
		limit = parsed
	}
	offset := core.DefaultPage().Offset()
	if rawOffset != "" {
		parsed, offsetErr := strconv.Atoi(rawOffset)
		if offsetErr != nil {
			return pageParseRejected{reason: "offset query parameter is invalid"}
		}
		offset = parsed
	}
	pageResult := core.NewPage(limit, offset)
	accepted, matched := pageResult.(core.PageAccepted)
	if !matched {
		return pageParseRejected{reason: pageResult.(core.PageRejected).Reason.Description()}
	}
	return pageParseAccepted{value: accepted.Value}
}

type submissionRequestResult interface {
	submissionRequestResult()
}

type submissionRequestAccepted struct {
	command submission.SubmitCommand
}

type submissionRequestRejected struct {
	reason string
}

func (submissionRequestAccepted) submissionRequestResult() {}

func (submissionRequestRejected) submissionRequestResult() {}

type authResponseResult interface {
	authResponseResult()
}

type authResponseAccepted struct {
	response authResponse
}

type authResponseRejected struct {
	reason string
}

func (authResponseAccepted) authResponseResult() {}

func (authResponseRejected) authResponseResult() {}

func authResponseForSubject(subject auth.Subject, accessToken auth.AccessToken) authResponseResult {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return authResponseAccepted{response: authResponse{SubjectKind: "user", SubjectID: typed.ID.String(), AccessToken: accessToken.String()}}
	case auth.GuestSubject:
		return authResponseAccepted{response: authResponse{SubjectKind: "guest", SubjectID: typed.ID.String(), AccessToken: accessToken.String()}}
	default:
		return authResponseRejected{reason: "subject is invalid"}
	}
}

// allowByIP rate-limits an unauthenticated endpoint by client IP. It writes a
// 429 and returns false when the caller should stop.
func (server Server) allowByIP(w http.ResponseWriter, r *http.Request) bool {
	if !server.ipRateLimiter.Allow(r.URL.Path + ":" + clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, core.ErrorCodeRateLimited, "too many requests; slow down and retry")
		return false
	}
	return true
}

// allowBySubject rate-limits an authenticated, DB-heavy endpoint by acting
// subject so a single account cannot spam transactional review operations.
func (server Server) allowBySubject(w http.ResponseWriter, subjectID string) bool {
	if !server.subjectRateLimiter.Allow(subjectID) {
		writeError(w, http.StatusTooManyRequests, core.ErrorCodeRateLimited, "too many requests; slow down and retry")
		return false
	}
	return true
}

func (server Server) setRefreshCookie(w http.ResponseWriter, refreshToken auth.RefreshTokenPlain) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sharecrop_refresh_token",
		Value:    refreshToken.String(),
		Path:     "/",
		HttpOnly: true,
		Secure:   server.secureCookies,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(30 * 24 * time.Hour),
	})
}

func (server Server) clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sharecrop_refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   server.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
	})
}

func organizationToResponse(value org.Organization) organizationResponse {
	return organizationResponse{ID: value.ID.String(), Name: value.Name.String(), CreatedBy: value.CreatedBy.String()}
}

func memberToResponse(value org.OrganizationMember) organizationMemberResponse {
	roles := make([]string, 0, len(value.Roles))
	for _, role := range value.Roles {
		roles = append(roles, role.String())
	}
	return organizationMemberResponse{
		ID:             value.ID.String(),
		OrganizationID: value.OrganizationID.String(),
		UserID:         value.UserID.String(),
		Status:         value.Status.String(),
		Roles:          roles,
	}
}

func teamToResponse(value org.Team) teamResponse {
	organizationID := ""
	ownerUserID := ""
	switch owner := value.Owner.(type) {
	case org.OrganizationOwnedTeam:
		organizationID = owner.OrganizationID.String()
	case org.UserOwnedTeam:
		ownerUserID = owner.OwnerUserID.String()
	}
	return teamResponse{
		ID:             value.ID.String(),
		OwnerKind:      value.Owner.Kind().String(),
		OrganizationID: organizationID,
		OwnerUserID:    ownerUserID,
		Name:           value.Name.String(),
		CreatedBy:      value.CreatedBy.String(),
	}
}

type activeAssigneeParts struct {
	kind string
	id   string
}

type rewardResponseParts struct {
	kind             string
	amount           int64
	collectibleCount int
}

type responseParts struct {
	kind     string
	id       string
	position int
	source   string
}

func (server Server) writeAuthResponse(w http.ResponseWriter, status int, response authResponse) {
	// Stamp the platform role from the bootstrap admin allowlist so the client can
	// gate admin-only UI without a separate request.
	userIDResult := core.ParseUserID(response.SubjectID)
	userID, matched := userIDResult.(core.UserIDCreated)
	if matched && server.isPlatformAdmin(context.Background(), userID.Value) {
		response.Role = "admin"
	} else {
		response.Role = "member"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationResponse(w http.ResponseWriter, status int, response organizationResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationsResponse(w http.ResponseWriter, status int, response organizationsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationMemberResponse(w http.ResponseWriter, status int, response organizationMemberResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeOrganizationMembersResponse(w http.ResponseWriter, status int, response organizationMembersResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTeamResponse(w http.ResponseWriter, status int, response teamResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTeamsResponse(w http.ResponseWriter, status int, response teamsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTaskResponse(w http.ResponseWriter, status int, response taskResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeTasksResponse(w http.ResponseWriter, status int, response tasksResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeSubmissionCreatedResponse(w http.ResponseWriter, status int, response submissionCreatedResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeSubmissionResponse(w http.ResponseWriter, status int, response submissionResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeSubmissionsResponse(w http.ResponseWriter, status int, response submissionsResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeEmptyResponse(w http.ResponseWriter, status int, response emptyResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(response)
}

func writeError(w http.ResponseWriter, status int, code core.ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message, Code: code.String()})
}

func writeDomainError(w http.ResponseWriter, reason core.DomainError) {
	writeError(w, statusForError(reason), reason.Code(), reason.Description())
}

func statusForError(reason core.DomainError) int {
	switch reason.Code() {
	case core.ErrorCodeInvalidID, core.ErrorCodeInvalidEnum, core.ErrorCodeInvalidArgument:
		return http.StatusBadRequest
	case core.ErrorCodeInvalidState:
		return http.StatusConflict
	case core.ErrorCodeNotFound:
		return http.StatusNotFound
	case core.ErrorCodePermissionDenied:
		return http.StatusForbidden
	case core.ErrorCodeConflict:
		return http.StatusConflict
	case core.ErrorCodeUnauthenticated:
		return http.StatusUnauthorized
	case core.ErrorCodeRateLimited:
		return http.StatusTooManyRequests
	case core.ErrorCodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}
