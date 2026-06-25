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
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/mcp"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

type healthResponse struct {
	Status string `json:"status"`
}

type AuthService interface {
	Register(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.RegisterResult
	Login(context.Context, auth.EmailAddress, auth.PasswordSecret) auth.LoginResult
	Refresh(context.Context, auth.RefreshTokenPlain) auth.RefreshResult
	CreateGuest(context.Context) auth.GuestResult
}

type SubjectVerifier interface {
	Verify(auth.AccessToken) auth.SubjectVerifyResult
}

type OrganizationService interface {
	CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult
	ListOrganizations(context.Context, auth.UserSubject, core.Page) org.ListOrganizationsResult
	ListMembers(context.Context, auth.UserSubject, core.OrganizationID, core.Page) org.ListMembersResult
	ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult
	DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult
	CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult
	CreateStandaloneTeam(context.Context, auth.UserSubject, org.TeamName) org.CreateTeamResult
	ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID, core.Page) org.ListTeamsResult
	ListStandaloneTeams(context.Context, auth.UserSubject, core.Page) org.ListTeamsResult
	GetTeam(context.Context, auth.UserSubject, core.TeamID) org.GetTeamResult
	AddTeamMember(context.Context, auth.UserSubject, core.TeamID, auth.EmailAddress) org.AddTeamMemberResult
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
}

