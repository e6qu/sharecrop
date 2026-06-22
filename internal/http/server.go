package httpserver

import (
	"context"
	"encoding/json"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/mcp"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/schema"
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
	ListOrganizations(context.Context, auth.UserSubject) org.ListOrganizationsResult
	ProvisionMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult
	DeactivateMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult
	CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult
	ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID) org.ListTeamsResult
	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
}

type TaskService interface {
	Create(context.Context, task.CreateCommand) task.CreateResult
	Get(context.Context, auth.UserSubject, core.TaskID) task.GetResult
	Open(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	Cancel(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult
	List(context.Context, auth.UserSubject, task.ListScope) task.ListResult
	CreateCapabilityToken(context.Context, auth.UserSubject, core.TaskID) task.CreateCapabilityTokenResult
	ListSeries(context.Context, auth.UserSubject) task.ListSeriesResult
	GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult
	Reserve(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ApproveReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	DeclineReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	CancelReservation(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	ListReservations(context.Context, auth.UserSubject, core.TaskID) task.ReservationsListResult
}

type AgentService interface {
	Create(context.Context, core.UserID, agent.Label, agent.ScopeSet) agent.CreateResult
	Verify(context.Context, agent.SecretPlain) agent.VerifyResult
	List(context.Context, core.UserID) agent.ListResult
	Revoke(context.Context, core.UserID, core.AgentCredentialID) agent.RevokeResult
}

type AssetService interface {
	Mint(context.Context, core.UserID, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy) assets.MintResult
	ListCollectibles(context.Context, core.UserID) assets.ListResult
	FundReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult
	RefundReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult
}

type SubmissionService interface {
	Submit(context.Context, submission.SubmitCommand) submission.SubmitResult
	FindByReceipt(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	ListForTask(context.Context, auth.UserSubject, core.TaskID) submission.ListResult
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
	ListEntries(context.Context, core.UserID) ledger.ListEntriesResult
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
}

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
	mux.HandleFunc("POST /api/organizations/{organization_id}/members", server.provisionOrganizationMember)
	mux.HandleFunc("PATCH /api/organizations/{organization_id}/members/{user_id}/deactivate", server.deactivateOrganizationMember)
	mux.HandleFunc("GET /api/organizations/{organization_id}/teams", server.listOrganizationTeams)
	mux.HandleFunc("POST /api/organizations/{organization_id}/teams", server.createOrganizationTeam)
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
	mux.HandleFunc("GET /api/organizations/{organization_id}/credits/balance", server.organizationCreditsBalance)
	mux.HandleFunc("GET /api/credits/balance", server.creditsBalance)
	mux.HandleFunc("GET /api/credits/ledger", server.creditsLedger)
	mux.HandleFunc("POST /api/tasks/{task_id}/funding", server.fundTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/refund", server.refundTask)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/accept", server.acceptSubmission)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/request-changes", server.requestSubmissionChanges)
	mux.HandleFunc("POST /api/tasks/{task_id}/submissions/{submission_id}/reject", server.rejectSubmission)
	mux.HandleFunc("GET /api/tasks/{task_id}", server.getTask)
	mux.HandleFunc("GET /api/task-series", server.listTaskSeries)
	mux.HandleFunc("GET /api/task-series/{series_id}", server.getTaskSeries)
	mux.HandleFunc("POST /api/collectibles", server.mintCollectible)
	mux.HandleFunc("GET /api/collectibles", server.listCollectibles)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-reward", server.fundCollectibleReward)
	mux.HandleFunc("POST /api/tasks/{task_id}/collectible-refund", server.refundCollectibleReward)
	mux.HandleFunc("POST /api/agent-credentials", server.createAgentCredential)
	mux.HandleFunc("GET /api/agent-credentials", server.listAgentCredentials)
	mux.HandleFunc("POST /api/agent-credentials/{credential_id}/revoke", server.revokeAgentCredential)
	mux.HandleFunc("POST /mcp", server.mcpEndpoint)
	mux.HandleFunc("GET /mcp", server.mcpStreamNotOffered)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.HandleFunc("GET /", index(staticFiles))
	return mux
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
		if r.URL.Path != "/" {
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

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	SubjectKind string `json:"subject_kind"`
	SubjectID   string `json:"subject_id"`
	AccessToken string `json:"access_token"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type organizationRequest struct {
	Name string `json:"name"`
}

type provisionMemberRequest struct {
	Email string   `json:"email"`
	Roles []string `json:"roles"`
}

type teamRequest struct {
	Name string `json:"name"`
}

type taskOwnerRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskVisibilityRequest struct {
	Kind           string `json:"kind"`
	UserID         string `json:"user_id"`
	TeamID         string `json:"team_id"`
	OrganizationID string `json:"organization_id"`
}

type taskPlacementRequest struct {
	Kind           string `json:"kind"`
	SeriesID       string `json:"series_id"`
	SeriesTitle    string `json:"series_title"`
	SeriesPosition int    `json:"series_position"`
}

type taskPayloadRequest struct {
	Kind string `json:"kind"`
	JSON string `json:"json"`
}

type taskRequest struct {
	Owner              taskOwnerRequest         `json:"owner"`
	Title              string                   `json:"title"`
	Description        string                   `json:"description"`
	Reward             taskRewardRequest        `json:"reward"`
	Participation      taskParticipationRequest `json:"participation"`
	Visibility         taskVisibilityRequest    `json:"visibility"`
	Placement          taskPlacementRequest     `json:"placement"`
	ResponseSchemaJSON string                   `json:"response_schema_json"`
	Payload            taskPayloadRequest       `json:"payload"`
}

type taskRewardRequest struct {
	Kind         string `json:"kind"`
	CreditAmount int64  `json:"credit_amount"`
}

type taskParticipationRequest struct {
	Policy                 string `json:"policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
}

type submissionRequest struct {
	ResponseJSON string `json:"response_json"`
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedBy string `json:"created_by"`
}

type organizationsResponse struct {
	Organizations []organizationResponse `json:"organizations"`
}

type organizationMemberResponse struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	UserID         string   `json:"user_id"`
	Status         string   `json:"status"`
	Roles          []string `json:"roles"`
}

type teamResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	CreatedBy      string `json:"created_by"`
}

type teamsResponse struct {
	Teams []teamResponse `json:"teams"`
}

type taskResponse struct {
	ID                     string `json:"id"`
	OwnerKind              string `json:"owner_kind"`
	OwnerID                string `json:"owner_id"`
	Title                  string `json:"title"`
	Description            string `json:"description"`
	RewardKind             string `json:"reward_kind"`
	RewardCreditAmount     int64  `json:"reward_credit_amount"`
	RewardCollectibleCount int    `json:"reward_collectible_count"`
	ParticipationPolicy    string `json:"participation_policy"`
	AssigneeScope          string `json:"assignee_scope"`
	ReservationExpiryHours int    `json:"reservation_expiry_hours"`
	State                  string `json:"state"`
	VisibilityKind         string `json:"visibility_kind"`
	VisibilityID           string `json:"visibility_id"`
	SeriesKind             string `json:"series_kind"`
	SeriesID               string `json:"series_id"`
	SeriesPosition         int    `json:"series_position"`
	ResponseSchemaJSON     string `json:"response_schema_json"`
	PayloadKind            string `json:"payload_kind"`
	PayloadJSON            string `json:"payload_json"`
	CreatedBy              string `json:"created_by"`
	AvailabilityKind       string `json:"availability_kind"`
	ViewerAction           string `json:"viewer_action"`
}

type tasksResponse struct {
	Tasks []taskResponse `json:"tasks"`
}

type taskCapabilityTokenResponse struct {
	ID     string `json:"id"`
	TaskID string `json:"task_id"`
	State  string `json:"state"`
	Token  string `json:"token"`
}

type reservationResponse struct {
	ID           string `json:"id"`
	TaskID       string `json:"task_id"`
	AssigneeKind string `json:"assignee_kind"`
	AssigneeID   string `json:"assignee_id"`
	State        string `json:"state"`
	RequestedBy  string `json:"requested_by"`
}

type reservationsResponse struct {
	Reservations []reservationResponse `json:"reservations"`
}

type submissionValidationErrorResponse struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type submissionResponse struct {
	ID               string                              `json:"id"`
	TaskID           string                              `json:"task_id"`
	SubmitterID      string                              `json:"submitter_id"`
	State            string                              `json:"state"`
	ResponseJSON     string                              `json:"response_json"`
	ReviewNote       string                              `json:"review_note"`
	ValidationErrors []submissionValidationErrorResponse `json:"validation_errors"`
}

type submissionsResponse struct {
	Submissions []submissionResponse `json:"submissions"`
}

type submissionCreatedResponse struct {
	Submission   submissionResponse `json:"submission"`
	ReceiptToken string             `json:"receipt_token"`
}

type emptyResponse struct {
	Status string `json:"status"`
}

type fundingRequest struct {
	Amount         int64  `json:"amount"`
	IdempotencyKey string `json:"idempotency_key"`
	OrganizationID string `json:"organization_id"`
}

type idempotentRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type acceptSubmissionRequest struct {
	IdempotencyKey string `json:"idempotency_key"`
	PayoutAmount   int64  `json:"payout_amount"`
	TipAmount      int64  `json:"tip_amount"`
}

type requestChangesRequest struct {
	ReviewNote string `json:"review_note"`
}

type rejectSubmissionRequest struct {
	IdempotencyKey      string `json:"idempotency_key"`
	ReviewNote          string `json:"review_note"`
	PartialCreditAmount int64  `json:"partial_credit_amount"`
	TipAmount           int64  `json:"tip_amount"`
	BanImplementor      bool   `json:"ban_implementor"`
}

type writableResponse interface {
	writableResponse()
}

type balanceResponse struct {
	Amount int64 `json:"amount"`
}

type ledgerEntryResponse struct {
	ID     string `json:"id"`
	Kind   string `json:"kind"`
	Amount int64  `json:"amount"`
	TaskID string `json:"task_id"`
}

type ledgerListResponse struct {
	Entries []ledgerEntryResponse `json:"entries"`
}

type taskEscrowResponse struct {
	TaskID string `json:"task_id"`
	Amount int64  `json:"amount"`
	State  string `json:"state"`
}

type acceptSubmissionResponse struct {
	TaskID        string `json:"task_id"`
	SubmissionID  string `json:"submission_id"`
	PayoutKind    string `json:"payout_kind"`
	PayoutAmount  int64  `json:"payout_amount"`
	WorkerUserID  string `json:"worker_user_id"`
	CollectibleID string `json:"collectible_id"`
	TipAmount     int64  `json:"tip_amount"`
}

type reviewSubmissionResponse struct {
	TaskID       string `json:"task_id"`
	SubmissionID string `json:"submission_id"`
	State        string `json:"state"`
	ReviewNote   string `json:"review_note"`
	PayoutKind   string `json:"payout_kind"`
	PayoutAmount int64  `json:"payout_amount"`
	WorkerUserID string `json:"worker_user_id"`
	TipAmount    int64  `json:"tip_amount"`
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

func (server Server) register(w http.ResponseWriter, r *http.Request) {
	requestResult := decodeAuthRequest(r)
	requestAccepted, requestMatched := requestResult.(authRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(authRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.authService.Register(r.Context(), requestAccepted.email, requestAccepted.password)
	accepted, matched := result.(auth.RegisterAccepted)
	if !matched {
		rejected := result.(auth.RegisterRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) login(w http.ResponseWriter, r *http.Request) {
	requestResult := decodeAuthRequest(r)
	requestAccepted, requestMatched := requestResult.(authRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(authRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.authService.Login(r.Context(), requestAccepted.email, requestAccepted.password)
	accepted, matched := result.(auth.LoginAccepted)
	if !matched {
		rejected := result.(auth.LoginRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusOK, authResponse{
		SubjectKind: "user",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("sharecrop_refresh_token")
	if err != nil {
		writeError(w, http.StatusUnauthorized, "refresh token is required")
		return
	}

	tokenResult := auth.ParseRefreshTokenPlain(cookie.Value)
	tokenAccepted, tokenMatched := tokenResult.(auth.RefreshTokenPlainAccepted)
	if !tokenMatched {
		rejected := tokenResult.(auth.RefreshTokenPlainRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	result := server.authService.Refresh(r.Context(), tokenAccepted.Value)
	accepted, matched := result.(auth.RefreshAccepted)
	if !matched {
		rejected := result.(auth.RefreshRejected)
		writeError(w, http.StatusUnauthorized, rejected.Reason.Description())
		return
	}

	setRefreshCookie(w, accepted.RefreshToken)
	responseResult := authResponseForSubject(accepted.Subject, accepted.AccessToken)
	responseAccepted, responseMatched := responseResult.(authResponseAccepted)
	if !responseMatched {
		rejected := responseResult.(authResponseRejected)
		writeError(w, http.StatusInternalServerError, rejected.reason)
		return
	}

	writeAuthResponse(w, http.StatusOK, responseAccepted.response)
}

func (server Server) logout(w http.ResponseWriter, r *http.Request) {
	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (server Server) guest(w http.ResponseWriter, r *http.Request) {
	result := server.authService.CreateGuest(r.Context())
	accepted, matched := result.(auth.GuestAccepted)
	if !matched {
		rejected := result.(auth.GuestRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	setRefreshCookie(w, accepted.RefreshToken)
	writeAuthResponse(w, http.StatusCreated, authResponse{
		SubjectKind: "guest",
		SubjectID:   accepted.Subject.ID.String(),
		AccessToken: accepted.AccessToken.String(),
	})
}

func (server Server) createOrganization(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	var request organizationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewOrganizationName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.OrganizationNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.OrganizationNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganization(r.Context(), actor.subject, nameAccepted.Value)
	created, matched := result.(org.OrganizationCreated)
	if !matched {
		rejected := result.(org.CreateOrganizationRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	writeOrganizationResponse(w, http.StatusCreated, organizationToResponse(created.Value))
}

func (server Server) listOrganizations(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizations(r.Context(), actor.subject)
	listed, matched := result.(org.OrganizationsListed)
	if !matched {
		rejected := result.(org.ListOrganizationsRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	response := organizationsResponse{Organizations: make([]organizationResponse, 0, len(listed.Values))}
	for _, organization := range listed.Values {
		response.Organizations = append(response.Organizations, organizationToResponse(organization))
	}
	writeOrganizationsResponse(w, http.StatusOK, response)
}

func (server Server) provisionOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	requestResult := decodeProvisionMemberRequest(r)
	requestAccepted, requestMatched := requestResult.(provisionMemberAccepted)
	if !requestMatched {
		rejected := requestResult.(provisionMemberRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ProvisionMember(r.Context(), actor.subject, organizationIDAccepted.value, requestAccepted.email, requestAccepted.roles)
	provisioned, matched := result.(org.MemberProvisioned)
	if !matched {
		rejected := result.(org.ProvisionMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeOrganizationMemberResponse(w, http.StatusCreated, memberToResponse(provisioned.Value))
}

func (server Server) deactivateOrganizationMember(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	userIDResult := core.ParseUserID(r.PathValue("user_id"))
	userIDAccepted, userIDMatched := userIDResult.(core.UserIDCreated)
	if !userIDMatched {
		rejected := userIDResult.(core.UserIDRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.DeactivateMember(r.Context(), actor.subject, organizationIDAccepted.value, userIDAccepted.Value)
	if _, matched := result.(org.MemberDeactivationAccepted); !matched {
		rejected := result.(org.DeactivateMemberRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeEmptyResponse(w, http.StatusOK, emptyResponse{Status: "deactivated"})
}

func (server Server) createOrganizationTeam(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	var request teamRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	nameResult := org.NewTeamName(request.Name)
	nameAccepted, nameMatched := nameResult.(org.TeamNameAccepted)
	if !nameMatched {
		rejected := nameResult.(org.TeamNameRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.organizationService.CreateOrganizationTeam(r.Context(), actor.subject, organizationIDAccepted.value, nameAccepted.Value)
	created, matched := result.(org.TeamCreated)
	if !matched {
		rejected := result.(org.CreateTeamRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	writeTeamResponse(w, http.StatusCreated, teamToResponse(created.Value))
}

func (server Server) listOrganizationTeams(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	organizationIDResult := parseOrganizationPathValue(r)
	organizationIDAccepted, organizationIDMatched := organizationIDResult.(organizationIDAccepted)
	if !organizationIDMatched {
		rejected := organizationIDResult.(organizationIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.organizationService.ListOrganizationTeams(r.Context(), actor.subject, organizationIDAccepted.value)
	listed, matched := result.(org.OrganizationTeamsListed)
	if !matched {
		rejected := result.(org.ListTeamsRejected)
		writeError(w, http.StatusForbidden, rejected.Reason.Description())
		return
	}

	response := teamsResponse{Teams: make([]teamResponse, 0, len(listed.Values))}
	for _, team := range listed.Values {
		response.Teams = append(response.Teams, teamToResponse(team))
	}
	writeTeamsResponse(w, http.StatusOK, response)
}

func (server Server) createTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	requestResult := decodeTaskRequest(r, actor.subject)
	requestAccepted, requestMatched := requestResult.(taskRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(taskRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.taskService.Create(r.Context(), requestAccepted.command)
	created, matched := result.(task.TaskCreated)
	if !matched {
		rejected := result.(task.CreateRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskResponse(w, http.StatusCreated, taskToResponse(created.Value))
}

func (server Server) listTasks(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	scopeResult := parseTaskListScope(r, actor.subject)
	scopeAccepted, scopeMatched := scopeResult.(taskListScopeAccepted)
	if !scopeMatched {
		rejected := scopeResult.(taskListScopeRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.taskService.List(r.Context(), actor.subject, scopeAccepted.value)
	switch listed := result.(type) {
	case task.ListRejected:
		writeDomainError(w, listed.Reason)
	case task.TasksListed:
		writeTasksResponse(w, http.StatusOK, tasksToResponse(listed.Values))
	}
}

func tasksToResponse(values []task.Task) tasksResponse {
	response := tasksResponse{Tasks: make([]taskResponse, 0, len(values))}
	for valueIndex := range values {
		response.Tasks = append(response.Tasks, taskToResponse(values[valueIndex]))
	}
	return response
}

func (server Server) openTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r, server.taskService.Open)
}

func (server Server) cancelTask(w http.ResponseWriter, r *http.Request) {
	server.changeTaskState(w, r, server.taskService.Cancel)
}

func (server Server) reserveTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	if rejected, matched := taskIDResult.(taskIDRejected); matched {
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}
	taskIDAccepted := taskIDResult.(taskIDAccepted)

	result := server.taskService.Reserve(r.Context(), actor.subject, taskIDAccepted.value)
	created, matched := result.(task.ReservationCreated)
	if !matched {
		writeDomainError(w, result.(task.ReservationRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, reservationToResponse(created.Value))
}

func (server Server) listTaskReservations(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	result := server.taskService.ListReservations(r.Context(), actor.subject, taskIDAccepted.value)
	listed, matched := result.(task.ReservationsListed)
	if !matched {
		writeDomainError(w, result.(task.ReservationsListRejected).Reason)
		return
	}
	response := reservationsResponse{Reservations: make([]reservationResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		response.Reservations = append(response.Reservations, reservationToResponse(value))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) approveTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.ApproveReservation)
}

func (server Server) declineTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.DeclineReservation)
}

func (server Server) cancelTaskReservation(w http.ResponseWriter, r *http.Request) {
	server.changeTaskReservation(w, r, server.taskService.CancelReservation)
}

type taskReservationChanger func(context.Context, auth.UserSubject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult

func (server Server) changeTaskReservation(w http.ResponseWriter, r *http.Request, changer taskReservationChanger) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}
	reservationIDResult := parseReservationPathValue(r)
	reservationIDAccepted, reservationIDMatched := reservationIDResult.(reservationIDAccepted)
	if !reservationIDMatched {
		writeError(w, http.StatusBadRequest, reservationIDResult.(reservationIDRejected).reason)
		return
	}

	result := changer(r.Context(), actor.subject, taskIDAccepted.value, reservationIDAccepted.value)
	changed, matched := result.(task.ReservationStateChanged)
	if !matched {
		writeDomainError(w, result.(task.ReservationStateChangeRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, reservationToResponse(changed.Value))
}

type taskStateChanger func(context.Context, auth.UserSubject, core.TaskID) task.ChangeStateResult

func (server Server) changeTaskState(w http.ResponseWriter, r *http.Request, changer taskStateChanger) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := changer(r.Context(), actor.subject, taskIDAccepted.value)
	changed, matched := result.(task.TaskStateChanged)
	if !matched {
		rejected := result.(task.ChangeStateRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskResponse(w, http.StatusOK, taskToResponse(changed.Value))
}

func (server Server) createTaskCapabilityToken(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.taskService.CreateCapabilityToken(r.Context(), actor.subject, taskIDAccepted.value)
	created, matched := result.(task.CapabilityTokenCreated)
	if !matched {
		rejected := result.(task.CreateCapabilityTokenRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeTaskCapabilityTokenResponse(w, http.StatusCreated, taskCapabilityTokenResponse{
		ID:     created.Value.ID.String(),
		TaskID: created.Value.TaskID.String(),
		State:  created.Value.State.String(),
		Token:  created.Plain.String(),
	})
}

func (server Server) createAuthenticatedSubmission(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	requestResult := decodeAuthenticatedSubmissionRequest(r, actor.subject, taskIDAccepted.value)
	requestAccepted, requestMatched := requestResult.(submissionRequestAccepted)
	if !requestMatched {
		rejected := requestResult.(submissionRequestRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	server.submitResponse(w, r, requestAccepted.command)
}

func (server Server) submitResponse(w http.ResponseWriter, r *http.Request, command submission.SubmitCommand) {
	result := server.submissionService.Submit(r.Context(), command)
	created, matched := result.(submission.SubmissionCreated)
	if !matched {
		rejected := result.(submission.SubmitRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeSubmissionCreatedResponse(w, http.StatusCreated, submissionCreatedResponse{
		Submission:   submissionToResponse(created.Value),
		ReceiptToken: created.ReceiptToken.String(),
	})
}

func (server Server) findSubmissionReceipt(w http.ResponseWriter, r *http.Request) {
	tokenResult := submission.ParseReceiptTokenPlain(r.PathValue("receipt_token"))
	tokenAccepted, tokenMatched := tokenResult.(submission.ReceiptTokenPlainAccepted)
	if !tokenMatched {
		rejected := tokenResult.(submission.ReceiptTokenPlainRejected)
		writeError(w, http.StatusBadRequest, rejected.Reason.Description())
		return
	}

	result := server.submissionService.FindByReceipt(r.Context(), tokenAccepted.Value)
	found, matched := result.(submission.ReceiptStatusFound)
	if !matched {
		rejected := result.(submission.ReceiptStatusRejected)
		writeError(w, http.StatusNotFound, rejected.Reason.Description())
		return
	}

	writeSubmissionResponse(w, http.StatusOK, submissionToResponse(found.Value))
}

func (server Server) listTaskSubmissions(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		rejected := taskIDResult.(taskIDRejected)
		writeError(w, http.StatusBadRequest, rejected.reason)
		return
	}

	result := server.submissionService.ListForTask(r.Context(), actor.subject, taskIDAccepted.value)
	listed, matched := result.(submission.SubmissionsListed)
	if !matched {
		rejected := result.(submission.ListRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	response := submissionsResponse{Submissions: make([]submissionResponse, 0, len(listed.Values))}
	for _, value := range listed.Values {
		response.Submissions = append(response.Submissions, submissionToResponse(value))
	}
	writeSubmissionsResponse(w, http.StatusOK, response)
}

func (server Server) creditsBalance(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.ledgerService.Balance(r.Context(), actor.subject.ID)
	found, matched := result.(ledger.BalanceFound)
	if !matched {
		rejected := result.(ledger.BalanceRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	writeJSON(w, http.StatusOK, balanceResponse{Amount: found.Value.Int64()})
}

func (server Server) creditsLedger(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	result := server.ledgerService.ListEntries(r.Context(), actor.subject.ID)
	listed, matched := result.(ledger.EntriesListed)
	if !matched {
		rejected := result.(ledger.ListEntriesRejected)
		writeDomainError(w, rejected.Reason)
		return
	}

	response := ledgerListResponse{Entries: make([]ledgerEntryResponse, 0, len(listed.Values))}
	for index := range listed.Values {
		response.Entries = append(response.Entries, ledgerEntryToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) fundTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request fundingRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	amountResult := ledger.NewCreditAmount(request.Amount)
	amount, amountMatched := amountResult.(ledger.CreditAmountAccepted)
	if !amountMatched {
		writeError(w, http.StatusBadRequest, amountResult.(ledger.CreditAmountRejected).Reason.Description())
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}

	if request.OrganizationID != "" {
		server.fundTaskFromOrganization(w, r, actor.subject, taskIDAccepted.value, amount.Value, key.Value, request.OrganizationID)
		return
	}

	result := server.ledgerService.FundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, amount.Value, key.Value)
	funded, matched := result.(ledger.TaskFunded)
	if !matched {
		writeDomainError(w, result.(ledger.FundRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, escrowToResponse(funded.Escrow))
}

func (server Server) refundTask(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	var request acceptSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}

	result := server.ledgerService.RefundTask(r.Context(), actor.subject.ID, taskIDAccepted.value, key.Value)
	refunded, matched := result.(ledger.TaskRefunded)
	if !matched {
		writeDomainError(w, result.(ledger.RefundRejected).Reason)
		return
	}

	writeJSON(w, http.StatusOK, escrowToResponse(refunded.Escrow))
}

func (server Server) acceptSubmission(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		rejected := actorResult.(userSubjectRejected)
		writeError(w, http.StatusUnauthorized, rejected.reason)
		return
	}

	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return
	}

	submissionIDResult := core.ParseSubmissionID(r.PathValue("submission_id"))
	submissionIDAccepted, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		writeError(w, http.StatusBadRequest, submissionIDResult.(core.SubmissionIDRejected).Reason.Description())
		return
	}

	var request acceptSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}

	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}

	creditSelectionResult := acceptCreditSelection(request.PayoutAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(creditSelectionAccepted)
	if !creditSelectionMatched {
		writeError(w, http.StatusBadRequest, creditSelectionResult.(creditSelectionRejected).reason)
		return
	}
	tipSelectionResult := tipSelectionFromAmount(request.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(tipSelectionAccepted)
	if !tipSelectionMatched {
		writeError(w, http.StatusBadRequest, tipSelectionResult.(tipSelectionRejected).reason)
		return
	}

	result := server.ledgerService.ReviewAcceptSubmission(r.Context(), actor.subject.ID, taskIDAccepted.value, submissionIDAccepted.Value, key.Value, creditSelection.value, tipSelection.value)
	accepted, matched := result.(ledger.SubmissionAccepted)
	if !matched {
		writeDomainError(w, result.(ledger.AcceptRejected).Reason)
		return
	}

	writeJSON(w, http.StatusOK, acceptToResponse(accepted))
}

func (server Server) requestSubmissionChanges(w http.ResponseWriter, r *http.Request) {
	pathResult := server.parseReviewPath(w, r)
	path, pathMatched := pathResult.(reviewPathAccepted)
	if !pathMatched {
		return
	}

	var request requestChangesRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	noteResult := submission.NewRequiredReviewNote(request.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		writeError(w, http.StatusBadRequest, noteResult.(submission.ReviewNoteRejected).Reason.Description())
		return
	}

	result := server.ledgerService.RequestChanges(r.Context(), path.actor.ID, path.taskID, path.submissionID, note.Value)
	changed, matched := result.(ledger.ChangesRequested)
	if !matched {
		writeDomainError(w, result.(ledger.RequestChangesRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, reviewSubmissionResponse{
		TaskID:       changed.TaskID.String(),
		SubmissionID: changed.SubmissionID.String(),
		State:        "changes_requested",
		ReviewNote:   changed.ReviewNote,
		PayoutKind:   "none",
	})
}

func (server Server) rejectSubmission(w http.ResponseWriter, r *http.Request) {
	pathResult := server.parseReviewPath(w, r)
	path, pathMatched := pathResult.(reviewPathAccepted)
	if !pathMatched {
		return
	}

	var request rejectSubmissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "request body is invalid")
		return
	}
	keyResult := ledger.NewIdempotencyKey(request.IdempotencyKey)
	key, keyMatched := keyResult.(ledger.IdempotencyKeyAccepted)
	if !keyMatched {
		writeError(w, http.StatusBadRequest, keyResult.(ledger.IdempotencyKeyRejected).Reason.Description())
		return
	}
	noteResult := submission.NewRequiredReviewNote(request.ReviewNote)
	note, noteMatched := noteResult.(submission.ReviewNoteAccepted)
	if !noteMatched {
		writeError(w, http.StatusBadRequest, noteResult.(submission.ReviewNoteRejected).Reason.Description())
		return
	}
	creditSelectionResult := rejectCreditSelection(request.PartialCreditAmount)
	creditSelection, creditSelectionMatched := creditSelectionResult.(creditSelectionAccepted)
	if !creditSelectionMatched {
		writeError(w, http.StatusBadRequest, creditSelectionResult.(creditSelectionRejected).reason)
		return
	}
	tipSelectionResult := tipSelectionFromAmount(request.TipAmount)
	tipSelection, tipSelectionMatched := tipSelectionResult.(tipSelectionAccepted)
	if !tipSelectionMatched {
		writeError(w, http.StatusBadRequest, tipSelectionResult.(tipSelectionRejected).reason)
		return
	}
	banSelection := ledger.BanSelection(ledger.NoBanSelection{})
	if request.BanImplementor {
		banSelection = ledger.BanImplementorSelection{}
	}

	result := server.ledgerService.RejectSubmission(r.Context(), path.actor.ID, path.taskID, path.submissionID, key.Value, note.Value, creditSelection.value, tipSelection.value, banSelection)
	rejected, matched := result.(ledger.SubmissionRejected)
	if !matched {
		writeDomainError(w, result.(ledger.RejectRejected).Reason)
		return
	}
	response := reviewOutcomeToResponse(rejected.Payout, rejected.Tip)
	response.TaskID = rejected.TaskID.String()
	response.SubmissionID = rejected.SubmissionID.String()
	response.State = "rejected"
	response.ReviewNote = note.Value.String()
	writeJSON(w, http.StatusOK, response)
}

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

func (server Server) parseReviewPath(w http.ResponseWriter, r *http.Request) reviewPathResult {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, actorResult.(userSubjectRejected).reason)
		return reviewPathRejected{}
	}
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, taskIDResult.(taskIDRejected).reason)
		return reviewPathRejected{}
	}
	submissionIDResult := core.ParseSubmissionID(r.PathValue("submission_id"))
	submissionIDAccepted, submissionIDMatched := submissionIDResult.(core.SubmissionIDCreated)
	if !submissionIDMatched {
		writeError(w, http.StatusBadRequest, submissionIDResult.(core.SubmissionIDRejected).Reason.Description())
		return reviewPathRejected{}
	}
	return reviewPathAccepted{actor: actor.subject, taskID: taskIDAccepted.value, submissionID: submissionIDAccepted.Value}
}

func ledgerEntryToResponse(entry ledger.LedgerEntry) ledgerEntryResponse {
	taskID := ""
	if referenced, matched := entry.TaskRef.(ledger.TaskReferenced); matched {
		taskID = referenced.TaskID.String()
	}
	return ledgerEntryResponse{
		ID:     entry.ID.String(),
		Kind:   entry.Kind.String(),
		Amount: entry.Amount.Int64(),
		TaskID: taskID,
	}
}

func escrowToResponse(escrow ledger.TaskEscrow) taskEscrowResponse {
	return taskEscrowResponse{
		TaskID: escrow.TaskID.String(),
		Amount: escrow.Amount.Int64(),
		State:  escrow.State.String(),
	}
}

func acceptToResponse(accepted ledger.SubmissionAccepted) acceptSubmissionResponse {
	response := acceptSubmissionResponse{
		TaskID:       accepted.TaskID.String(),
		SubmissionID: accepted.SubmissionID.String(),
		PayoutKind:   "none",
	}
	switch payout := accepted.Payout.(type) {
	case ledger.CreditPayout:
		response.PayoutKind = "credit"
		response.PayoutAmount = payout.Amount.Int64()
		response.WorkerUserID = payout.WorkerUserID.String()
	case ledger.CollectiblePayout:
		response.PayoutKind = "collectible"
		response.CollectibleID = payout.CollectibleID.String()
		response.WorkerUserID = payout.WorkerUserID.String()
	case ledger.BundlePayout:
		response.PayoutKind = "bundle"
		response.PayoutAmount = payout.Amount.Int64()
		response.CollectibleID = payout.CollectibleID.String()
		response.WorkerUserID = payout.WorkerUserID.String()
	}
	if tip, matched := accepted.Tip.(ledger.CreditTip); matched {
		response.TipAmount = tip.Amount.Int64()
	}
	return response
}

func reviewOutcomeToResponse(payout ledger.PayoutOutcome, tip ledger.TipOutcome) reviewSubmissionResponse {
	response := reviewSubmissionResponse{PayoutKind: "none"}
	if credit, matched := payout.(ledger.CreditPayout); matched {
		response.PayoutKind = "credit"
		response.PayoutAmount = credit.Amount.Int64()
		response.WorkerUserID = credit.WorkerUserID.String()
	}
	if bundle, matched := payout.(ledger.BundlePayout); matched {
		response.PayoutKind = "bundle"
		response.PayoutAmount = bundle.Amount.Int64()
		response.WorkerUserID = bundle.WorkerUserID.String()
	}
	if creditTip, matched := tip.(ledger.CreditTip); matched {
		response.TipAmount = creditTip.Amount.Int64()
		if response.WorkerUserID == "" {
			response.WorkerUserID = creditTip.WorkerUserID.String()
		}
	}
	return response
}

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

func acceptCreditSelection(amount int64) creditSelectionResult {
	if amount < 0 {
		return creditSelectionRejected{reason: "payout amount cannot be negative"}
	}
	if amount == 0 {
		return creditSelectionAccepted{value: ledger.FullCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return creditSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return creditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}

func rejectCreditSelection(amount int64) creditSelectionResult {
	if amount < 0 {
		return creditSelectionRejected{reason: "partial credit amount cannot be negative"}
	}
	if amount == 0 {
		return creditSelectionAccepted{value: ledger.NoCreditReviewSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return creditSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return creditSelectionAccepted{value: ledger.PartialCreditReviewSelection{Amount: credit.Value}}
}

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

func tipSelectionFromAmount(amount int64) tipSelectionResult {
	if amount < 0 {
		return tipSelectionRejected{reason: "tip amount cannot be negative"}
	}
	if amount == 0 {
		return tipSelectionAccepted{value: ledger.NoTipSelection{}}
	}
	creditResult := ledger.NewCreditAmount(amount)
	credit, matched := creditResult.(ledger.CreditAmountAccepted)
	if !matched {
		return tipSelectionRejected{reason: creditResult.(ledger.CreditAmountRejected).Reason.Description()}
	}
	return tipSelectionAccepted{value: ledger.CreditTipSelection{Amount: credit.Value}}
}

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

func decodeTaskRequest(r *http.Request, actor auth.UserSubject) taskRequestResult {
	var request taskRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return taskRequestRejected{reason: "request body is invalid"}
	}

	ownerResult := parseTaskOwnerRequest(request.Owner)
	ownerAccepted, ownerMatched := ownerResult.(taskOwnerAccepted)
	if !ownerMatched {
		rejected := ownerResult.(taskOwnerRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	titleResult := task.NewTitle(request.Title)
	titleAccepted, titleMatched := titleResult.(task.TitleAccepted)
	if !titleMatched {
		rejected := titleResult.(task.TitleRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	descriptionResult := task.NewDescription(request.Description)
	descriptionAccepted, descriptionMatched := descriptionResult.(task.DescriptionAccepted)
	if !descriptionMatched {
		rejected := descriptionResult.(task.DescriptionRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	rewardResult := parseTaskRewardRequest(request.Reward)
	rewardAccepted, rewardMatched := rewardResult.(taskRewardAccepted)
	if !rewardMatched {
		rejected := rewardResult.(taskRewardRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	participationResult := parseTaskParticipationRequest(request.Participation)
	participationAccepted, participationMatched := participationResult.(taskParticipationAccepted)
	if !participationMatched {
		rejected := participationResult.(taskParticipationRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	visibilityResult := parseTaskVisibilityRequest(request.Visibility, ownerAccepted.value)
	visibilityAccepted, visibilityMatched := visibilityResult.(taskVisibilityAccepted)
	if !visibilityMatched {
		rejected := visibilityResult.(taskVisibilityRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	placementResult := parseTaskPlacementRequest(request.Placement)
	placementAccepted, placementMatched := placementResult.(taskPlacementAccepted)
	if !placementMatched {
		rejected := placementResult.(taskPlacementRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	schemaResult := schema.ParseSchemaJSON([]byte(request.ResponseSchemaJSON))
	if _, schemaMatched := schemaResult.(schema.SchemaParsed); !schemaMatched {
		rejected := schemaResult.(schema.SchemaParseRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	schemaSourceResult := task.NewResponseSchemaSource(request.ResponseSchemaJSON)
	schemaSourceAccepted, schemaSourceMatched := schemaSourceResult.(task.ResponseSchemaSourceAccepted)
	if !schemaSourceMatched {
		rejected := schemaSourceResult.(task.ResponseSchemaSourceRejected)
		return taskRequestRejected{reason: rejected.Reason.Description()}
	}

	payloadResult := parseTaskPayloadRequest(request.Payload)
	payloadAccepted, payloadMatched := payloadResult.(taskPayloadAccepted)
	if !payloadMatched {
		rejected := payloadResult.(taskPayloadRejected)
		return taskRequestRejected{reason: rejected.reason}
	}

	return taskRequestAccepted{command: task.CreateCommand{
		Actor:          actor,
		Owner:          ownerAccepted.value,
		Title:          titleAccepted.Value,
		Description:    descriptionAccepted.Value,
		Reward:         rewardAccepted.value,
		Participation:  participationAccepted.policy,
		AssigneeScope:  participationAccepted.assigneeScope,
		ReservationTTL: participationAccepted.ttl,
		Visibility:     visibilityAccepted.value,
		Placement:      placementAccepted.value,
		ResponseSchema: schemaSourceAccepted.Value,
		Payload:        payloadAccepted.value,
	}}
}

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

func parseTaskParticipationRequest(request taskParticipationRequest) taskParticipationResult {
	rawPolicy := request.Policy
	if rawPolicy == "" {
		rawPolicy = task.ParticipationPolicyOpen.String()
	}
	policyResult := task.ParseParticipationPolicy(rawPolicy)
	policyAccepted, policyMatched := policyResult.(task.ParticipationPolicyAccepted)
	if !policyMatched {
		rejected := policyResult.(task.ParticipationPolicyRejected)
		return taskParticipationRejected{reason: rejected.Reason.Description()}
	}

	rawAssigneeScope := request.AssigneeScope
	if rawAssigneeScope == "" {
		rawAssigneeScope = task.AssigneeScopeUser.String()
	}
	assigneeScopeResult := task.ParseAssigneeScope(rawAssigneeScope)
	assigneeScopeAccepted, assigneeScopeMatched := assigneeScopeResult.(task.AssigneeScopeAccepted)
	if !assigneeScopeMatched {
		rejected := assigneeScopeResult.(task.AssigneeScopeRejected)
		return taskParticipationRejected{reason: rejected.Reason.Description()}
	}

	ttl := task.DefaultReservationTTL()
	if request.ReservationExpiryHours != 0 {
		ttlResult := task.NewReservationTTL(request.ReservationExpiryHours)
		ttlAccepted, ttlMatched := ttlResult.(task.ReservationTTLAccepted)
		if !ttlMatched {
			rejected := ttlResult.(task.ReservationTTLRejected)
			return taskParticipationRejected{reason: rejected.Reason.Description()}
		}
		ttl = ttlAccepted.Value
	}

	return taskParticipationAccepted{policy: policyAccepted.Value, assigneeScope: assigneeScopeAccepted.Value, ttl: ttl}
}

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

func parseTaskRewardRequest(request taskRewardRequest) taskRewardResult {
	switch request.Kind {
	case task.RewardKindNone.String():
		return taskRewardAccepted{value: task.NoRewardSpec{}}
	case task.RewardKindCredit.String():
		amountResult := task.NewCreditRewardAmount(request.CreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			rejected := amountResult.(task.CreditRewardAmountRejected)
			return taskRewardRejected{reason: rejected.Reason.Description()}
		}
		return taskRewardAccepted{value: task.CreditRewardSpec{Amount: amount.Value}}
	case task.RewardKindCollectible.String():
		countResult := task.NewCollectibleRewardCount(1)
		count := countResult.(task.CollectibleRewardCountAccepted)
		return taskRewardAccepted{value: task.CollectibleRewardSpec{Count: count.Value}}
	case task.RewardKindBundle.String():
		amountResult := task.NewCreditRewardAmount(request.CreditAmount)
		amount, matched := amountResult.(task.CreditRewardAmountAccepted)
		if !matched {
			rejected := amountResult.(task.CreditRewardAmountRejected)
			return taskRewardRejected{reason: rejected.Reason.Description()}
		}
		countResult := task.NewCollectibleRewardCount(1)
		count := countResult.(task.CollectibleRewardCountAccepted)
		return taskRewardAccepted{value: task.BundleRewardSpec{Credit: amount.Value, Collectible: count.Value}}
	default:
		return taskRewardRejected{reason: "task reward kind is invalid"}
	}
}

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

func parseTaskOwnerRequest(request taskOwnerRequest) taskOwnerResult {
	switch request.Kind {
	case task.OwnerKindUser.String():
		userIDResult := core.ParseUserID(request.UserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.UserOwner{UserID: userID.Value}}
	case task.OwnerKindTeam.String():
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.TeamOwner{TeamID: teamID.Value}}
	case task.OwnerKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.OrganizationOwner{OrganizationID: organizationID.Value}}
	case task.OwnerKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskOwnerRejected{reason: rejected.Reason.Description()}
		}
		return taskOwnerAccepted{value: task.OrganizationTeamOwner{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskOwnerRejected{reason: "task owner kind is invalid"}
	}
}

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

func parseTaskVisibilityRequest(request taskVisibilityRequest, owner task.Owner) taskVisibilityResult {
	if request.Kind == "default" {
		return defaultVisibilityForOwner(owner)
	}
	switch request.Kind {
	case task.VisibilityKindPublic.String():
		return taskVisibilityAccepted{value: task.PublicVisibility{}}
	case task.VisibilityKindUser.String():
		userIDResult := core.ParseUserID(request.UserID)
		userID, matched := userIDResult.(core.UserIDCreated)
		if !matched {
			rejected := userIDResult.(core.UserIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.UserVisibility{UserID: userID.Value}}
	case task.VisibilityKindTeam.String():
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, matched := teamIDResult.(core.TeamIDCreated)
		if !matched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.TeamVisibility{TeamID: teamID.Value}}
	case task.VisibilityKindOrganization.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.OrganizationVisibility{OrganizationID: organizationID.Value}}
	case task.VisibilityKindOrganizationTeam.String():
		organizationIDResult := core.ParseOrganizationID(request.OrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		teamIDResult := core.ParseTeamID(request.TeamID)
		teamID, teamMatched := teamIDResult.(core.TeamIDCreated)
		if !teamMatched {
			rejected := teamIDResult.(core.TeamIDRejected)
			return taskVisibilityRejected{reason: rejected.Reason.Description()}
		}
		return taskVisibilityAccepted{value: task.OrganizationTeamVisibility{OrganizationID: organizationID.Value, TeamID: teamID.Value}}
	default:
		return taskVisibilityRejected{reason: "task visibility kind is invalid"}
	}
}

func defaultVisibilityForOwner(owner task.Owner) taskVisibilityResult {
	switch typed := owner.(type) {
	case task.UserOwner:
		return taskVisibilityAccepted{value: task.UserVisibility{UserID: typed.UserID}}
	case task.TeamOwner:
		return taskVisibilityAccepted{value: task.TeamVisibility{TeamID: typed.TeamID}}
	case task.OrganizationOwner:
		return taskVisibilityAccepted{value: task.OrganizationVisibility{OrganizationID: typed.OrganizationID}}
	case task.OrganizationTeamOwner:
		return taskVisibilityAccepted{value: task.OrganizationTeamVisibility{OrganizationID: typed.OrganizationID, TeamID: typed.TeamID}}
	default:
		return taskVisibilityRejected{reason: "task owner is invalid"}
	}
}

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

func parseTaskPlacementRequest(request taskPlacementRequest) taskPlacementResult {
	switch request.Kind {
	case "standalone":
		return taskPlacementAccepted{value: task.StandalonePlacement{}}
	case "new_series":
		titleResult := task.NewSeriesTitle(request.SeriesTitle)
		title, titleMatched := titleResult.(task.SeriesTitleAccepted)
		if !titleMatched {
			rejected := titleResult.(task.SeriesTitleRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		positionResult := task.NewSeriesPosition(request.SeriesPosition)
		position, positionMatched := positionResult.(task.SeriesPositionAccepted)
		if !positionMatched {
			rejected := positionResult.(task.SeriesPositionRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		return taskPlacementAccepted{value: task.NewSeriesPlacement{Title: title.Value, Position: position.Value}}
	case "existing_series":
		seriesIDResult := core.ParseTaskSeriesID(request.SeriesID)
		seriesID, seriesMatched := seriesIDResult.(core.TaskSeriesIDCreated)
		if !seriesMatched {
			rejected := seriesIDResult.(core.TaskSeriesIDRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		positionResult := task.NewSeriesPosition(request.SeriesPosition)
		position, positionMatched := positionResult.(task.SeriesPositionAccepted)
		if !positionMatched {
			rejected := positionResult.(task.SeriesPositionRejected)
			return taskPlacementRejected{reason: rejected.Reason.Description()}
		}
		return taskPlacementAccepted{value: task.ExistingSeriesPlacement{SeriesID: seriesID.Value, Position: position.Value}}
	default:
		return taskPlacementRejected{reason: "task series placement kind is invalid"}
	}
}

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

func parseTaskPayloadRequest(request taskPayloadRequest) taskPayloadResult {
	switch request.Kind {
	case "none":
		return taskPayloadAccepted{value: task.NoDataPayload{}}
	case "json":
		if !json.Valid([]byte(request.JSON)) {
			return taskPayloadRejected{reason: "task payload JSON is invalid"}
		}
		sourceResult := task.NewPayloadSource(request.JSON)
		source, matched := sourceResult.(task.PayloadSourceAccepted)
		if !matched {
			rejected := sourceResult.(task.PayloadSourceRejected)
			return taskPayloadRejected{reason: rejected.Reason.Description()}
		}
		return taskPayloadAccepted{value: task.JSONDataPayload{Source: source.Value}}
	default:
		return taskPayloadRejected{reason: "task payload kind is invalid"}
	}
}

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

func parseTaskPathValue(r *http.Request) taskIDResult {
	result := core.ParseTaskID(r.PathValue("task_id"))
	accepted, matched := result.(core.TaskIDCreated)
	if !matched {
		rejected := result.(core.TaskIDRejected)
		return taskIDRejected{reason: rejected.Reason.Description()}
	}
	return taskIDAccepted{value: accepted.Value}
}

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

func parseReservationPathValue(r *http.Request) reservationIDResult {
	result := core.ParseTaskReservationID(r.PathValue("reservation_id"))
	accepted, matched := result.(core.TaskReservationIDCreated)
	if !matched {
		rejected := result.(core.TaskReservationIDRejected)
		return reservationIDRejected{reason: rejected.Reason.Description()}
	}
	return reservationIDAccepted{value: accepted.Value}
}

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

func parseTaskListScope(r *http.Request, actor auth.UserSubject) taskListScopeResult {
	scope := r.URL.Query().Get("scope")
	includeReserved := r.URL.Query().Get("include_reserved") == "true"
	switch scope {
	case "public":
		return taskListScopeAccepted{value: task.PublicListScope{ViewerID: actor.ID, IncludeReserved: includeReserved}}
	case "user":
		return taskListScopeAccepted{value: task.UserListScope{UserID: actor.ID, IncludeReserved: includeReserved}}
	case "organization":
		organizationIDResult := core.ParseOrganizationID(r.URL.Query().Get("organization_id"))
		organizationID, matched := organizationIDResult.(core.OrganizationIDCreated)
		if !matched {
			rejected := organizationIDResult.(core.OrganizationIDRejected)
			return taskListScopeRejected{reason: rejected.Reason.Description()}
		}
		return taskListScopeAccepted{value: task.OrganizationListScope{OrganizationID: organizationID.Value, UserID: actor.ID, IncludeReserved: includeReserved}}
	default:
		return taskListScopeRejected{reason: "task list scope is invalid"}
	}
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

func decodeAuthenticatedSubmissionRequest(r *http.Request, actor auth.UserSubject, taskID core.TaskID) submissionRequestResult {
	var request submissionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return submissionRequestRejected{reason: "request body is invalid"}
	}

	sourceResult := submission.NewResponseSource(request.ResponseJSON)
	source, sourceMatched := sourceResult.(submission.ResponseSourceAccepted)
	if !sourceMatched {
		rejected := sourceResult.(submission.ResponseSourceRejected)
		return submissionRequestRejected{reason: rejected.Reason.Description()}
	}

	return submissionRequestAccepted{command: submission.SubmitCommand{
		TaskID:         taskID,
		SubmitterID:    actor.ID,
		ResponseSource: source.Value,
	}}
}

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

func setRefreshCookie(w http.ResponseWriter, refreshToken auth.RefreshTokenPlain) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sharecrop_refresh_token",
		Value:    refreshToken.String(),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().UTC().Add(30 * 24 * time.Hour),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "sharecrop_refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
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
	return teamResponse{ID: value.ID.String(), OrganizationID: value.OrganizationID.String(), Name: value.Name.String(), CreatedBy: value.CreatedBy.String()}
}

func taskToResponse(value task.Task) taskResponse {
	owner := taskOwnerResponseParts(value.Owner)
	visibility := taskVisibilityResponseParts(value.Visibility)
	placement := taskPlacementResponseParts(value.Placement)
	payload := taskPayloadResponseParts(value.Payload)
	reward := taskRewardResponseParts(value.Reward)
	return taskResponse{
		ID:                     value.ID.String(),
		OwnerKind:              owner.kind,
		OwnerID:                owner.id,
		Title:                  value.Title.String(),
		Description:            value.Description.String(),
		RewardKind:             reward.kind,
		RewardCreditAmount:     reward.amount,
		RewardCollectibleCount: reward.collectibleCount,
		ParticipationPolicy:    value.Participation.String(),
		AssigneeScope:          value.AssigneeScope.String(),
		ReservationExpiryHours: value.ReservationTTL.Hours(),
		State:                  value.State.String(),
		VisibilityKind:         visibility.kind,
		VisibilityID:           visibility.id,
		SeriesKind:             placement.kind,
		SeriesID:               placement.id,
		SeriesPosition:         placement.position,
		ResponseSchemaJSON:     value.ResponseSchema.String(),
		PayloadKind:            payload.kind,
		PayloadJSON:            payload.source,
		CreatedBy:              value.CreatedBy.String(),
		AvailabilityKind:       taskAvailabilityKind(value).String(),
		ViewerAction:           taskViewerAction(value).String(),
	}
}

func taskAvailabilityKind(value task.Task) task.AvailabilityKind {
	if value.State != task.StateOpen {
		return task.AvailabilityClosed
	}
	if value.Participation == task.ParticipationPolicyApprovalRequired {
		return task.AvailabilityAwaitingApproval
	}
	return task.AvailabilityAvailable
}

func taskViewerAction(value task.Task) task.ViewerAction {
	if value.State != task.StateOpen {
		return task.ViewerActionNone
	}
	switch value.Participation {
	case task.ParticipationPolicyOpen:
		return task.ViewerActionSubmit
	case task.ParticipationPolicyReservationRequired:
		return task.ViewerActionReserve
	case task.ParticipationPolicyApprovalRequired:
		return task.ViewerActionRequestApproval
	default:
		return task.ViewerActionNone
	}
}

func reservationToResponse(value task.Reservation) reservationResponse {
	assignee := reservationAssigneeResponseParts(value.Assignee)
	return reservationResponse{
		ID:           value.ID.String(),
		TaskID:       value.TaskID.String(),
		AssigneeKind: assignee.kind,
		AssigneeID:   assignee.id,
		State:        value.State.String(),
		RequestedBy:  value.RequestedBy.String(),
	}
}

func reservationAssigneeResponseParts(assignee task.Assignee) responseParts {
	switch typed := assignee.(type) {
	case task.UserAssignee:
		return responseParts{kind: task.AssigneeScopeUser.String(), id: typed.UserID.String()}
	case task.OrganizationTeamAssignee:
		return responseParts{kind: task.AssigneeScopeOrganizationTeam.String(), id: typed.TeamID.String()}
	default:
		return responseParts{}
	}
}

type rewardResponseParts struct {
	kind             string
	amount           int64
	collectibleCount int
}

func taskRewardResponseParts(reward task.RewardSpec) rewardResponseParts {
	switch typed := reward.(type) {
	case task.NoRewardSpec:
		return rewardResponseParts{kind: task.RewardKindNone.String()}
	case task.CreditRewardSpec:
		return rewardResponseParts{kind: task.RewardKindCredit.String(), amount: typed.Amount.Int64()}
	case task.CollectibleRewardSpec:
		return rewardResponseParts{kind: task.RewardKindCollectible.String(), collectibleCount: typed.Count.Int()}
	case task.BundleRewardSpec:
		return rewardResponseParts{kind: task.RewardKindBundle.String(), amount: typed.Credit.Int64(), collectibleCount: typed.Collectible.Int()}
	default:
		return rewardResponseParts{}
	}
}

type responseParts struct {
	kind     string
	id       string
	position int
	source   string
}

func taskOwnerResponseParts(owner task.Owner) responseParts {
	switch typed := owner.(type) {
	case task.UserOwner:
		return responseParts{kind: task.OwnerKindUser.String(), id: typed.UserID.String()}
	case task.TeamOwner:
		return responseParts{kind: task.OwnerKindTeam.String(), id: typed.TeamID.String()}
	case task.OrganizationOwner:
		return responseParts{kind: task.OwnerKindOrganization.String(), id: typed.OrganizationID.String()}
	case task.OrganizationTeamOwner:
		return responseParts{kind: task.OwnerKindOrganizationTeam.String(), id: typed.OrganizationID.String() + ":" + typed.TeamID.String()}
	default:
		return responseParts{}
	}
}

func taskVisibilityResponseParts(visibility task.Visibility) responseParts {
	switch typed := visibility.(type) {
	case task.PublicVisibility:
		return responseParts{kind: task.VisibilityKindPublic.String()}
	case task.UserVisibility:
		return responseParts{kind: task.VisibilityKindUser.String(), id: typed.UserID.String()}
	case task.TeamVisibility:
		return responseParts{kind: task.VisibilityKindTeam.String(), id: typed.TeamID.String()}
	case task.OrganizationVisibility:
		return responseParts{kind: task.VisibilityKindOrganization.String(), id: typed.OrganizationID.String()}
	case task.OrganizationTeamVisibility:
		return responseParts{kind: task.VisibilityKindOrganizationTeam.String(), id: typed.OrganizationID.String() + ":" + typed.TeamID.String()}
	default:
		return responseParts{}
	}
}

func taskPlacementResponseParts(placement task.SeriesPlacement) responseParts {
	switch typed := placement.(type) {
	case task.StandalonePlacement:
		return responseParts{kind: "standalone"}
	case task.NewSeriesPlacement:
		return responseParts{kind: "new_series", position: typed.Position.Int()}
	case task.ExistingSeriesPlacement:
		return responseParts{kind: "existing_series", id: typed.SeriesID.String(), position: typed.Position.Int()}
	default:
		return responseParts{}
	}
}

func taskPayloadResponseParts(payload task.DataPayload) responseParts {
	switch typed := payload.(type) {
	case task.NoDataPayload:
		return responseParts{kind: "none"}
	case task.JSONDataPayload:
		return responseParts{kind: "json", source: typed.Source.String()}
	default:
		return responseParts{}
	}
}

func submissionToResponse(value submission.Submission) submissionResponse {
	errors := submissionValidationErrorsToResponse(value.Validation)
	return submissionResponse{
		ID:               value.ID.String(),
		TaskID:           value.TaskID.String(),
		SubmitterID:      value.SubmitterID.String(),
		State:            value.State.String(),
		ResponseJSON:     value.ResponseSource.String(),
		ReviewNote:       value.ReviewNote.String(),
		ValidationErrors: errors,
	}
}

func submissionValidationErrorsToResponse(outcome submission.ValidationOutcome) []submissionValidationErrorResponse {
	failed, matched := outcome.(submission.ValidationFailed)
	if !matched {
		return []submissionValidationErrorResponse{}
	}
	errors := make([]submissionValidationErrorResponse, 0, len(failed.Errors))
	for errorIndex := range failed.Errors {
		validationError := failed.Errors[errorIndex]
		errors = append(errors, submissionValidationErrorResponse{Path: validationError.Path, Message: validationError.Message})
	}
	return errors
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
