package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/assets"
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
)

// mcpServices adapts the HTTP server's domain services to the MCP tool surface.
type mcpServices struct {
	taskService          TaskService
	submissionService    SubmissionService
	ledgerService        LedgerService
	organizationService  OrganizationService
	orgCredentialService OrgCredentialService
	assetService         AssetService
	notificationService  NotificationService
	authService          AuthService
	platformAdmins       PlatformAdminService
	moderationTriage     ModerationTriageService
	privacyService       PrivacyService
	auditService         AuditService
	webhookService       webhook.Service
	eventStore           event.Store
}

func (services mcpServices) CreateWebhookSubscription(ctx context.Context, owner webhook.Owner, endpoint webhook.EndpointURL, kinds webhook.KindFilter, audience webhook.Audience) webhook.CreateResult {
	return services.webhookService.Create(ctx, owner, endpoint, kinds, audience)
}

func (services mcpServices) ListWebhookSubscriptions(ctx context.Context, owner webhook.Owner, page core.Page) webhook.ListResult {
	return services.webhookService.List(ctx, owner, page)
}

func (services mcpServices) RevokeWebhookSubscription(ctx context.Context, owner webhook.Owner, id core.WebhookSubscriptionID) webhook.RevokeResult {
	return services.webhookService.Revoke(ctx, owner, id)
}

func (services mcpServices) ListWebhookDeliveries(ctx context.Context, owner webhook.Owner, id core.WebhookSubscriptionID, page core.Page) webhook.ListDeliveriesResult {
	return services.webhookService.ListDeliveries(ctx, owner, id, page)
}

func (services mcpServices) ListTasks(ctx context.Context, subject auth.Subject, scope task.ListScope, filters task.ListFilters, page core.Page) task.ListResult {
	return services.taskService.List(ctx, subject, scope, filters, page)
}

func (services mcpServices) GetTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.GetResult {
	return services.taskService.Get(ctx, subject, taskID)
}

func (services mcpServices) CreateTask(ctx context.Context, command task.CreateCommand) task.CreateResult {
	return services.taskService.Create(ctx, command)
}

func (services mcpServices) OpenTask(ctx context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	return services.taskService.Open(ctx, subject, taskID)
}

func (services mcpServices) CancelTask(ctx context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	return services.taskService.Cancel(ctx, subject, taskID)
}

func (services mcpServices) FundTask(ctx context.Context, funder core.UserID, taskID core.TaskID, amount ledger.CreditAmount, key ledger.IdempotencyKey) ledger.FundResult {
	return services.ledgerService.FundTask(ctx, funder, taskID, amount, key)
}

func (services mcpServices) RefundTask(ctx context.Context, requester core.UserID, taskID core.TaskID, key ledger.IdempotencyKey) ledger.RefundResult {
	return services.ledgerService.RefundTask(ctx, requester, taskID, key)
}

func (services mcpServices) SubmitResponse(ctx context.Context, command submission.SubmitCommand) submission.SubmitResult {
	return services.submissionService.Submit(ctx, command)
}

func (services mcpServices) GetSubmissionStatus(ctx context.Context, token submission.ReceiptTokenPlain) submission.ReceiptStatusResult {
	return services.submissionService.FindByReceipt(ctx, token)
}

func (services mcpServices) GetSubmission(ctx context.Context, subject auth.Subject, submissionID core.SubmissionID) submission.GetResult {
	return services.submissionService.Get(ctx, subject, submissionID)
}

func (services mcpServices) ListTaskSubmissions(ctx context.Context, subject auth.Subject, taskID core.TaskID, page core.Page) submission.ListResult {
	return services.submissionService.ListForTask(ctx, subject, taskID, page)
}

// The MCP review tools pass the ledger.Reviewer union straight through, so
// an organization credential reviews its own organization's tasks with the
// same authorization REST's review routes grant it.
func (services mcpServices) ReviewAcceptSubmission(ctx context.Context, reviewer ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, creditSelection ledger.CreditReviewSelection, tipSelection ledger.TipSelection, collectibleTip ledger.CollectibleTipSelection) ledger.AcceptResult {
	return services.ledgerService.ReviewAcceptSubmission(ctx, reviewer, taskID, submissionID, key, creditSelection, tipSelection, collectibleTip)
}

func (services mcpServices) RequestChanges(ctx context.Context, reviewer ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, note submission.ReviewNote) ledger.RequestChangesResult {
	return services.ledgerService.RequestChanges(ctx, reviewer, taskID, submissionID, key, note)
}

func (services mcpServices) RejectSubmission(ctx context.Context, reviewer ledger.Reviewer, taskID core.TaskID, submissionID core.SubmissionID, key ledger.IdempotencyKey, note submission.ReviewNote, creditSelection ledger.CreditReviewSelection, tipSelection ledger.TipSelection, banSelection ledger.BanSelection) ledger.RejectResult {
	return services.ledgerService.RejectSubmission(ctx, reviewer, taskID, submissionID, key, note, creditSelection, tipSelection, banSelection)
}