type TaskService interface {
	Create(context.Context, task.CreateCommand) task.CreateResult
	Get(context.Context, auth.UserSubject, core.TaskID) task.GetResult
	Open(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	Cancel(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	Unpublish(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	List(context.Context, auth.UserSubject, task.ListScope, task.ListFilters, core.Page) task.ListResult
	CreateCapabilityToken(context.Context, auth.UserSubject, core.TaskID) task.CreateCapabilityTokenResult
	ListSeries(context.Context, auth.UserSubject, core.Page) task.ListSeriesResult
	GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult
	CreateSeries(context.Context, auth.UserSubject, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult
	UpdateSeries(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesTitle, task.SeriesDescription) task.SeriesMutationResult
	ChangeSeriesState(context.Context, auth.UserSubject, core.TaskSeriesID, task.SeriesStateTransition) task.SeriesMutationResult
	AddTaskToSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	RemoveTaskFromSeries(context.Context, auth.UserSubject, core.TaskSeriesID, core.TaskID) task.SeriesMutationResult
	ReorderSeries(context.Context, auth.UserSubject, core.TaskSeriesID, []core.TaskID) task.SeriesMutationResult
	AddSeriesComment(context.Context, auth.UserSubject, core.TaskSeriesID, task.CommentBody) task.SeriesCommentResult
	ListSeriesComments(context.Context, auth.UserSubject, core.TaskSeriesID) task.SeriesCommentsResult
	AddTaskComment(context.Context, auth.UserSubject, core.TaskID, task.CommentBody) task.TaskCommentResult
	ListTaskComments(context.Context, auth.UserSubject, core.TaskID) task.TaskCommentsResult
	Reserve(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ApproveReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	DeclineReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	CancelReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	ListReservations(context.Context, auth.UserSubject, core.TaskID) task.ReservationsListResult
}

type AgentService interface {
	Create(context.Context, core.UserID, agent.Label, agent.ScopeSet) agent.CreateResult
	Verify(context.Context, agent.SecretPlain) agent.VerifyResult
	List(context.Context, core.UserID, core.Page) agent.ListResult
	Revoke(context.Context, core.UserID, core.AgentCredentialID) agent.RevokeResult
}

type AssetService interface {
	Mint(context.Context, string, string, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy, string) assets.MintResult
	ListCollectibles(context.Context, core.UserID, core.Page) assets.ListResult
	ListByOwner(context.Context, string, string, core.Page) assets.ListResult
	FundReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult
	RefundReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult
	GiftCollectible(context.Context, core.UserID, core.UserID, core.CollectibleID) assets.GiftResult
}

type SubmissionService interface {
	Submit(context.Context, submission.SubmitCommand) submission.SubmitResult
	FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	ListForTask(context.Context, auth.UserSubject, core.TaskID, core.Page) submission.ListResult
	ListForSubmitter(context.Context, auth.UserSubject, core.UserID) submission.ListResult
	AddSubmissionComment(context.Context, auth.UserSubject, core.SubmissionID, task.CommentBody) submission.SubmissionCommentResult
	ListSubmissionComments(context.Context, auth.UserSubject, core.SubmissionID) submission.SubmissionCommentsResult
}

type LedgerService interface {
	FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	FundTaskFromOrganization(context.Context, core.OrganizationID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	AcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey) ledger.AcceptResult
	ReviewAcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, ledger.CreditReviewSelection, ledger.TipSelection) ledger.AcceptResult
	RequestChanges(context.Context, core.UserID, core.TaskID, core.SubmissionID, submission.ReviewNote) ledger.RequestChangesResult
	RejectSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote, ledger.CreditReviewSelection, ledger.TipSelection, ledger.BanSelection) ledger.RejectResult
	RefundTask(context.Context, core.UserID, core.TaskID, ledger.IdempotencyKey) ledger.RefundResult
	Balance(context.Context, core.UserID) ledger.BalanceResult
	OrganizationBalance(context.Context, core.OrganizationID) ledger.BalanceResult
	ListEntries(context.Context, core.UserID, core.Page) ledger.ListEntriesResult
}

type Server struct {
	staticFiles         fs.FS
	authService         AuthService
	subjectVerifier     SubjectVerifier
	organizationService OrganizationService
	taskService         TaskService
	submissionService   SubmissionService
	ledgerService       LedgerService
	agentService        AgentService
	assetService        AssetService
	mcpServer           mcp.Server
	mcpSessions         *mcpHTTPSessionStore
	secureCookies       bool
	ipRateLimiter       *rateLimiter
	subjectRateLimiter  *rateLimiter
	adminUserIDs        map[string]bool
}

// Rate-limit budgets (burst capacity + steady refill per second): bound abusive
// volume on unauthenticated endpoints (by client IP) and MCP tool calls (by agent
// subject) without impeding normal use.
const (
	ipRateCapacity      = 20
	ipRateRefillPerSec  = 5
	mcpRateCapacity     = 60
	mcpRateRefillPerSec = 10
)

func New(staticFiles fs.FS, authService AuthService, subjectVerifier SubjectVerifier, organizationService OrganizationService, taskService TaskService, submissionService SubmissionService, ledgerService LedgerService, agentService AgentService, assetService AssetService) http.Handler {
	server := Server{
		staticFiles:         staticFiles,
		authService:         authService,
		subjectVerifier:     subjectVerifier,
		organizationService: organizationService,
		taskService:         taskService,
		submissionService:   submissionService,
		ledgerService:       ledgerService,
		agentService:        agentService,
		assetService:        assetService,
		mcpServer:           mcp.NewServer(mcpServices{taskService: taskService, submissionService: submissionService, ledgerService: ledgerService}),
		mcpSessions:         newMCPHTTPSessionStore(),
		// The refresh-token cookie is Secure by default; local plain-HTTP dev can
		// opt out explicitly with SHARECROP_INSECURE_COOKIES=true.
		secureCookies:      os.Getenv("SHARECROP_INSECURE_COOKIES") != "true",
		ipRateLimiter:      newRateLimiter(ipRateCapacity, ipRateRefillPerSec),
		subjectRateLimiter: newRateLimiter(mcpRateCapacity, mcpRateRefillPerSec),
		// Platform admins (e.g. for awarding default collectibles) are bootstrapped
		// from a comma-separated env list of user ids.
		adminUserIDs: parseAdminUserIDs(os.Getenv("SHARECROP_ADMIN_USER_IDS")),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health)
	mux.HandleFunc("POST /api/auth/register", server.register)
	mux.HandleFunc("POST /api/auth/login", server.login)
	mux.HandleFunc("POST /api/auth/refresh", server.refresh)
	mux.HandleFunc("POST /api/auth/logout", server.logout)
	mux.HandleFunc("POST /api/auth/guest", server.guest)
	mux.HandleFunc("GET /api/organizations", server.listOrganizations)
	mux.HandleFunc("POST /api/organizations", server.createOrganization)
	mux.HandleFunc("GET /api/organizations/{organization_id}/members", server.listOrganizationMembers)
	mux.HandleFunc("POST /api/organizations/{organization_id}/members", server.provisionOrganizationMember)
	mux.HandleFunc("PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate", server.deactivateOrganizationMember)
	mux.HandleFunc("GET /api/organizations/{organization_id}/teams", server.listOrganizationTeams)
	mux.HandleFunc("POST /api/organizations/{organization_id}/teams", server.createOrganizationTeam)
	mux.HandleFunc("GET /api/teams", server.listStandaloneTeams)
	mux.HandleFunc("POST /api/teams", server.createStandaloneTeam)
	mux.HandleFunc("GET /api/teams/{team_id}", server.getTeam)
	mux.HandleFunc("POST /api/teams/{team_id}/members", server.addTeamMember)
	mux.HandleFunc("GET /api/users/{user_id}", server.getUserProfile)
	mux.HandleFunc("GET /api/users/{user_id}/work", server.getUserWork)
	mux.HandleFunc("GET /api/users/{user_id}/submissions", server.getUserSubmissions)
	mux.HandleFunc("GET /api/tasks", server.listTasks)
	mux.HandleFunc("POST /api/tasks", server.createTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/open", server.openTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/cancel", server.cancelTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/capability-tokens", server.createTaskCapabilityToken)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions", server.createAuthenticatedSubmission)
	mux.HandleFunc("GET /api/tasks/{task_id}/submissions", server.listTaskSubmissions)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations", server.reserveTask)
	mux.HandleFunc("GET /api/tasks/{task_id}/reservations", server.listTaskReservations)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations/{reservation_id}/approve", server.approveTaskReservation)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations/{reservation_id}/decline", server.declineTaskReservation)
	mux.HandleFunc("POST /api/tasks/{task_id}/reservations/{reservation_id}/cancel", server.cancelTaskReservation)
	mux.HandleFunc("GET /api/submission-receipts/{receipt_token}", server.findSubmissionReceipt)
	mux.HandleFunc("GET /api/submissions/{submission_id}/comments", server.listSubmissionComments)
	mux.HandleFunc("POST /api/submissions/{submission_id}/comments", server.addSubmissionComment)
	mux.HandleFunc("GET /api/organizations/{organization_id}/credits/balance", server.organizationCreditsBalance)
	mux.HandleFunc("GET /api/credits/balance", server.creditsBalance)
	mux.HandleFunc("GET /api/credits/ledger", server.creditsLedger)
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
	mux.HandleFunc("POST /api/collectibles/{id}/transfer", server.transferCollectible)
	mux.HandleFunc("GET /api/organizations/{id}/collectibles", server.listOrganizationCollectibles)
	mux.HandleFunc("GET /api/teams/{id}/collectibles", server.listTeamCollectibles)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-reward", server.fundCollectibleReward)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-refund", server.refundCollectibleReward)
	mux.HandleFunc("POST /api/agent-credentials", server.createAgentCredential)
	mux.HandleFunc("GET /api/agent-credentials", server.listAgentCredentials)
	mux.HandleFunc("POST /api/agent-credentials/{credential_id}/revoke", server.revokeAgentCredential)
	mux.HandleFunc("POST /mcp", server.mcpEndpoint)
	mux.HandleFunc("GET /mcp", server.mcpStream)
	mux.HandleFunc("DELETE /mcp", server.mcpDeleteSession)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /", index(staticFiles))
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
func NewMCPServer(taskService TaskService, submissionService SubmissionService, ledgerService LedgerService) mcp.Server {
	return mcp.NewServer(mcpServices{taskService: taskService, submissionService: submissionService, ledgerService: ledgerService})
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
}

func index(staticFiles fs.FS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The browser app is a single-page application served from the same
		// shell for every in-app route, so deep links and refreshes load the
		// app. Unmatched API paths still return 404 rather than the shell.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		data, err := fs.ReadFile(staticFiles, "index.html")
		if err != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	}
}

func (balanceResponse) writableResponse() {}

func (ledgerListResponse) writableResponse() {}

func (taskEscrowResponse) writableResponse() {}

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

type taskReservationChanger func(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult

type taskStateChanger func(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult

type reviewPathResult interface {
	reviewPathResult()
}

type reviewPathAccepted struct {
	actor        auth.UserSubject
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

	emailResult := auth.NewEmailAddress(request.Email)
	emailAccepted, emailMatched := emailResult.(auth.EmailAddressAccepted)
	if !emailMatched {
		rejected := emailResult.(auth.EmailAddressRejected)
		return authRequestRejected{reason: rejected.Reason.Description()}
	}

	passwordResult := auth.NewPasswordSecret(request.Password)
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

// requireAdmin resolves a request to a platform-admin user subject, rejecting
// authenticated-but-non-admin users with a forbidden result.
func (server Server) requireAdmin(r *http.Request) userSubjectResult {
	accepted, matched := server.requireUserSubject(r).(userSubjectAccepted)
	if !matched {
		return userSubjectRejected{reason: "a user access token is required"}
	}
	if !server.adminUserIDs[accepted.subject.ID.String()] {
		return userSubjectRejected{reason: "this action requires a platform administrator"}
	}
	return accepted
}

// requireWorkerSubject resolves a request to an acting user subject from either
// a user access token or an agent credential that holds the required scope. This
// lets a single agent token drive the worker REST endpoints as well as MCP (an
// agent credential always acts as its owning user, exactly as it does over MCP).
func (server Server) requireWorkerSubject(r *http.Request, scope agent.Scope) userSubjectResult {
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
	return userSubjectAccepted{subject: verified.Subject}
}

func (server Server) requireUserSubject(r *http.Request) userSubjectResult {
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

	roles := make([]org.Role, 0, len(request.Roles))
	for _, rawRole := range request.Roles {
		roleResult := org.ParseRole(rawRole)
		roleAccepted, roleMatched := roleResult.(org.RoleAccepted)
		if !roleMatched {
			rejected := roleResult.(org.RoleRejected)
			return provisionMemberRejected{reason: rejected.Reason.Description()}
		}
		roles = append(roles, roleAccepted.Value)
	}

	if len(roles) == 0 {
		return provisionMemberRejected{reason: "at least one organization role is required"}
	}

	return provisionMemberAccepted{email: emailAccepted.Value, roles: roles}
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
	value task.RewardSpec
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

func parsePage(r *http.Request) core.Page {
	query := r.URL.Query()
	rawLimit := query.Get("limit")
	rawOffset := query.Get("offset")
	if rawLimit == "" && rawOffset == "" {
		return core.DefaultPage()
	}
	limit, limitErr := strconv.Atoi(rawLimit)
	if limitErr != nil {
		limit = core.DefaultPage().Limit()
	}
	offset, offsetErr := strconv.Atoi(rawOffset)
	if offsetErr != nil {
		offset = core.DefaultPage().Offset()
	}
	pageResult := core.NewPage(limit, offset)
	accepted, matched := pageResult.(core.PageAccepted)
	if !matched {
		return core.DefaultPage()
	}
	return accepted.Value
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
	if !server.ipRateLimiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "too many requests; slow down and retry")
		return false
	}
	return true
}

// allowBySubject rate-limits an authenticated, DB-heavy endpoint by acting
// subject so a single account cannot spam transactional review operations.
func (server Server) allowBySubject(w http.ResponseWriter, subjectID string) bool {
	if !server.subjectRateLimiter.allow(subjectID) {
		writeError(w, http.StatusTooManyRequests, "too many requests; slow down and retry")
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

func writeAuthResponse(w http.ResponseWriter, status int, response authResponse) {
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

func writeTaskCapabilityTokenResponse(w http.ResponseWriter, status int, response taskCapabilityTokenResponse) {
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

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorResponse{Error: message})
}

func writeDomainError(w http.ResponseWriter, reason core.DomainError) {
	writeError(w, statusForError(reason), reason.Description())
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
	default:
		return http.StatusInternalServerError
	}
}
