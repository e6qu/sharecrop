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
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
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
	ListTasks(context.Context, auth.Subject, task.ListScope, task.ListFilters, core.Page) task.ListResult
	GetTask(context.Context, auth.UserSubject, core.TaskID) task.GetResult
	CreateTask(context.Context, task.CreateCommand) task.CreateResult
	OpenTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	CancelTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	FundTask(context.Context, core.UserID, core.TaskID, ledger.CreditAmount, ledger.IdempotencyKey) ledger.FundResult
	RefundTask(context.Context, core.UserID, core.TaskID, ledger.IdempotencyKey) ledger.RefundResult
	SubmitResponse(context.Context, submission.SubmitCommand) submission.SubmitResult
	GetSubmissionStatus(context.Context, submission.ReceiptTokenPlain) submission.ReceiptStatusResult
	GetSubmission(context.Context, auth.Subject, core.SubmissionID) submission.GetResult
	ListTaskSubmissions(context.Context, auth.Subject, core.TaskID, core.Page) submission.ListResult
	// The review methods take the ledger.Reviewer union (a user reviewer or
	// an organization reviewer), mirroring REST: an organization credential
	// reviews its own organization's tasks with org-admin parity, but tips
	// and bans stay user-only (enforced by the ledger service).
	ReviewAcceptSubmission(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, ledger.CreditReviewSelection, ledger.TipSelection, ledger.CollectibleTipSelection) ledger.AcceptResult
	RequestChanges(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote) ledger.RequestChangesResult
	RejectSubmission(context.Context, ledger.Reviewer, core.TaskID, core.SubmissionID, ledger.IdempotencyKey, submission.ReviewNote, ledger.CreditReviewSelection, ledger.TipSelection, ledger.BanSelection) ledger.RejectResult
	ListSeries(context.Context, auth.UserSubject, core.Page) task.ListSeriesResult
	GetSeries(context.Context, auth.UserSubject, core.TaskSeriesID) task.GetSeriesResult
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
	AddSubmissionComment(context.Context, auth.UserSubject, core.SubmissionID, task.CommentBody) submission.SubmissionCommentResult
	ListSubmissionComments(context.Context, auth.UserSubject, core.SubmissionID, core.Page) submission.SubmissionCommentsResult
	UnpublishTask(context.Context, auth.Subject, core.TaskID) task.ChangeStateResult
	ReserveTask(context.Context, auth.UserSubject, core.TaskID) task.ReservationResult
	ReserveTaskForOrganizationTeam(context.Context, auth.UserSubject, core.TaskID, core.OrganizationID, core.TeamID) task.ReservationResult
	ListReservations(context.Context, auth.Subject, core.TaskID, core.Page) task.ReservationsListResult
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

	MintCollectible(context.Context, core.UserID, string, string, string, assets.CollectibleName, assets.CollectibleKind, assets.TransferPolicy, string) assets.MintResult
	ListCollectibleCatalog(context.Context) assets.CatalogListResult
	ListCollectibles(context.Context, core.UserID, core.Page) assets.ListResult
	ListCollectiblesByOwner(context.Context, string, string, core.Page) assets.ListResult
	TransferCollectible(context.Context, core.UserID, core.UserID, core.CollectibleID) assets.GiftResult
	TransferCollectibleToOrganization(context.Context, core.UserID, core.OrganizationID, core.CollectibleID) assets.GiftResult
	TransferCollectibleFromOrganization(context.Context, core.UserID, core.OrganizationID, core.UserID, core.CollectibleID) assets.GiftResult
	AwardOrganizationCollectible(context.Context, core.OrganizationID, core.CollectibleID, core.UserID) assets.GiftResult
	FundCollectibleReward(context.Context, core.UserID, core.TaskID, core.CollectibleID) assets.FundRewardResult
	RefundCollectibleReward(context.Context, core.UserID, core.TaskID) assets.RefundRewardResult
	AddCatalogEntry(context.Context, assets.CatalogEntry) assets.CatalogMutationResult
	WithdrawCatalogEntry(context.Context, assets.CatalogSlug) assets.CatalogMutationResult
	DeleteCatalogEntry(context.Context, assets.CatalogSlug) assets.CatalogMutationResult
	WithdrawCollectible(context.Context, core.UserID, core.CollectibleID) assets.WithdrawResult
	DeleteWithdrawnCollectible(context.Context, core.CollectibleID) assets.DeleteCollectibleResult

	ListNotifications(context.Context, core.UserID, notification.StateFilter, core.Page) notification.ListResult
	CountUnreadNotifications(context.Context, core.UserID) notification.CountResult
	MarkNotificationRead(context.Context, core.UserID, core.NotificationID) notification.MarkReadResult

	// ListEvents serves the caller's cursor event feed through the same
	// store path the REST feed uses: a user subject reads its own
	// recipient-scoped feed, an org subject reads the organization's
	// subject events.
	ListEvents(context.Context, auth.Subject, event.CursorFilter, core.Page) event.ListStoreResult

	GetCreditBalance(context.Context, core.UserID) ledger.BalanceResult
	ListLedger(context.Context, core.UserID, core.Page) ledger.ListEntriesResult
	GrantCredits(context.Context, core.UserID, ledger.GrantTarget, ledger.CreditAmount, ledger.GrantNote, ledger.IdempotencyKey) ledger.GrantResult
	SendCredits(context.Context, core.UserID, ledger.TransferSource, ledger.TransferTarget, ledger.CreditAmount, ledger.TransferNote, ledger.IdempotencyKey) ledger.SendResult

	CreateWebhookSubscription(context.Context, webhook.Owner, webhook.EndpointURL, webhook.KindFilter, webhook.Audience) webhook.CreateResult
	ListWebhookSubscriptions(context.Context, webhook.Owner, core.Page) webhook.ListResult
	RevokeWebhookSubscription(context.Context, webhook.Owner, core.WebhookSubscriptionID) webhook.RevokeResult
	ListWebhookDeliveries(context.Context, webhook.Owner, core.WebhookSubscriptionID, core.Page) webhook.ListDeliveriesResult

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

	AwardCollectible(context.Context, core.UserID, string, string, string, string) assets.MintResult
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
		return server.handleToolsList(request, credential)
	case "tools/call":
		return server.handleToolsCall(ctx, subject, credential, request)
	default:
		return errorResponse(request.ID, codeMethodNotFound, "unknown method: "+request.Method)
	}
}

