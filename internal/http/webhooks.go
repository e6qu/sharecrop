package httpserver

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
)

type webhookSubscriptionRequest struct {
	URL            string   `json:"url"`
	Kinds          []string `json:"kinds"`
	OrganizationID string   `json:"organization_id"`
	// Audience is "recipient" (the default when absent) or "marketplace".
	// A marketplace subscription receives every public open task_opened
	// event rather than only events addressed to the owner, and requires
	// kinds to be exactly ["task_opened"].
	Audience string `json:"audience"`
	// FilterTaskType optionally narrows a marketplace subscription to one
	// task type. Valid only with audience "marketplace".
	FilterTaskType string `json:"filter_task_type"`
	// FilterMinCreditReward optionally narrows a marketplace subscription
	// to tasks declaring at least this credit reward (a positive integer).
	// Valid only with audience "marketplace".
	FilterMinCreditReward int64 `json:"filter_min_credit_reward"`
}

type webhookSubscriptionResponse struct {
	ID                  string   `json:"id"`
	OwnerKind           string   `json:"owner_kind"`
	OwnerUserID         string   `json:"owner_user_id"`
	OwnerOrganizationID string   `json:"owner_organization_id"`
	URL                 string   `json:"url"`
	Kinds               []string `json:"kinds"`
	State               string   `json:"state"`
	CreatedAt           string   `json:"created_at"`
	// Audience is "recipient" or "marketplace". The filter fields are the
	// marketplace narrowing filters: empty / 0 mean no filter, and they are
	// always empty / 0 on recipient subscriptions.
	Audience              string `json:"audience"`
	FilterTaskType        string `json:"filter_task_type"`
	FilterMinCreditReward int64  `json:"filter_min_credit_reward"`
}

type webhookSubscriptionCreatedResponse struct {
	Subscription webhookSubscriptionResponse `json:"subscription"`
	// Secret appears ONLY in the create response; listings never carry it.
	Secret string `json:"secret"`
}

type webhookSubscriptionsResponse struct {
	Subscriptions []webhookSubscriptionResponse `json:"subscriptions"`
	NextOffset    int                           `json:"next_offset"`
}

type webhookDeliveryResponse struct {
	ID            string `json:"id"`
	EventCursor   string `json:"event_cursor"`
	State         string `json:"state"`
	AttemptCount  int64  `json:"attempt_count"`
	NextAttemptAt string `json:"next_attempt_at"`
	LastStatus    string `json:"last_status"`
}

type webhookDeliveriesResponse struct {
	Deliveries []webhookDeliveryResponse `json:"deliveries"`
	NextOffset int                       `json:"next_offset"`
}

func (webhookSubscriptionResponse) writableResponse() {}

func (webhookSubscriptionCreatedResponse) writableResponse() {}

func (webhookSubscriptionsResponse) writableResponse() {}

func (webhookDeliveriesResponse) writableResponse() {}

// webhookCallerResult resolves who is acting on the webhook routes. It
// mirrors requireUserOrOrgSubject, but keeps the verified organization
// credential visible: the kind-entitlement rule needs the credential's
// scopes, which the generic helper deliberately discards.
type webhookCallerResult interface {
	webhookCallerResult()
}

type webhookCallerUser struct {
	subject auth.UserSubject
}

type webhookCallerOrgCredential struct {
	credential orgcred.Credential
}

type webhookCallerRejected struct {
	reason string
}

type webhookCallerScopeDenied struct {
	reason string
}

func (webhookCallerUser) webhookCallerResult() {}

func (webhookCallerOrgCredential) webhookCallerResult() {}

func (webhookCallerRejected) webhookCallerResult() {}

func (webhookCallerScopeDenied) webhookCallerResult() {}

