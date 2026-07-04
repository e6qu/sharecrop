package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
)

// CallerCredential is the scope/task-restriction facts of whichever
// credential authenticated an MCP call, independent of its concrete kind
// (a personal agent.Credential or an organization-wide orgcred.Credential —
// only the former can ever be task-scoped).
type CallerCredential struct {
	Scopes agent.ScopeSet
	TaskID *core.TaskID
}

// Services is the set of domain operations the MCP adapter exposes as tools.
// Methods here take auth.Subject when their REST counterpart already
// accepts an organization-wide credential with full parity (see
// requireUserOrOrgSubject in internal/http); every other method stays
// auth.UserSubject-only, matching REST exactly rather than exceeding it.
type Services interface {
	ListTasks(context.Context, auth.Subject, task.ListScope, task.ListFilters) task.ListResult
	GetTask(context.Context, auth.UserSubject, core.TaskID) task.GetResult
	CreateTask(context.Context, task.CreateCommand) task.CreateResult
	OpenTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	CancelTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	RefundTask(context.Context, core.UserID, core.TaskID, ledger.IdempotencyKey) ledger.RefundResult
	SubmitResponse(context.Context, submission.SubmitCommand) submission.SubmitResult
	GetSubmissionStatus(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	ListTaskSubmissions(context.Context, auth.UserSubject, core.TaskID) submission.ListResult
	AcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey) ledger.AcceptResult
	ReviewAcceptSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, ledger.CreditReviewSelection, ledger.TipSelection, ledger.CollectibleTipSelection) ledger.AcceptResult
	RequestChanges(context.Context, core.UserID, core.TaskID, core.SubmissionID, submission.ReviewNote) ledger.RequestChangesResult
	RejectSubmission(context.Context, core.UserID, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote, ledger.CreditReviewSelection, ledger.TipSelection, ledger.BanSelection) ledger.RejectResult
	ListSeries(context.Context, auth.UserSubject) task.ListSeriesResult
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
	AddSubmissionComment(context.Context, auth.UserSubject, core.SubmissionID, task.CommentBody) submission.SubmissionCommentResult
	ListSubmissionComments(context.Context, auth.UserSubject, core.SubmissionID) submission.SubmissionCommentsResult
	UnpublishTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	ReserveTask(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ReserveTaskForOrganizationTeam(context.Context, auth.UserSubject, core.TaskID, core.OrganizationID, core.TeamID) task.ReservationResult
	ListReservations(context.Context, auth.Subject, core.TaskID) task.ReservationsListResult
	ApproveReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	DeclineReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult
	CancelReservation(context.Context, auth.Subject, core.TaskID, core.TaskReservationID) task.ReservationStateChangeResult

	CreateOrganization(context.Context, auth.UserSubject, org.OrganizationName) org.CreateOrganizationResult
	ListOrganizations(context.Context, auth.UserSubject, string, core.Page) org.ListOrganizationsResult
	ListOrganizationMembers(context.Context, auth.UserSubject, core.OrganizationID, core.Page) org.ListMembersResult
	ProvisionOrganizationMember(context.Context, auth.UserSubject, core.OrganizationID, auth.EmailAddress, []org.Role) org.ProvisionMemberResult
	DeactivateOrganizationMember(context.Context, auth.UserSubject, core.OrganizationID, core.UserID) org.DeactivateMemberResult
	UpdateOrganizationMemberRoles(context.Context, auth.UserSubject, core.OrganizationID, core.UserID, []org.Role) org.UpdateMemberRolesResult
	CreateOrganizationTeam(context.Context, auth.UserSubject, core.OrganizationID, org.TeamName) org.CreateTeamResult
	ListOrganizationTeams(context.Context, auth.UserSubject, core.OrganizationID, string, core.Page) org.ListTeamsResult
	CreateStandaloneTeam(context.Context, auth.UserSubject, org.TeamName) org.CreateTeamResult
	ListStandaloneTeams(context.Context, auth.UserSubject, string, core.Page) org.ListTeamsResult
	GetTeam(context.Context, auth.Subject, core.TeamID) org.GetTeamResult
	GetTeamWork(context.Context, auth.UserSubject, core.TeamID, task.ListFilters, core.Page) task.ListResult
	AddTeamMember(context.Context, auth.Subject, core.TeamID, auth.EmailAddress) org.AddTeamMemberResult

	CheckOrganizationPermission(context.Context, core.OrganizationID, core.UserID, org.Permission) org.PermissionCheck
	CreateOrgCredential(context.Context, core.OrganizationID, agent.Label, agent.ScopeSet, *time.Time) orgcred.CreateResult
	ListOrgCredentials(context.Context, core.OrganizationID, core.Page) orgcred.ListResult
	RevokeOrgCredential(context.Context, core.OrganizationID, core.OrgCredentialID) orgcred.RevokeResult

	MintCollectible(context.Context, string, string, string, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy, string) assets.MintResult
	ListCollectibles(context.Context, core.UserID, core.Page) assets.ListResult
	ListCollectiblesByOwner(context.Context, string, string, core.Page) assets.ListResult
	TransferCollectible(context.Context, core.UserID, core.UserID, core.CollectibleID) assets.GiftResult
	FundCollectibleReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult
	RefundCollectibleReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult

	ListNotifications(context.Context, core.UserID, core.Page) notification.ListResult
	MarkNotificationRead(context.Context, core.UserID, core.NotificationID) notification.MarkReadResult

	ListUsers(context.Context, string, core.Page) auth.UserDirectoryResult
	GetUserProfile(context.Context, auth.UserSubject, core.UserID, core.Page) task.ListResult
	GetUserWork(context.Context, auth.UserSubject, core.UserID, core.Page) task.ListResult
	GetUserSubmissions(context.Context, auth.UserSubject, core.UserID, core.Page) submission.ListResult

	IsPlatformAdmin(context.Context, core.UserID) bool
	ListPlatformAdmins(context.Context, core.Page) PlatformAdminListResult
	GrantPlatformAdmin(context.Context, core.UserID, core.UserID) PlatformAdminMutationResult
	RevokePlatformAdmin(context.Context, core.UserID) PlatformAdminMutationResult

	CreateModerationReport(context.Context, core.UserID, string, string, string, string) ModerationReportResult
	ListAdminModerationReports(context.Context, string, core.Page) ModerationReportsListResult
	TriageModerationReport(context.Context, core.UserID, core.AuditEventID, string, string) ModerationReportResult

	CreatePrivacyRequest(context.Context, core.UserID, string) PrivacyRequestResult
	ListPrivacyRequests(context.Context, core.UserID, core.Page) PrivacyRequestsListResult
	ListAdminPrivacyRequests(context.Context, core.Page) PrivacyRequestsListResult
	ResolveAdminPrivacyRequest(context.Context, string, string) PrivacyRequestResult
	RunPrivacyRetention(context.Context, core.UserID) PrivacyRetentionResult

	ListAuditEvents(context.Context, audit.ListFilters, core.Page) audit.ListResult

	AwardCollectible(context.Context, string, string, string, string) assets.MintResult
}