func (services mcpServices) ListSeries(ctx context.Context, subject auth.UserSubject, page core.Page) task.ListSeriesResult {
	return services.taskService.ListSeries(ctx, subject, page)
}

func (services mcpServices) GetSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID) task.GetSeriesResult {
	return services.taskService.GetSeries(ctx, subject, seriesID)
}

func (services mcpServices) CreateSeries(ctx context.Context, subject auth.UserSubject, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationResult {
	return services.taskService.CreateSeries(ctx, subject, title, description)
}

func (services mcpServices) UpdateSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, title task.SeriesTitle, description task.SeriesDescription) task.SeriesMutationResult {
	return services.taskService.UpdateSeries(ctx, subject, seriesID, title, description)
}

func (services mcpServices) ChangeSeriesState(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, transition task.SeriesStateTransition) task.SeriesMutationResult {
	return services.taskService.ChangeSeriesState(ctx, subject, seriesID, transition)
}

func (services mcpServices) AddTaskToSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationResult {
	return services.taskService.AddTaskToSeries(ctx, subject, seriesID, taskID)
}

func (services mcpServices) RemoveTaskFromSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, taskID core.TaskID) task.SeriesMutationResult {
	return services.taskService.RemoveTaskFromSeries(ctx, subject, seriesID, taskID)
}

func (services mcpServices) ReorderSeries(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, order []core.TaskID) task.SeriesMutationResult {
	return services.taskService.ReorderSeries(ctx, subject, seriesID, order)
}

func (services mcpServices) AddSeriesComment(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, body task.CommentBody) task.SeriesCommentResult {
	return services.taskService.AddSeriesComment(ctx, subject, seriesID, body)
}

func (services mcpServices) ListSeriesComments(ctx context.Context, subject auth.UserSubject, seriesID core.TaskSeriesID, page core.Page) task.SeriesCommentsResult {
	return services.taskService.ListSeriesComments(ctx, subject, seriesID, page)
}

func (services mcpServices) UnpublishTask(ctx context.Context, subject auth.Subject, taskID core.TaskID) task.ChangeStateResult {
	return services.taskService.Unpublish(ctx, subject, taskID)
}

func (services mcpServices) AddTaskComment(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, body task.CommentBody) task.TaskCommentResult {
	return services.taskService.AddTaskComment(ctx, subject, taskID, body)
}

func (services mcpServices) ListTaskComments(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, page core.Page) task.TaskCommentsResult {
	return services.taskService.ListTaskComments(ctx, subject, taskID, page)
}

func (services mcpServices) AddSubmissionComment(ctx context.Context, subject auth.UserSubject, submissionID core.SubmissionID, body task.CommentBody) submission.SubmissionCommentResult {
	return services.submissionService.AddSubmissionComment(ctx, subject, submissionID, body)
}

func (services mcpServices) ListSubmissionComments(ctx context.Context, subject auth.UserSubject, submissionID core.SubmissionID, page core.Page) submission.SubmissionCommentsResult {
	return services.submissionService.ListSubmissionComments(ctx, subject, submissionID, page)
}

func (services mcpServices) ReserveTask(ctx context.Context, subject auth.UserSubject, taskID core.TaskID) task.ReservationResult {
	return services.taskService.Reserve(ctx, subject, taskID)
}

func (services mcpServices) ReserveTaskForOrganizationTeam(ctx context.Context, subject auth.UserSubject, taskID core.TaskID, organizationID core.OrganizationID, teamID core.TeamID) task.ReservationResult {
	return services.taskService.ReserveForOrganizationTeam(ctx, subject, taskID, organizationID, teamID)
}

func (services mcpServices) ListReservations(ctx context.Context, subject auth.Subject, taskID core.TaskID, page core.Page) task.ReservationsListResult {
	return services.taskService.ListReservations(ctx, subject, taskID, page)
}

func (services mcpServices) CancelReservation(ctx context.Context, subject auth.Subject, taskID core.TaskID, reservationID core.TaskReservationID) task.ReservationStateChangeResult {
	return services.taskService.CancelReservation(ctx, subject, taskID, reservationID)
}

func (services mcpServices) CreateOrganization(ctx context.Context, subject auth.UserSubject, name org.OrganizationName) org.CreateOrganizationResult {
	return services.organizationService.CreateOrganization(ctx, subject, name)
}

func (services mcpServices) ListOrganizations(ctx context.Context, subject auth.UserSubject, query string, page core.Page) org.ListOrganizationsResult {
	return services.organizationService.ListOrganizations(ctx, subject, query, page)
}

func (services mcpServices) ListOrganizationMembers(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, page core.Page) org.ListMembersResult {
	return services.organizationService.ListMembers(ctx, subject, organizationID, page)
}

func (services mcpServices) ProvisionOrganizationMember(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, email auth.EmailAddress, roles []org.Role) org.ProvisionMemberResult {
	return services.organizationService.ProvisionMember(ctx, subject, organizationID, email, roles)
}