// resolveWebhookCaller accepts a user session (never scope-checked) or an
// organization credential holding the required scope, exactly like
// requireUserOrOrgSubject. Personal agent credentials manage webhooks over
// MCP, where the same entitlement rule is enforced per tool call.
func (server Server) resolveWebhookCaller(r *http.Request, required agent.Scope) webhookCallerResult {
	if accepted, matched := server.requireUserSubject(r).(userSubjectAccepted); matched {
		return webhookCallerUser{subject: accepted.subject}
	}
	rawHeader := r.Header.Get("Authorization")
	rawToken, matched := strings.CutPrefix(rawHeader, "Bearer ")
	if !matched || !orgcred.HasSecretPrefix(rawToken) {
		return webhookCallerRejected{reason: "a user access token or an organization credential is required"}
	}
	secretResult := orgcred.ParseSecretPlain(rawToken)
	secret, secretMatched := secretResult.(orgcred.SecretPlainAccepted)
	if !secretMatched {
		return webhookCallerRejected{reason: secretResult.(orgcred.SecretPlainRejected).Reason.Description()}
	}
	verifyResult := server.orgCredentialService.Verify(r.Context(), secret.Value)
	verified, verifyMatched := verifyResult.(orgcred.CredentialVerified)
	if !verifyMatched {
		return webhookCallerRejected{reason: verifyResult.(orgcred.VerifyRejected).Reason.Description()}
	}
	if _, granted := verified.Credential.Scopes.Allows(required).(agent.ScopeGranted); !granted {
		return webhookCallerScopeDenied{reason: "organization credential lacks the " + required.String() + " scope"}
	}
	return webhookCallerOrgCredential{credential: verified.Credential}
}

func writeWebhookCallerRejection(w http.ResponseWriter, result webhookCallerResult) {
	if denied, matched := result.(webhookCallerScopeDenied); matched {
		writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, denied.reason)
		return
	}
	writeError(w, http.StatusUnauthorized, core.ErrorCodeUnauthenticated, result.(webhookCallerRejected).reason)
}

// webhookOwnerResult is the outcome of resolving which owner a webhook
// request acts for, after the caller-specific authorization checks.
type webhookOwnerResult interface {
	webhookOwnerResult()
}

type webhookOwnerAccepted struct {
	owner webhook.Owner
}

type webhookOwnerRejected struct {
	status int
	code   core.ErrorCode
	reason string
}

func (webhookOwnerAccepted) webhookOwnerResult() {}

func (webhookOwnerRejected) webhookOwnerResult() {}

// resolveWebhookOwner maps (caller, organization_id) onto the acting owner.
// A user acts for themselves by default, or for an organization they hold
// PermissionManageMembers in — the same gate that mints org credentials,
// since a webhook receives that organization's event stream. An organization
// credential always acts for its own organization; naming a different one is
// rejected.
func (server Server) resolveWebhookOwner(r *http.Request, caller webhookCallerResult, rawOrganizationID string) webhookOwnerResult {
	switch typed := caller.(type) {
	case webhookCallerUser:
		if rawOrganizationID == "" {
			return webhookOwnerAccepted{owner: webhook.OwnerUser{ID: typed.subject.ID}}
		}
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return webhookOwnerRejected{status: http.StatusBadRequest, code: core.ErrorCodeInvalidID, reason: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}
		}
		permissionResult := server.organizationService.CheckOrganizationPermission(r.Context(), organizationID.Value, typed.subject.ID, org.PermissionManageMembers)
		if _, denied := permissionResult.(org.PermissionDenied); denied {
			return webhookOwnerRejected{status: http.StatusForbidden, code: core.ErrorCodePermissionDenied, reason: "organization webhook management access denied"}
		}
		return webhookOwnerAccepted{owner: webhook.OwnerOrganization{ID: organizationID.Value}}
	case webhookCallerOrgCredential:
		if rawOrganizationID != "" && rawOrganizationID != typed.credential.OrganizationID.String() {
			return webhookOwnerRejected{status: http.StatusForbidden, code: core.ErrorCodePermissionDenied, reason: "organization credential may only manage webhooks for its own organization"}
		}
		return webhookOwnerAccepted{owner: webhook.OwnerOrganization{ID: typed.credential.OrganizationID}}
	default:
		return webhookOwnerRejected{status: http.StatusUnauthorized, code: core.ErrorCodeUnauthenticated, reason: "a user access token or an organization credential is required"}
	}
}