type Server struct {
	services Services
}

func NewServer(services Services) Server {
	return Server{services: services}
}

// Handle dispatches a single JSON-RPC request for an authenticated agent or
// organization-wide credential.
func (server Server) Handle(ctx context.Context, subject auth.Subject, credential CallerCredential, request Request) Response {
	if request.JSONRPC != jsonRPCVersion {
		return errorResponse(request.ID, codeInvalidRequest, "jsonrpc version must be 2.0")
	}

	switch request.Method {
	case "initialize":
		return server.handleInitialize(request)
	case "ping":
		return successResponse(request.ID, json.RawMessage(`{}`))
	case "tools/list":
		return server.handleToolsList(request)
	case "tools/call":
		return server.handleToolsCall(ctx, subject, credential, request)
	default:
		return errorResponse(request.ID, codeMethodNotFound, "unknown method: "+request.Method)
	}
}

func (server Server) handleInitialize(request Request) Response {
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities{Tools: toolsCapability{}},
		ServerInfo:      serverInfo{Name: serverName, Version: serverVersion},
	}
	return marshalResult(request.ID, result)
}

func (server Server) handleToolsList(request Request) Response {
	definitions := toolDefinitions()
	entries := make([]toolListEntry, 0, len(definitions))
	for index := range definitions {
		entries = append(entries, toolListEntry{
			Name:        definitions[index].Name,
			Description: definitions[index].Description,
			InputSchema: definitions[index].InputSchema,
		})
	}
	return marshalResult(request.ID, toolListResult{Tools: entries})
}