func (services mcpServices) DeactivateOrganizationMember(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, userID core.UserID) org.DeactivateMemberResult {
	return services.organizationService.DeactivateMember(ctx, subject, organizationID, userID)
}

func (services mcpServices) UpdateOrganizationMemberRoles(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, userID core.UserID, roles []org.Role) org.UpdateMemberRolesResult {
	return services.organizationService.UpdateMemberRoles(ctx, subject, organizationID, userID, roles)
}

func (services mcpServices) CreateOrganizationTeam(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, name org.TeamName) org.CreateTeamResult {
	return services.organizationService.CreateOrganizationTeam(ctx, subject, organizationID, name)
}

func (services mcpServices) ListOrganizationTeams(ctx context.Context, subject auth.UserSubject, organizationID core.OrganizationID, query string, page core.Page) org.ListTeamsResult {
	return services.organizationService.ListOrganizationTeams(ctx, subject, organizationID, query, page)
}

func (services mcpServices) CreateStandaloneTeam(ctx context.Context, subject auth.UserSubject, name org.TeamName) org.CreateTeamResult {
	return services.organizationService.CreateStandaloneTeam(ctx, subject, name)
}

func (services mcpServices) ListStandaloneTeams(ctx context.Context, subject auth.UserSubject, query string, page core.Page) org.ListTeamsResult {
	return services.organizationService.ListStandaloneTeams(ctx, subject, query, page)
}

func (services mcpServices) GetTeam(ctx context.Context, subject auth.Subject, teamID core.TeamID) org.GetTeamResult {
	return services.organizationService.GetTeam(ctx, subject, teamID)
}

func (services mcpServices) GetTeamWork(ctx context.Context, subject auth.UserSubject, teamID core.TeamID, filters task.ListFilters, page core.Page) task.ListResult {
	teamResult := services.organizationService.GetTeam(ctx, subject, teamID)
	if _, matched := teamResult.(org.TeamGot); !matched {
		return task.ListRejected{Reason: teamResult.(org.GetTeamRejected).Reason}
	}
	return services.taskService.List(ctx, subject, task.TeamListScope{TeamID: teamID, IncludeReserved: true}, filters, page)
}

func (services mcpServices) AddTeamMember(ctx context.Context, subject auth.Subject, teamID core.TeamID, email auth.EmailAddress) org.AddTeamMemberResult {
	return services.organizationService.AddTeamMember(ctx, subject, teamID, email)
}

func (services mcpServices) CheckOrganizationPermission(ctx context.Context, organizationID core.OrganizationID, userID core.UserID, permission org.Permission) org.PermissionCheck {
	return services.organizationService.CheckOrganizationPermission(ctx, organizationID, userID, permission)
}

func (services mcpServices) CreateOrgCredential(ctx context.Context, organizationID core.OrganizationID, label agent.Label, scopes agent.ScopeSet, expiresAt *time.Time) orgcred.CreateResult {
	return services.orgCredentialService.Create(ctx, organizationID, label, scopes, expiresAt)
}

func (services mcpServices) ListOrgCredentials(ctx context.Context, organizationID core.OrganizationID, page core.Page) orgcred.ListResult {
	return services.orgCredentialService.List(ctx, organizationID, page)
}

func (services mcpServices) RevokeOrgCredential(ctx context.Context, organizationID core.OrganizationID, credentialID core.OrgCredentialID) orgcred.RevokeResult {
	return services.orgCredentialService.Revoke(ctx, organizationID, credentialID)
}

func (services mcpServices) MintCollectible(ctx context.Context, issuer core.UserID, ownerKind string, ownerID string, organizationID string, name assets.CollectibleName, kind assets.CollectibleKind, policy assets.TransferPolicy, art string) assets.MintResult {
	return services.assetService.Mint(ctx, issuer, ownerKind, ownerID, organizationID, name, kind, policy, art)
}

func (services mcpServices) ListCollectibleCatalog(ctx context.Context) assets.CatalogListResult {
	return services.assetService.ListCatalog(ctx)
}

func (services mcpServices) ListCollectibles(ctx context.Context, owner core.UserID, page core.Page) assets.ListResult {
	return services.assetService.ListCollectibles(ctx, owner, page)
}

func (services mcpServices) ListCollectiblesByOwner(ctx context.Context, ownerKind string, ownerID string, page core.Page) assets.ListResult {
	return services.assetService.ListByOwner(ctx, ownerKind, ownerID, page)
}

func (services mcpServices) TransferCollectible(ctx context.Context, from core.UserID, to core.UserID, collectibleID core.CollectibleID) assets.GiftResult {
	return services.assetService.GiftCollectible(ctx, from, to, collectibleID)
}

func (services mcpServices) TransferCollectibleToOrganization(ctx context.Context, from core.UserID, organizationID core.OrganizationID, collectibleID core.CollectibleID) assets.GiftResult {
	return services.assetService.TransferCollectibleToOrganization(ctx, from, organizationID, collectibleID)
}