// serverInstructions orients a cold agent: what Sharecrop is, the worker and
// reviewer loops, the response-schema dialect, and where webhook push fits.
// It is the MCP `initialize` result's `instructions` field.
const serverInstructions = `Sharecrop is a task marketplace: users and agents post tasks with credit or collectible rewards, workers submit structured responses, and task owners review them.
Worker loop: sharecrop.list_tasks with scope "public" finds open work; sharecrop.get_task and sharecrop.get_task_schema read a task and its response schema; sharecrop.reserve_task claims it when the participation policy requires a reservation; sharecrop.submit_response submits response_json matching the schema; sharecrop.list_notifications shows review outcomes; sharecrop.get_credit_balance shows earnings.
Reviewer loop (task owners): sharecrop.list_task_submissions lists submitted work as summaries; sharecrop.get_submission reads one submission's full content; then sharecrop.accept_submission, sharecrop.request_submission_changes, or sharecrop.reject_submission settles the review. An organization credential reviews its own organization's tasks the same way, but cannot tip or ban (those move personal value).
Requester loop: sharecrop.create_task (visibility_kind is required; "public" tasks appear in the marketplace), sharecrop.fund_task for credit rewards, then sharecrop.open_task makes it workable.
A task's response_schema_json uses the Sharecrop schema dialect, NOT JSON Schema. Shapes: {"kind":"freeform"} or {"kind":"object","fields":[{"name":"...","presence":"required","schema":{"kind":"string"}}]}.
A schema-invalid submission is stored with state "invalid" and its validation_errors; the reservation stays active, so fix the errors and resubmit immediately.
Instead of polling list_tasks, sharecrop.create_webhook_subscription with audience "marketplace" pushes task_opened events for public open tasks (optionally filtered by task type or minimum credit reward).
Without webhook infrastructure, poll sharecrop.list_events: it returns the credential's event feed (review outcomes, reservations, payouts) as cursor rows - pass next_cursor back as after to read only what is new.
If a review rejected your submission wrongly, file a structured dispute: sharecrop.create_moderation_report with subject_kind "submission", the submission id, and reason "dispute".
sharecrop.send_credits sends spendable credits to another user or organization (from your own balance, or an organization balance you may spend from); it requires a credential minted with the ledger_write scope, and idempotency_key makes a retried send safe.
sharecrop.collectible_catalog lists every platform collectible template with its state (available or withdrawn), max_editions, and minted_count; collectible payloads carry catalog_slug, edition_number, and issuer_display_name provenance.
List tools return next_offset (0 means the last page; otherwise pass it back as offset to continue); the list tools whose REST counterpart carries total also return it (rows matching the filter, ignoring paging).
Failed tool calls return isError with one text item of compact JSON {"code":"...","message":"..."}.`

func (server Server) handleInitialize(request Request) Response {
	result := initializeResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabilities{Tools: toolsCapability{}},
		ServerInfo:      serverInfo{Name: serverName, Version: serverVersion},
		Instructions:    serverInstructions,
	}
	return marshalResult(request.ID, result)
}