func webhookOwnerIdentity(owner webhook.Owner) string {
	switch typed := owner.(type) {
	case webhook.OwnerUser:
		return typed.ID.String()
	case webhook.OwnerOrganization:
		return typed.ID.String()
	default:
		return ""
	}
}

func (server Server) createWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	callerResult := server.resolveWebhookCaller(r, agent.ScopeWebhooksManage)
	switch callerResult.(type) {
	case webhookCallerUser, webhookCallerOrgCredential:
	default:
		writeWebhookCallerRejection(w, callerResult)
		return
	}

	var request webhookSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, core.ErrorCodeInvalidArgument, "request body is invalid")
		return
	}

	endpointResult := webhook.NewEndpointURL(request.URL)
	endpoint, endpointMatched := endpointResult.(webhook.EndpointURLAccepted)
	if !endpointMatched {
		writeDomainError(w, endpointResult.(webhook.EndpointURLRejected).Reason)
		return
	}

	kinds := make([]event.Kind, 0, len(request.Kinds))
	for _, rawKind := range request.Kinds {
		kindResult := event.ParseKind(rawKind)
		kind, kindMatched := kindResult.(event.KindParsed)
		if !kindMatched {
			writeDomainError(w, kindResult.(event.KindRejected).Reason)
			return
		}
		kinds = append(kinds, kind.Value)
	}
	filterResult := webhook.NewKindFilter(kinds)
	filter, filterMatched := filterResult.(webhook.KindFilterAccepted)
	if !filterMatched {
		writeDomainError(w, filterResult.(webhook.KindFilterRejected).Reason)
		return
	}

	// A credential caller must be entitled to read every kind it subscribes
	// to: holding webhooks_manage alone must not widen what the credential
	// could already observe through the matching read scopes.
	if credentialCaller, matched := callerResult.(webhookCallerOrgCredential); matched {
		if missing := webhook.MissingKindEntitlements(credentialCaller.credential.Scopes, filter.Value); len(missing) > 0 {
			writeError(w, http.StatusForbidden, core.ErrorCodePermissionDenied, "organization credential lacks the "+webhook.RequiredScopeForKind(missing[0]).String()+" scope required to subscribe to "+missing[0].String())
			return
		}
	}

	audienceResult := parseWebhookAudience(request)
	audience, audienceMatched := audienceResult.(webhookAudienceAccepted)
	if !audienceMatched {
		writeDomainError(w, audienceResult.(webhookAudienceRejected).reason)
		return
	}

	ownerResult := server.resolveWebhookOwner(r, callerResult, request.OrganizationID)
	owner, ownerMatched := ownerResult.(webhookOwnerAccepted)
	if !ownerMatched {
		rejected := ownerResult.(webhookOwnerRejected)
		writeError(w, rejected.status, rejected.code, rejected.reason)
		return
	}
	if !server.allowBySubject(w, webhookOwnerIdentity(owner.owner)) {
		return
	}

	result := server.webhookService.Create(r.Context(), owner.owner, endpoint.Value, filter.Value, audience.value)
	created, createdMatched := result.(webhook.SubscriptionCreated)
	if !createdMatched {
		writeDomainError(w, result.(webhook.CreateRejected).Reason)
		return
	}
	writeJSON(w, http.StatusCreated, webhookSubscriptionCreatedResponse{
		Subscription: webhookSubscriptionToResponse(created.Value),
		Secret:       created.Secret.String(),
	})
}