func (services mcpServices) TransferCollectibleFromOrganization(ctx context.Context, actor core.UserID, organizationID core.OrganizationID, to core.UserID, collectibleID core.CollectibleID) assets.GiftResult {
	return services.assetService.TransferCollectibleFromOrganization(ctx, actor, organizationID, to, collectibleID)
}

func (services mcpServices) AwardOrganizationCollectible(ctx context.Context, organizationID core.OrganizationID, collectibleID core.CollectibleID, recipient core.UserID) assets.GiftResult {
	return services.assetService.AwardOrganizationCollectible(ctx, organizationID, collectibleID, recipient)
}

func (services mcpServices) AddCatalogEntry(ctx context.Context, entry assets.CatalogEntry) assets.CatalogMutationResult {
	return services.assetService.AddCatalogEntry(ctx, entry)
}

func (services mcpServices) WithdrawCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return services.assetService.WithdrawCatalogEntry(ctx, slug)
}

func (services mcpServices) DeleteCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return services.assetService.DeleteCatalogEntry(ctx, slug)
}

func (services mcpServices) WithdrawCollectible(ctx context.Context, actor core.UserID, collectibleID core.CollectibleID) assets.WithdrawResult {
	return services.assetService.WithdrawCollectible(ctx, actor, collectibleID)
}

func (services mcpServices) ReleaseCatalogEntry(ctx context.Context, slug assets.CatalogSlug) assets.CatalogMutationResult {
	return services.assetService.ReleaseCatalogEntry(ctx, slug)
}

func (services mcpServices) ReleaseCollectible(ctx context.Context, actor core.UserID, collectibleID core.CollectibleID) assets.ReleaseResult {
	return services.assetService.ReleaseCollectible(ctx, actor, collectibleID)
}

func (services mcpServices) DeleteWithdrawnCollectible(ctx context.Context, collectibleID core.CollectibleID) assets.DeleteCollectibleResult {
	return services.assetService.DeleteWithdrawnCollectible(ctx, collectibleID)
}

func (services mcpServices) FundCollectibleReward(ctx context.Context, funder core.UserID, taskID core.TaskID, collectibleID core.CollectibleID) assets.FundRewardResult {
	return services.assetService.FundReward(ctx, funder, taskID, collectibleID)
}

func (services mcpServices) RefundCollectibleReward(ctx context.Context, requester core.UserID, taskID core.TaskID) assets.RefundRewardResult {
	return services.assetService.RefundReward(ctx, requester, taskID)
}

func (services mcpServices) ListNotifications(ctx context.Context, recipient core.UserID, filter notification.StateFilter, page core.Page) notification.ListResult {
	return services.notificationService.List(ctx, recipient, filter, page)
}

func (services mcpServices) CountUnreadNotifications(ctx context.Context, recipient core.UserID) notification.CountResult {
	return services.notificationService.CountUnread(ctx, recipient)
}

func (services mcpServices) MarkNotificationRead(ctx context.Context, recipient core.UserID, notificationID core.NotificationID) notification.MarkReadResult {
	return services.notificationService.MarkRead(ctx, recipient, notificationID)
}

// ListEvents serves the MCP event-feed tool through the same store reads the
// REST feed (GET /api/events) uses: a personal agent credential's user
// subject reads the owner's recipient-scoped feed, and an organization
// credential's org subject reads the organization's subject events. MCP has
// no other subject kinds (verifyMCPCaller only produces these two), so the
// default arm is unreachable in practice but keeps the boundary total.
func (services mcpServices) ListEvents(ctx context.Context, subject auth.Subject, filter event.CursorFilter, page core.Page) event.ListStoreResult {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return services.eventStore.ListForRecipient(ctx, typed.ID, filter, page)
	case auth.OrgSubject:
		return services.eventStore.ListForOrganization(ctx, typed.ID, filter, page)
	default:
		return event.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeUnauthenticated, "an agent credential or organization credential is required")}
	}
}

func (services mcpServices) GetCreditBalance(ctx context.Context, owner core.UserID) ledger.BalanceResult {
	return services.ledgerService.Balance(ctx, owner)
}

func (services mcpServices) ListLedger(ctx context.Context, owner core.UserID, page core.Page) ledger.ListEntriesResult {
	return services.ledgerService.ListEntries(ctx, owner, page)
}

func (services mcpServices) GrantCredits(ctx context.Context, actingAdmin core.UserID, target ledger.GrantTarget, amount ledger.CreditAmount, note ledger.GrantNote, key ledger.IdempotencyKey) ledger.GrantResult {
	return services.ledgerService.GrantCredits(ctx, actingAdmin, target, amount, note, key)
}

func (services mcpServices) SendCredits(ctx context.Context, actor core.UserID, source ledger.TransferSource, target ledger.TransferTarget, amount ledger.CreditAmount, note ledger.TransferNote, key ledger.IdempotencyKey) ledger.SendResult {
	return services.ledgerService.SendCredits(ctx, actor, source, target, amount, note, key)
}