func (server Server) handleToolsCall(ctx context.Context, subject auth.Subject, credential CallerCredential, request Request) Response {
	var params toolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return errorResponse(request.ID, codeInvalidParams, "tools/call params are invalid")
	}

	definition, found := findTool(params.Name)
	if !found {
		return errorResponse(request.ID, codeInvalidParams, "unknown tool: "+params.Name)
	}
	if _, granted := credential.Scopes.Allows(definition.Scope).(agent.ScopeGranted); !granted {
		return errorResponse(request.ID, codeScopeDenied, "agent credential is missing the "+definition.Scope.String()+" scope")
	}
	// A task-scoped credential (e.g. auto-issued when a reservation becomes
	// active) may only call tools whose arguments target that exact task.
	// Tools with no task_id argument aren't restricted here: list_tasks is
	// equivalent to public task browsing (not a leak); submission-comment
	// tools take a submission_id instead (no task_id to check against) — a
	// known, narrower gap where the credential could reach a submission on
	// a *different* task, but only one the same underlying user is already
	// legitimately the submitter or task owner/reviewer for, since the
	// service-layer authorization checks below still apply regardless.
	if credential.TaskID != nil {
		if argTaskID, present := toolArgumentTaskID(params.Arguments); present {
			// Parse before comparing rather than a raw string match, so a
			// non-canonically-cased (but otherwise valid) task ID isn't
			// spuriously rejected as "not valid for this task".
			parsedResult := core.ParseTaskID(argTaskID)
			parsed, parsedMatched := parsedResult.(core.TaskIDCreated)
			if !parsedMatched || parsed.Value.String() != credential.TaskID.String() {
				return errorResponse(request.ID, codeScopeDenied, "agent credential is not valid for this task")
			}
		}
	}

	outcome := server.dispatchTool(ctx, subject, definition.Name, params.Arguments)
	switch typed := outcome.(type) {
	case toolSucceeded:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: string(typed.payload)}}})
	case toolFailed:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: typed.message}}, IsError: true})
	case toolProtocolError:
		return errorResponse(request.ID, typed.code, typed.message)
	default:
		return errorResponse(request.ID, codeInternalError, "tool produced no result")
	}
}

// toolArgumentTaskID extracts a tool call's "task_id" argument, if it has
// one. present is false for tools with no such argument (e.g. list_tasks).
func toolArgumentTaskID(arguments json.RawMessage) (taskID string, present bool) {
	var args struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil || args.TaskID == "" {
		return "", false
	}
	return args.TaskID, true
}

// requireUserSubjectForTool guards tools whose underlying domain method is
// auth.UserSubject-only (matching their REST counterpart, which likewise has
// no organization-credential fallback): an org-wide credential is rejected
// with a clear message rather than a type assertion panic.
func requireUserSubjectForTool(subject auth.Subject) (auth.UserSubject, toolResult, bool) {
	userActor, isUser := subject.(auth.UserSubject)
	if !isUser {
		return auth.UserSubject{}, toolFailed{message: "this tool requires a personal agent credential, not an organization credential"}, false
	}
	return userActor, nil, true
}

// requireAdminSubjectForTool gates an admin-only tool exactly like REST's
// requireAdminSubject: beyond the scope check handleToolsCall already did,
// it re-checks that the underlying user is actually a platform admin right
// now. This matters because a credential's scopes are fixed at mint time —
// without this check, a platform_admin-scoped credential minted by an admin
// who is later demoted would still pass the scope gate alone.
func (server Server) requireAdminSubjectForTool(ctx context.Context, subject auth.Subject) (auth.UserSubject, toolResult, bool) {
	userActor, failure, ok := requireUserSubjectForTool(subject)
	if !ok {
		return userActor, failure, false
	}
	if !server.services.IsPlatformAdmin(ctx, userActor.ID) {
		return userActor, toolFailed{message: "platform admin access is required"}, false
	}
	return userActor, nil, true
}