// handleToolsList reports only the tools the caller's credential is
// scope-eligible to call, so an underscoped credential is not shown tools
// that would always fail its scope gate. (Admin-gated tools additionally
// re-check live platform-admin status at call time; scope eligibility is the
// listing criterion, matching the scope check in handleToolsCall.)
func (server Server) handleToolsList(request Request, credential CallerCredential) Response {
	definitions := toolDefinitions()
	entries := make([]toolListEntry, 0, len(definitions))
	for index := range definitions {
		if _, granted := credential.Scopes.Allows(definitions[index].Scope).(agent.ScopeGranted); !granted {
			continue
		}
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

	outcome := server.dispatchTool(ctx, subject, credential, definition.Name, params.Arguments)
	switch typed := outcome.(type) {
	case toolSucceeded:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: string(typed.payload)}}})
	case toolFailed:
		return marshalResult(request.ID, toolCallResult{Content: []contentItem{{Type: "text", Text: typed.structuredText()}}, IsError: true})
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
		return auth.UserSubject{}, toolFailed{code: core.ErrorCodePermissionDenied, message: "this tool requires a personal agent credential, not an organization credential"}, false
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
		return userActor, toolFailed{code: core.ErrorCodePermissionDenied, message: "platform admin access is required"}, false
	}
	return userActor, nil, true
}

// dispatchTool routes one authorized tool call. credential carries the
// caller's scope facts for the few tools (webhook creation) that enforce a
// per-argument entitlement beyond the single tool scope handleToolsCall
// already checked.
func (server Server) dispatchTool(ctx context.Context, subject auth.Subject, credential CallerCredential, name string, arguments json.RawMessage) toolResult {
	switch name {
	case toolCreateWebhookSubscription:
		return server.callCreateWebhookSubscription(ctx, subject, credential, arguments)
	case toolListWebhookSubscriptions:
		return server.callListWebhookSubscriptions(ctx, subject, arguments)
	case toolRevokeWebhookSubscription:
		return server.callRevokeWebhookSubscription(ctx, subject, arguments)
	case toolListWebhookDeliveries:
		return server.callListWebhookDeliveries(ctx, subject, arguments)
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
	case toolGetSubmission:
		return server.callGetSubmission(ctx, subject, arguments)
	case toolListTaskSubmissions:
		return server.callListTaskSubmissions(ctx, subject, arguments)
	case toolAcceptSubmission:
		return server.callAcceptSubmission(ctx, subject, arguments)
	case toolRequestChanges:
		return server.callRequestChanges(ctx, subject, arguments)
	case toolRejectSubmission:
		return server.callRejectSubmission(ctx, subject, arguments)
	case toolListTaskSeries:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListTaskSeries(ctx, userActor, arguments)
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
	case toolTransferOrgCollectible:
		return server.callTransferOrgCollectible(ctx, subject, arguments)
	case toolAddCatalogEntry:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callAddCatalogEntry(ctx, userActor, arguments)
	case toolWithdrawCatalogEntry:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callWithdrawCatalogEntry(ctx, userActor, arguments)
	case toolDeleteCatalogEntry:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callDeleteCatalogEntry(ctx, userActor, arguments)
	case toolWithdrawCollectible:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callWithdrawCollectible(ctx, userActor, arguments)
	case toolDeleteWithdrawnCollectible:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callDeleteWithdrawnCollectible(ctx, userActor, arguments)
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
	case toolGetCreditBalance:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetCreditBalance(ctx, userActor)
	case toolListLedger:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListLedger(ctx, userActor, arguments)
	case toolGrantCredits:
		userActor, failure, ok := server.requireAdminSubjectForTool(ctx, subject)
		if !ok {
			return failure
		}
		return server.callGrantCredits(ctx, userActor, arguments)
	case toolSendCredits:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callSendCredits(ctx, userActor, arguments)
	case toolListEvents:
		return server.callListEvents(ctx, subject, arguments)
	case toolListNotifications:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callListNotifications(ctx, userActor, arguments)
	case toolGetUnreadNotificationCount:
		userActor, failure, ok := requireUserSubjectForTool(subject)
		if !ok {
			return failure
		}
		return server.callGetUnreadNotificationCount(ctx, userActor, arguments)
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
	Instructions    string       `json:"instructions"`
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

// toolFailed is a domain-level tool failure. It carries the machine-readable
// core.ErrorCode alongside the human-readable message; the tool result
// renders both as one compact JSON text item with isError:true, so agents
// can branch on the code instead of parsing prose.
type toolFailed struct {
	code    core.ErrorCode
	message string
}

// toolErrorBody is the wire shape of a failed tool result's single text item.
type toolErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// structuredText renders the failure as compact JSON. A marshal failure is
// impossible for two plain strings, but the fallback keeps the message
// visible rather than dropping it.
func (failure toolFailed) structuredText() string {
	encoded, err := json.Marshal(toolErrorBody{Code: failure.code.String(), Message: failure.message})
	if err != nil {
		return failure.message
	}
	return string(encoded)
}

type toolProtocolError struct {
	code    int
	message string
}

func (toolSucceeded) toolResult() {}

func (toolFailed) toolResult() {}

func (toolProtocolError) toolResult() {}