func (services mcpServices) ListUsers(ctx context.Context, query string, page core.Page) auth.UserDirectoryResult {
	return services.authService.ListUsers(ctx, query, page)
}

func (services mcpServices) GetUserProfile(ctx context.Context, subject auth.UserSubject, userID core.UserID, page core.Page) task.ListResult {
	return services.taskService.List(ctx, subject, task.CreatorListScope{CreatorID: userID}, task.NoListFilters(), page)
}

func (services mcpServices) GetUserWork(ctx context.Context, subject auth.UserSubject, userID core.UserID, page core.Page) task.ListResult {
	return services.taskService.List(ctx, subject, task.AssigneeListScope{AssigneeID: userID}, task.NoListFilters(), page)
}

func (services mcpServices) GetUserSubmissions(ctx context.Context, subject auth.UserSubject, userID core.UserID, page core.Page) submission.ListResult {
	return services.submissionService.ListForSubmitter(ctx, subject, userID, page)
}

type agentCredentialRequest struct {
	Label  string   `json:"label"`
	Scopes []string `json:"scopes"`
	// ExpiresAt is an optional RFC3339 expiry; absent mints a non-expiring
	// credential.
	ExpiresAt string `json:"expires_at,omitempty"`
}

type agentCredentialResponse struct {
	ID        string   `json:"id"`
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	State     string   `json:"state"`
	ExpiresAt string   `json:"expires_at"`
	TaskID    string   `json:"task_id"`
}

type agentCredentialCreatedResponse struct {
	Credential agentCredentialResponse `json:"credential"`
	Secret     string                  `json:"secret"`
}

type agentCredentialsResponse struct {
	Credentials []agentCredentialResponse `json:"credentials"`
	NextOffset  int                       `json:"next_offset"`
}

func (agentCredentialResponse) writableResponse() {}

func (agentCredentialCreatedResponse) writableResponse() {}

func (agentCredentialsResponse) writableResponse() {}

func (server Server) getTask(w http.ResponseWriter, r *http.Request) {
	taskIDResult := parseTaskPathValue(r)
	taskIDAccepted, taskIDMatched := taskIDResult.(taskIDAccepted)
	if !taskIDMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, taskIDResult.(taskIDRejected).reason)
		return
	}

	actorResult := server.requireWorkerSubject(r, agent.ScopeTasksRead, taskIDAccepted.value)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	result := server.taskService.Get(r.Context(), actor.subject, taskIDAccepted.value)
	got, matched := result.(task.TaskGot)
	if !matched {
		writeDomainError(w, result.(task.GetRejected).Reason)
		return
	}

	response, fundingErr := server.taskToResponseForActor(r.Context(), actor.subject, got.Value)
	if fundingErr != nil {
		writeDomainError(w, *fundingErr)
		return
	}
	response.CreatorDisplayName = got.CreatorDisplayName.String()
	writeTaskResponse(w, http.StatusOK, response)
}

// credentialFields is the label/scopes/expiration triple common to both a
// personal agent credential and an org-wide credential request body.
type credentialFields struct {
	label     agent.Label
	scopes    agent.ScopeSet
	expiresAt *time.Time
}

// parseCredentialFields decodes and validates the label/scopes/expiration
// fields shared by the agent-credential and org-credential mint requests,
// writing an error response and returning false on the first invalid field.
func parseCredentialFields(w http.ResponseWriter, rawLabel string, rawScopes []string, rawExpiresAt string) (credentialFields, bool) {
	labelResult := agent.NewLabel(rawLabel)
	label, labelMatched := labelResult.(agent.LabelAccepted)
	if !labelMatched {
		writeDomainError(w, labelResult.(agent.LabelRejected).Reason)
		return credentialFields{}, false
	}

	scopesResult := parseAgentScopes(rawScopes)
	scopes, scopesMatched := scopesResult.(agentScopesAccepted)
	if !scopesMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, scopesResult.(agentScopesRejected).reason)
		return credentialFields{}, false
	}

	expiresAtResult := parseOptionalExpiresAt(rawExpiresAt)
	expiresAt, expiresAtMatched := expiresAtResult.(expiresAtAccepted)
	if !expiresAtMatched {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, expiresAtResult.(expiresAtRejected).reason)
		return credentialFields{}, false
	}

	return credentialFields{label: label.Value, scopes: scopes.value, expiresAt: expiresAt.value}, true
}

func (server Server) createAgentCredential(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	if !server.allowBySubject(w, actor.subject.ID.String()) {
		return
	}

	var request agentCredentialRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	fields, ok := parseCredentialFields(w, request.Label, request.Scopes, request.ExpiresAt)
	if !ok {
		return
	}

	result := server.agentService.Create(r.Context(), actor.subject.ID, fields.label, fields.scopes, fields.expiresAt, nil)
	created, matched := result.(agent.CredentialCreated)
	if !matched {
		writeDomainError(w, result.(agent.CreateRejected).Reason)
		return
	}

	writeJSON(w, http.StatusCreated, agentCredentialCreatedResponse{
		Credential: credentialToResponse(created.Value),
		Secret:     created.Secret.String(),
	})
}