func (server Server) dispatchTool(ctx context.Context, subject auth.Subject, name string, arguments json.RawMessage) toolResult {
	switch name {
	case toolListTasks:
		return server.callListTasks(ctx, subject, arguments)
	case toolGetTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetTask(ctx, userActor, arguments)
	case toolGetTaskSchema:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetTaskSchema(ctx, userActor, arguments)
	case toolCreateTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateTask(ctx, userActor, arguments)
	case toolOpenTask:
		return server.callOpenTask(ctx, subject, arguments)
	case toolCancelTask:
		return server.callCancelTask(ctx, subject, arguments)
	case toolFundTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callFundTask(ctx, userActor, arguments)
	case toolRefundTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRefundTask(ctx, userActor, arguments)
	case toolSubmitResponse:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callSubmitResponse(ctx, userActor, arguments)
	case toolGetSubmissionStatus:
		return server.callGetSubmissionStatus(ctx, arguments)
	case toolListTaskSubmissions:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListTaskSubmissions(ctx, userActor, arguments)
	case toolAcceptSubmission:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callAcceptSubmission(ctx, userActor, arguments)
	case toolRequestChanges:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRequestChanges(ctx, userActor, arguments)
	case toolRejectSubmission:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRejectSubmission(ctx, userActor, arguments)
	case toolListTaskSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListTaskSeries(ctx, userActor)
	case toolGetTaskSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetTaskSeries(ctx, userActor, arguments)
	case toolCreateSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateSeries(ctx, userActor, arguments)
	case toolUpdateSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callUpdateSeries(ctx, userActor, arguments)
	case toolAddTaskToSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callAddTaskToSeries(ctx, userActor, arguments)
	case toolRemoveSeriesTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRemoveTaskFromSeries(ctx, userActor, arguments)
	case toolReorderSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callReorderSeries(ctx, userActor, arguments)
	case toolPublishSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callChangeSeriesState(ctx, userActor, arguments, task.PublishSeriesState)
	case toolUnpublishSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callChangeSeriesState(ctx, userActor, arguments, task.UnpublishSeriesState)
	case toolCloseSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callChangeSeriesState(ctx, userActor, arguments, task.CloseSeriesState)
	case toolReopenSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callChangeSeriesState(ctx, userActor, arguments, task.ReopenSeriesState)
	case toolAddSeriesComment:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callAddSeriesComment(ctx, userActor, arguments)
	case toolListSeriesComments:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListSeriesComments(ctx, userActor, arguments)
	case toolAddTaskComment:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callAddTaskComment(ctx, userActor, arguments)
	case toolListTaskComments:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListTaskComments(ctx, userActor, arguments)
	case toolAddSubmissionComment:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callAddSubmissionComment(ctx, userActor, arguments)
	case toolListSubmissionComments:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListSubmissionComments(ctx, userActor, arguments)
	case toolUnpublishTask:
		return server.callUnpublishTask(ctx, subject, arguments)
	case toolReserveTask:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callReserveTask(ctx, userActor, arguments)
	case toolListReservations:
		return server.callListReservations(ctx, subject, arguments)
	case toolApproveReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.ApproveReservation)
	case toolDeclineReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.DeclineReservation)
	case toolCancelReservation:
		return server.callChangeReservation(ctx, subject, arguments, server.services.CancelReservation)
	case toolCreateOrganization:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateOrganization(ctx, userActor, arguments)
	case toolListOrganizations:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrganizations(ctx, userActor, arguments)
	case toolListOrgMembers:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrgMembers(ctx, userActor, arguments)
	case toolProvisionOrgMember:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callProvisionOrgMember(ctx, userActor, arguments)
	case toolDeactivateOrgMember:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callDeactivateOrgMember(ctx, userActor, arguments)
	case toolUpdateOrgMemberRoles:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callUpdateOrgMemberRoles(ctx, userActor, arguments)
	case toolCreateOrganizationTeam:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateOrganizationTeam(ctx, userActor, arguments)
	case toolListOrganizationTeams:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrganizationTeams(ctx, userActor, arguments)
	case toolCreateStandaloneTeam:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateStandaloneTeam(ctx, userActor, arguments)
	case toolListStandaloneTeams:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListStandaloneTeams(ctx, userActor, arguments)
	case toolGetTeam:
		return server.callGetTeam(ctx, subject, arguments)
	case toolGetTeamWork:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetTeamWork(ctx, userActor, arguments)
	case toolAddTeamMember:
		return server.callAddTeamMember(ctx, subject, arguments)
	case toolCreateOrgCredential:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateOrgCredential(ctx, userActor, arguments)
	case toolListOrgCredentials:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrgCredentials(ctx, userActor, arguments)
	case toolRevokeOrgCredential:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRevokeOrgCredential(ctx, userActor, arguments)
	case toolMintCollectible:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callMintCollectible(ctx, userActor, arguments)
	case toolCollectibleCatalog:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCollectibleCatalog(ctx, userActor, arguments)
	case toolTransferCollectible:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callTransferCollectible(ctx, userActor, arguments)
	case toolListCollectibles:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListCollectibles(ctx, userActor, arguments)
	case toolFundCollectibleReward:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callFundCollectibleReward(ctx, userActor, arguments)
	case toolRefundCollectibleReward:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callRefundCollectibleReward(ctx, userActor, arguments)
	case toolListOrganizationCollectibles:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrganizationCollectibles(ctx, userActor, arguments)
	case toolListTeamCollectibles:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListTeamCollectibles(ctx, userActor, arguments)
	case toolListNotifications:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListNotifications(ctx, userActor, arguments)
	case toolMarkNotificationRead:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callMarkNotificationRead(ctx, userActor, arguments)
	case toolListUsers:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListUsers(ctx, userActor, arguments)
	case toolGetUserProfile:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetUserProfile(ctx, userActor, arguments)
	case toolGetUserWork:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetUserWork(ctx, userActor, arguments)
	case toolGetUserSubmissions:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetUserSubmissions(ctx, userActor, arguments)
	case toolListPlatformAdmins:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callListPlatformAdmins(ctx, userActor, arguments)
	case toolGrantPlatformAdmin:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callGrantPlatformAdmin(ctx, userActor, arguments)
	case toolRevokePlatformAdmin:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callRevokePlatformAdmin(ctx, userActor, arguments)
	case toolCreateModerationReport:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreateModerationReport(ctx, userActor, arguments)
	case toolListAdminModerationReports:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callListAdminModerationReports(ctx, userActor, arguments)
	case toolTriageModerationReport:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callTriageModerationReport(ctx, userActor, arguments)
	case toolCreatePrivacyRequest:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callCreatePrivacyRequest(ctx, userActor, arguments)
	case toolListPrivacyRequests:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListPrivacyRequests(ctx, userActor, arguments)
	case toolListAdminPrivacyRequests:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callListAdminPrivacyRequests(ctx, userActor, arguments)
	case toolResolveAdminPrivacyRequest:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callResolveAdminPrivacyRequest(ctx, userActor, arguments)
	case toolRunPrivacyRetention:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callRunPrivacyRetention(ctx, userActor, arguments)
	case toolListOrganizationAuditEvents:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListOrganizationAuditEvents(ctx, userActor, arguments)
	case toolListAdminAuditEvents:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callListAdminAuditEvents(ctx, userActor, arguments)
	case toolAwardCollectible:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callAwardCollectible(ctx, userActor, arguments)
	default:
		return toolProtocolError{code: codeInvalidParams, message: "unknown tool: " + name}
	}
}