func (server Server) listWebhookSubscriptions(w http.ResponseWriter, r *http.Request) {
	callerResult := server.resolveWebhookCaller(r, agent.ScopeWebhooksRead)
	ownerResult := server.resolveWebhookOwner(r, callerResult, r.URL.Query().Get("organization_id"))
	owner, ownerMatched := ownerResult.(webhookOwnerAccepted)
	if !ownerMatched {
		writeWebhookRejection(w, callerResult, ownerResult)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.webhookService.List(r.Context(), owner.owner, page.Probe())
	listed, listedMatched := result.(webhook.SubscriptionsListed)
	if !listedMatched {
		writeDomainError(w, result.(webhook.ListRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := webhookSubscriptionsResponse{Subscriptions: make([]webhookSubscriptionResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Subscriptions = append(response.Subscriptions, webhookSubscriptionToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

func (server Server) revokeWebhookSubscription(w http.ResponseWriter, r *http.Request) {
	callerResult := server.resolveWebhookCaller(r, agent.ScopeWebhooksManage)
	ownerResult := server.resolveWebhookOwner(r, callerResult, r.URL.Query().Get("organization_id"))
	owner, ownerMatched := ownerResult.(webhookOwnerAccepted)
	if !ownerMatched {
		writeWebhookRejection(w, callerResult, ownerResult)
		return
	}

	idResult := core.ParseWebhookSubscriptionID(r.PathValue("subscription_id"))
	id, idMatched := idResult.(core.WebhookSubscriptionIDCreated)
	if !idMatched {
		writeDomainError(w, idResult.(core.WebhookSubscriptionIDRejected).Reason)
		return
	}

	result := server.webhookService.Revoke(r.Context(), owner.owner, id.Value)
	revoked, revokedMatched := result.(webhook.SubscriptionRevoked)
	if !revokedMatched {
		writeDomainError(w, result.(webhook.RevokeRejected).Reason)
		return
	}
	writeJSON(w, http.StatusOK, webhookSubscriptionToResponse(revoked.Value))
}

func (server Server) listWebhookDeliveries(w http.ResponseWriter, r *http.Request) {
	callerResult := server.resolveWebhookCaller(r, agent.ScopeWebhooksRead)
	ownerResult := server.resolveWebhookOwner(r, callerResult, r.URL.Query().Get("organization_id"))
	owner, ownerMatched := ownerResult.(webhookOwnerAccepted)
	if !ownerMatched {
		writeWebhookRejection(w, callerResult, ownerResult)
		return
	}

	idResult := core.ParseWebhookSubscriptionID(r.PathValue("subscription_id"))
	id, idMatched := idResult.(core.WebhookSubscriptionIDCreated)
	if !idMatched {
		writeDomainError(w, idResult.(core.WebhookSubscriptionIDRejected).Reason)
		return
	}

	page, pageOK := parsePageOrReject(w, r)
	if !pageOK {
		return
	}
	result := server.webhookService.ListDeliveries(r.Context(), owner.owner, id.Value, page.Probe())
	listed, listedMatched := result.(webhook.DeliveriesListed)
	if !listedMatched {
		writeDomainError(w, result.(webhook.ListDeliveriesRejected).Reason)
		return
	}

	visible, nextOffset := probeListWindow(len(listed.Values), page)
	response := webhookDeliveriesResponse{Deliveries: make([]webhookDeliveryResponse, 0, visible), NextOffset: nextOffset}
	for index := range listed.Values[:visible] {
		response.Deliveries = append(response.Deliveries, webhookDeliveryToResponse(listed.Values[index]))
	}
	writeJSON(w, http.StatusOK, response)
}

type webhookAudienceResult interface {
	webhookAudienceResult()
}

type webhookAudienceAccepted struct {
	value webhook.Audience
}

type webhookAudienceRejected struct {
	reason core.DomainError
}

func (webhookAudienceAccepted) webhookAudienceResult() {}

func (webhookAudienceRejected) webhookAudienceResult() {}

// parseWebhookAudience maps the request's audience and marketplace filter
// fields onto a webhook.Audience. An absent audience is the recipient
// default; the filter fields are valid only with the marketplace audience.
// The kinds compatibility rule (marketplace listens only for task_opened) is
// enforced by webhook.Service.Create.
func parseWebhookAudience(request webhookSubscriptionRequest) webhookAudienceResult {
	switch request.Audience {
	case "", "recipient":
		if request.FilterTaskType != "" || request.FilterMinCreditReward != 0 {
			return webhookAudienceRejected{reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "filter_task_type and filter_min_credit_reward require the marketplace audience")}
		}
		return webhookAudienceAccepted{value: webhook.RecipientAudience{}}
	case "marketplace":
		audience := webhook.NewMarketplaceAudience()
		if request.FilterTaskType != "" {
			typeResult := task.ParseTaskType(request.FilterTaskType)
			taskType, matched := typeResult.(task.TaskTypeAccepted)
			if !matched {
				return webhookAudienceRejected{reason: typeResult.(task.TaskTypeRejected).Reason}
			}
			audience.TaskType = webhook.MarketplaceTaskTypeIs{Value: taskType.Value}
		}
		if request.FilterMinCreditReward != 0 {
			rewardResult := webhook.NewMinimumCreditReward(request.FilterMinCreditReward)
			reward, matched := rewardResult.(webhook.MinimumCreditRewardAccepted)
			if !matched {
				return webhookAudienceRejected{reason: rewardResult.(webhook.MinimumCreditRewardRejected).Reason}
			}
			audience.MinReward = reward.Value
		}
		return webhookAudienceAccepted{value: audience}
	default:
		return webhookAudienceRejected{reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "webhook audience must be recipient or marketplace")}
	}
}

// writeWebhookRejection writes the most specific failure: caller rejections
// keep their 401/403 split, and owner-resolution failures carry their own
// status.
func writeWebhookRejection(w http.ResponseWriter, callerResult webhookCallerResult, ownerResult webhookOwnerResult) {
	switch callerResult.(type) {
	case webhookCallerUser, webhookCallerOrgCredential:
		rejected := ownerResult.(webhookOwnerRejected)
		writeError(w, rejected.status, rejected.code, rejected.reason)
	default:
		writeWebhookCallerRejection(w, callerResult)
	}
}

func webhookSubscriptionToResponse(value webhook.Subscription) webhookSubscriptionResponse {
	kinds := value.Kinds.Values()
	rawKinds := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		rawKinds = append(rawKinds, kind.String())
	}
	response := webhookSubscriptionResponse{
		ID:        value.ID.String(),
		URL:       value.URL.String(),
		Kinds:     rawKinds,
		State:     value.State.String(),
		CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano),
		Audience:  "recipient",
	}
	if marketplace, matched := value.Audience.(webhook.MarketplaceAudience); matched {
		response.Audience = "marketplace"
		if typed, filtered := marketplace.TaskType.(webhook.MarketplaceTaskTypeIs); filtered {
			response.FilterTaskType = typed.Value.String()
		}
		if reward, filtered := marketplace.MinReward.(webhook.MinimumCreditReward); filtered {
			response.FilterMinCreditReward = reward.Amount()
		}
	}
	switch owner := value.Owner.(type) {
	case webhook.OwnerUser:
		response.OwnerKind = "user"
		response.OwnerUserID = owner.ID.String()
	case webhook.OwnerOrganization:
		response.OwnerKind = "organization"
		response.OwnerOrganizationID = owner.ID.String()
	}
	return response
}

func webhookDeliveryToResponse(value webhook.Delivery) webhookDeliveryResponse {
	return webhookDeliveryResponse{
		ID:            value.ID.String(),
		EventCursor:   value.EventCursor.String(),
		State:         value.State.String(),
		AttemptCount:  value.AttemptCount,
		NextAttemptAt: value.NextAttemptAt.UTC().Format(time.RFC3339Nano),
		LastStatus:    value.LastStatus,
	}
}