func (server Server) listAgentCredentials(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.agentService.List(r.Context(), actor.subject.ID, page.Probe())
	listed, matched := result.(agent.CredentialsListed)
	if !matched {
		writeDomainError(w, result.(agent.ListRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := agentCredentialsResponse{Credentials: make([]agentCredentialResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Credentials = append(response.Credentials, credentialToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) revokeAgentCredential(w http.ResponseWriter, r *http.Request) {
	actorResult := server.requireUserSubject(r)
	actor, actorMatched := actorResult.(userSubjectAccepted)
	if !actorMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, actorResult.(userSubjectRejected).reason)
		return
	}

	credentialIDResult := core.ParseAgentCredentialID(r.PathValue("credential_id"))
	credentialID, credentialMatched := credentialIDResult.(core.AgentCredentialIDCreated)
	if !credentialMatched {
		writeDomainError(w, credentialIDResult.(core.AgentCredentialIDRejected).Reason)
		return
	}

	result := server.agentService.Revoke(r.Context(), actor.subject.ID, credentialID.Value)
	revoked, matched := result.(agent.CredentialRevoked)
	if !matched {
		writeDomainError(w, result.(agent.RevokeRejected).Reason)
		return
	}

	writeJSON(w, http.StatusOK, credentialToResponse(revoked.Value))
}

func (server Server) mcpEndpoint(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, "origin is not allowed")
		return
	}
	if !mcpAcceptAllowed(r.Header.Get("Accept")) {
		writeError(w, http.StatusNotAcceptable, core.ErrorCodeInvalidArgument, "MCP endpoint requires an Accept header allowing application/json")
		return
	}
	if !mcpProtocolVersionAllowed(r.Header.Get("MCP-Protocol-Version")) {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "MCP protocol version is unsupported")
		return
	}

	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMCPBodyBytes))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, core.ErrorCodeInvalidArgument, "request body exceeds the MCP size limit")
		return
	}
	requestInfo := classifyMCPBody(body)
	if requestInfo.invalid {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is not valid JSON-RPC")
		return
	}

	verifyResult := server.verifyMCPCaller(r)
	verified, verifiedMatched := verifyResult.(mcpCallerVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, verifyResult.(mcpCallerRejected).reason)
		return
	}
	subjectIdentity := mcpSubjectIdentity(verified.subject)

	if !server.subjectRateLimiter.Allow(subjectIdentity) {
		writeError(w, http.StatusTooManyRequests, core.ErrorCodeRateLimited, "too many MCP requests; slow down and retry")
		return
	}

	sessionID := r.Header.Get(mcpSessionHeader)
	if requestInfo.initializes {
		if sessionID != "" {
			writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "initialize requests must not include an MCP session id")
			return
		}
	} else if sessionID == "" {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "MCP session id is required")
		return
	} else if !server.mcpSessions.existsForSubject(sessionID, subjectIdentity) {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "MCP session was not found")
		return
	}

	result := server.mcpServer.HandleRaw(r.Context(), verified.subject, verified.credential, body)
	if result.SessionID != "" {
		if !server.mcpSessions.create(result.SessionID, subjectIdentity) {
			writeError(w, http.StatusTooManyRequests, core.ErrorCodeRateLimited, "too many active MCP sessions for this agent")
			return
		}
		w.Header().Set(mcpSessionHeader, result.SessionID)
	} else if requestInfo.initializes {
		generatedSessionID := newMCPHTTPSessionID()
		if generatedSessionID == "" {
			writeError(w, http.StatusInternalServerError, core.ErrorCodeUnavailable, "MCP session could not be created")
			return
		}
		if !server.mcpSessions.create(generatedSessionID, subjectIdentity) {
			writeError(w, http.StatusTooManyRequests, core.ErrorCodeRateLimited, "too many active MCP sessions for this agent")
			return
		}
		w.Header().Set(mcpSessionHeader, generatedSessionID)
	}
	if !result.HasResponse {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if sessionID != "" {
		_, _ = server.mcpSessions.appendEvent(sessionID, result.Payload)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(result.Payload)
}

func mcpAcceptAllowed(raw string) bool {
	if raw == "" {
		return true
	}
	for _, value := range strings.Split(raw, ",") {
		mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
		if mediaType == "application/json" || mediaType == "*/*" {
			return true
		}
	}
	return false
}

func mcpProtocolVersionAllowed(raw string) bool {
	return raw == "" || raw == mcp.ProtocolVersion()
}

func (server Server) mcpStream(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, "origin is not allowed")
		return
	}
	if !mcpStreamAcceptAllowed(r.Header.Get("Accept")) {
		writeError(w, http.StatusNotAcceptable, core.ErrorCodeInvalidArgument, "MCP stream requires an Accept header allowing text/event-stream")
		return
	}
	if !mcpProtocolVersionAllowed(r.Header.Get("MCP-Protocol-Version")) {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "MCP protocol version is unsupported")
		return
	}
	verifyResult := server.verifyMCPCaller(r)
	verified, verifiedMatched := verifyResult.(mcpCallerVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, verifyResult.(mcpCallerRejected).reason)
		return
	}
	sessionID := r.Header.Get(mcpSessionHeader)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "MCP session id is required")
		return
	}
	if !server.mcpSessions.existsForSubject(sessionID, mcpSubjectIdentity(verified.subject)) {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "MCP session was not found")
		return
	}
	events, liveEvents, cancel, ok := server.mcpSessions.replayAndSubscribe(sessionID, r.Header.Get(mcpLastEventIDHeader))
	if !ok {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "MCP session was not found")
		return
	}
	defer cancel()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	if len(events) == 0 {
		_, _ = w.Write([]byte(": sharecrop mcp stream ready\n\n"))
	}
	for index := range events {
		writeSSEEvent(w, events[index])
	}
	flusher, canStream := w.(http.Flusher)
	if !canStream {
		// The transport cannot stream - it buffers the whole response and
		// returns it as one unit (the WASI guest bridge). Blocking here to wait
		// for live events would never return and would pin the worker forever,
		// so send the replayed events and stop. The client's EventSource
		// reconnects with Last-Event-ID and picks up later events, turning
		// server push into short polling over a transport that cannot do better.
		return
	}
	flusher.Flush()
	for {
		select {
		case event, open := <-liveEvents:
			if !open {
				return
			}
			writeSSEEvent(w, event)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (server Server) mcpDeleteSession(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r) {
		writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, "origin is not allowed")
		return
	}
	verifyResult := server.verifyMCPCaller(r)
	verified, verifiedMatched := verifyResult.(mcpCallerVerified)
	if !verifiedMatched {
		writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, verifyResult.(mcpCallerRejected).reason)
		return
	}
	sessionID := r.Header.Get(mcpSessionHeader)
	if sessionID == "" {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "MCP session id is required")
		return
	}
	if !server.mcpSessions.existsForSubject(sessionID, mcpSubjectIdentity(verified.subject)) {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "MCP session was not found")
		return
	}
	if !server.mcpSessions.terminate(sessionID) {
		writeError(w, http.StatusNotFound, core.ErrorCodeNotFound, "MCP session was not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

const maxMCPBodyBytes = 1 << 20

type mcpBodyInfo struct {
	initializes bool
	invalid     bool
}

func classifyMCPBody(body []byte) mcpBodyInfo {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return mcpBodyInfo{invalid: true}
	}
	if trimmed[0] == '[' {
		var requests []mcp.Request
		if err := json.Unmarshal(trimmed, &requests); err != nil {
			return mcpBodyInfo{invalid: true}
		}
		for index := range requests {
			if len(requests[index].ID) > 0 && requests[index].Method != "initialize" && !mcpRawClientResponse(requests[index]) {
				return mcpBodyInfo{}
			}
		}
		for index := range requests {
			if requests[index].Method == "initialize" {
				return mcpBodyInfo{initializes: true}
			}
		}
		return mcpBodyInfo{}
	}
	var request mcp.Request
	if err := json.Unmarshal(trimmed, &request); err != nil {
		return mcpBodyInfo{invalid: true}
	}
	return mcpBodyInfo{initializes: request.Method == "initialize"}
}

func mcpRawClientResponse(request mcp.Request) bool {
	return request.Method == "" && len(request.ID) > 0
}

// originAllowed implements the MCP DNS-rebinding protection: requests without
// an Origin header (non-browser agents) are allowed; a browser Origin must
// match the server's own host.
func originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	parsed, err := url.Parse(origin)
	if err != nil {
		return false
	}
	return parsed.Host == r.Host
}