func findTool(name string) (toolDefinition, bool) {
	for _, definition := range toolDefinitions() {
		if definition.Name == name {
			return definition, true
		}
	}
	return toolDefinition{}, false
}

func marshalResult(id json.RawMessage, value resultValue) Response {
	encoded, err := json.Marshal(value)
	if err != nil {
		return errorResponse(id, codeInternalError, "failed to encode result")
	}
	return successResponse(id, encoded)
}

type resultValue interface {
	resultValue()
}

func (initializeResult) resultValue() {}

func (toolListResult) resultValue() {}

func (toolCallResult) resultValue() {}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type initializeResult struct {
	ProtocolVersion string       `json:"protocolVersion"`
	Capabilities    capabilities `json:"capabilities"`
	ServerInfo      serverInfo   `json:"serverInfo"`
}

type capabilities struct {
	Tools toolsCapability `json:"tools"`
}

type toolsCapability struct{}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type contentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type toolCallResult struct {
	Content []contentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type toolResult interface {
	toolResult()
}

type toolSucceeded struct {
	payload json.RawMessage
}

type toolFailed struct {
	message string
}

type toolProtocolError struct {
	code    int
	message string
}

func (toolSucceeded) toolResult() {}

func (toolFailed) toolResult() {}

func (toolProtocolError) toolResult() {}