func mcpStreamAcceptAllowed(raw string) bool {
	if raw == "" {
		return false
	}
	for _, value := range strings.Split(raw, ",") {
		mediaType := strings.TrimSpace(strings.Split(value, ";")[0])
		if mediaType == "text/event-stream" || mediaType == "*/*" {
			return true
		}
	}
	return false
}

func (server Server) verifyAgent(r *http.Request) agent.VerifyResult {
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched {
		return agent.VerifyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential is required")}
	}
	secretResult := agent.ParseSecretPlain(rawToken)
	secret, secretMatched := secretResult.(agent.SecretPlainAccepted)
	if !secretMatched {
		return agent.VerifyRejected{Reason: secretResult.(agent.SecretPlainRejected).Reason}
	}
	return server.agentService.Verify(r.Context(), secret.Value)
}

type mcpCallerResult interface {
	mcpCallerResult()
}

type mcpCallerVerified struct {
	subject    auth.Subject
	credential mcp.CallerCredential
}

type mcpCallerRejected struct {
	reason string
}

func (mcpCallerVerified) mcpCallerResult() {}

func (mcpCallerRejected) mcpCallerResult() {}

// verifyMCPCaller resolves the bearer token on an MCP transport request to
// either a personal agent credential or an organization-wide credential
// (dispatched by secret prefix, mirroring requireUserOrOrgSubject's fallback
// for REST endpoints) — MCP has no user-session concept, so unlike REST this
// never tries a user access token.
func (server Server) verifyMCPCaller(r *http.Request) mcpCallerResult {
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched {
		return mcpCallerRejected{reason: "an agent credential or organization credential is required"}
	}
	if orgcred.HasSecretPrefix(rawToken) {
		secretResult := orgcred.ParseSecretPlain(rawToken)
		secret, secretMatched := secretResult.(orgcred.SecretPlainAccepted)
		if !secretMatched {
			return mcpCallerRejected{reason: secretResult.(orgcred.SecretPlainRejected).Reason.Description()}
		}
		verifyResult := server.orgCredentialService.Verify(r.Context(), secret.Value)
		verified, verifyMatched := verifyResult.(orgcred.CredentialVerified)
		if !verifyMatched {
			return mcpCallerRejected{reason: verifyResult.(orgcred.VerifyRejected).Reason.Description()}
		}
		return mcpCallerVerified{
			subject:    verified.Subject,
			credential: mcp.CallerCredential{Scopes: verified.Credential.Scopes, TaskID: nil},
		}
	}
	verifyResult := server.verifyAgent(r)
	verified, verifyMatched := verifyResult.(agent.CredentialVerified)
	if !verifyMatched {
		return mcpCallerRejected{reason: verifyResult.(agent.VerifyRejected).Reason.Description()}
	}
	return mcpCallerVerified{
		subject:    verified.Subject,
		credential: mcp.CallerCredential{Scopes: verified.Credential.Scopes, TaskID: verified.Credential.TaskID},
	}
}

// mcpSubjectIdentity returns a stable string key for an MCP caller's
// subject, used for rate limiting and session-ownership checks regardless
// of whether the caller authenticated as a personal agent credential
// (auth.UserSubject) or an organization-wide one (auth.OrgSubject).
func mcpSubjectIdentity(subject auth.Subject) string {
	switch typed := subject.(type) {
	case auth.UserSubject:
		return "user:" + typed.ID.String()
	case auth.OrgSubject:
		return "org:" + typed.ID.String()
	default:
		return ""
	}
}

func credentialToResponse(value agent.Credential) agentCredentialResponse {
	scopes := value.Scopes.Values()
	rawScopes := make([]string, 0, len(scopes))
	for index := range scopes {
		rawScopes = append(rawScopes, scopes[index].String())
	}
	var rawExpiresAt string
	if value.ExpiresAt != nil {
		rawExpiresAt = value.ExpiresAt.UTC().Format(time.RFC3339)
	}
	var rawTaskID string
	if value.TaskID != nil {
		rawTaskID = value.TaskID.String()
	}
	return agentCredentialResponse{
		ID:        value.ID.String(),
		Label:     value.Label.String(),
		Scopes:    rawScopes,
		State:     value.State.String(),
		ExpiresAt: rawExpiresAt,
		TaskID:    rawTaskID,
	}
}

type expiresAtResult interface {
	expiresAtResult()
}

type expiresAtAccepted struct {
	value *time.Time
}

type expiresAtRejected struct {
	reason string
}

func (expiresAtAccepted) expiresAtResult() {}

func (expiresAtRejected) expiresAtResult() {}

// parseOptionalExpiresAt treats an empty string as "never expires", matching
// this codebase's convention of empty-string sentinels over JSON null for
// optional fields.
func parseOptionalExpiresAt(raw string) expiresAtResult {
	if raw == "" {
		return expiresAtAccepted{value: nil}
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return expiresAtRejected{reason: "expires_at must be an RFC3339 timestamp"}
	}
	return expiresAtAccepted{value: &parsed}
}

type agentScopesResult interface {
	agentScopesResult()
}

type agentScopesAccepted struct {
	value agent.ScopeSet
}

type agentScopesRejected struct {
	reason string
}

func (agentScopesAccepted) agentScopesResult() {}

func (agentScopesRejected) agentScopesResult() {}

func parseAgentScopes(raw []string) agentScopesResult {
	scopes := make([]agent.Scope, 0, len(raw))
	for _, rawScope := range raw {
		scopeResult := agent.ParseScope(rawScope)
		scope, matched := scopeResult.(agent.ScopeAccepted)
		if !matched {
			return agentScopesRejected{reason: scopeResult.(agent.ScopeRejected).Reason.Description()}
		}
		scopes = append(scopes, scope.Value)
	}
	set := agent.NewScopeSet(scopes)
	if set.IsEmpty() {
		return agentScopesRejected{reason: "at least one agent scope is required"}
	}
	return agentScopesAccepted{value: set}
}
