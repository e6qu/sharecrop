package mcp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
)

type webhookSubscriptionSummary struct {
	ID                  string   `json:"id"`
	OwnerKind           string   `json:"owner_kind"`
	OwnerUserID         string   `json:"owner_user_id"`
	OwnerOrganizationID string   `json:"owner_organization_id"`
	URL                 string   `json:"url"`
	Kinds               []string `json:"kinds"`
	State               string   `json:"state"`
	CreatedAt           string   `json:"created_at"`
	// Audience is "recipient" or "marketplace"; the filter fields are the
	// marketplace narrowing filters (empty / 0 mean no filter, and they are
	// always empty / 0 on recipient subscriptions), matching REST.
	Audience              string `json:"audience"`
	FilterTaskType        string `json:"filter_task_type"`
	FilterMinCreditReward int64  `json:"filter_min_credit_reward"`
}

type webhookSubscriptionCreatedPayload struct {
	Subscription webhookSubscriptionSummary `json:"subscription"`
	// Secret is the signing secret, surfaced exactly once at creation.
	Secret string `json:"secret"`
}

type webhookSubscriptionsPayload struct {
	Subscriptions []webhookSubscriptionSummary `json:"subscriptions"`
	NextOffset    int                          `json:"next_offset"`
}

type webhookDeliverySummary struct {
	ID            string `json:"id"`
	EventCursor   string `json:"event_cursor"`
	State         string `json:"state"`
	AttemptCount  int64  `json:"attempt_count"`
	NextAttemptAt string `json:"next_attempt_at"`
	LastStatus    string `json:"last_status"`
}

type webhookDeliveriesPayload struct {
	Deliveries []webhookDeliverySummary `json:"deliveries"`
	NextOffset int                      `json:"next_offset"`
	// Total counts every row matching the filter, ignoring limit/offset.
	Total int64 `json:"total"`
}

func (webhookSubscriptionSummary) payloadValue() {}

func (webhookSubscriptionCreatedPayload) payloadValue() {}

func (webhookSubscriptionsPayload) payloadValue() {}

func (webhookDeliveriesPayload) payloadValue() {}

func webhookSubscriptionToSummary(value webhook.Subscription) webhookSubscriptionSummary {
	kinds := value.Kinds.Values()
	rawKinds := make([]string, 0, len(kinds))
	for _, kind := range kinds {
		rawKinds = append(rawKinds, kind.String())
	}
	summary := webhookSubscriptionSummary{
		ID:        value.ID.String(),
		URL:       value.URL.String(),
		Kinds:     rawKinds,
		State:     value.State.String(),
		CreatedAt: value.CreatedAt.UTC().Format(time.RFC3339Nano),
		Audience:  "recipient",
	}
	if marketplace, matched := value.Audience.(webhook.MarketplaceAudience); matched {
		summary.Audience = "marketplace"
		if typed, filtered := marketplace.TaskType.(webhook.MarketplaceTaskTypeIs); filtered {
			summary.FilterTaskType = typed.Value.String()
		}
		if reward, filtered := marketplace.MinReward.(webhook.MinimumCreditReward); filtered {
			summary.FilterMinCreditReward = reward.Amount()
		}
	}
	switch owner := value.Owner.(type) {
	case webhook.OwnerUser:
		summary.OwnerKind = "user"
		summary.OwnerUserID = owner.ID.String()
	case webhook.OwnerOrganization:
		summary.OwnerKind = "organization"
		summary.OwnerOrganizationID = owner.ID.String()
	}
	return summary
}

// webhookToolOwnerResult resolves which owner a webhook tool call acts for.
type webhookToolOwnerResult interface {
	webhookToolOwnerResult()
}

type webhookToolOwnerAccepted struct {
	owner webhook.Owner
}

type webhookToolOwnerRejected struct {
	failure toolResult
}

func (webhookToolOwnerAccepted) webhookToolOwnerResult() {}

func (webhookToolOwnerRejected) webhookToolOwnerResult() {}

// resolveWebhookToolOwner mirrors the REST owner rule: a personal agent
// credential acts for its user, or for an organization the user holds
// PermissionManageMembers in when organization_id is given; an organization
// credential always acts for its own organization and may not name a
// different one.
func (server Server) resolveWebhookToolOwner(ctx context.Context, subject auth.Subject, rawOrganizationID string) webhookToolOwnerResult {
	switch typed := subject.(type) {
	case auth.UserSubject:
		if rawOrganizationID == "" {
			return webhookToolOwnerAccepted{owner: webhook.OwnerUser{ID: typed.ID}}
		}
		organizationIDResult := core.ParseOrganizationID(rawOrganizationID)
		organizationID, organizationMatched := organizationIDResult.(core.OrganizationIDCreated)
		if !organizationMatched {
			return webhookToolOwnerRejected{failure: toolProtocolError{code: codeInvalidParams, message: organizationIDResult.(core.OrganizationIDRejected).Reason.Description()}}
		}
		permissionResult := server.services.CheckOrganizationPermission(ctx, organizationID.Value, typed.ID, org.PermissionManageMembers)
		if _, denied := permissionResult.(org.PermissionDenied); denied {
			return webhookToolOwnerRejected{failure: toolFailed{code: core.ErrorCodePermissionDenied, message: "organization webhook management access denied"}}
		}
		return webhookToolOwnerAccepted{owner: webhook.OwnerOrganization{ID: organizationID.Value}}
	case auth.OrgSubject:
		if rawOrganizationID != "" && rawOrganizationID != typed.ID.String() {
			return webhookToolOwnerRejected{failure: toolFailed{code: core.ErrorCodePermissionDenied, message: "organization credential may only manage webhooks for its own organization"}}
		}
		return webhookToolOwnerAccepted{owner: webhook.OwnerOrganization{ID: typed.ID}}
	default:
		return webhookToolOwnerRejected{failure: toolFailed{code: core.ErrorCodePermissionDenied, message: "a personal agent credential or an organization credential is required"}}
	}
}

// parseMCPWebhookAudience maps the tool call's audience and marketplace
// filter arguments onto a webhook.Audience, mirroring REST's
// parseWebhookAudience: an absent audience is the recipient default, and the
// filter fields are valid only with the marketplace audience. The kinds
// compatibility rule (marketplace listens only for task_opened) is enforced
// by webhook.Service.Create.
func parseMCPWebhookAudience(rawAudience string, rawTaskType string, rawMinReward int64) (webhook.Audience, toolResult) {
	switch rawAudience {
	case "", "recipient":
		if rawTaskType != "" || rawMinReward != 0 {
			return nil, toolProtocolError{code: codeInvalidParams, message: "filter_task_type and filter_min_credit_reward require the marketplace audience"}
		}
		return webhook.RecipientAudience{}, nil
	case "marketplace":
		audience := webhook.NewMarketplaceAudience()
		if rawTaskType != "" {
			typeResult := task.ParseTaskType(rawTaskType)
			taskType, matched := typeResult.(task.TaskTypeAccepted)
			if !matched {
				return nil, toolProtocolError{code: codeInvalidParams, message: typeResult.(task.TaskTypeRejected).Reason.Description()}
			}
			audience.TaskType = webhook.MarketplaceTaskTypeIs{Value: taskType.Value}
		}
		if rawMinReward != 0 {
			rewardResult := webhook.NewMinimumCreditReward(rawMinReward)
			reward, matched := rewardResult.(webhook.MinimumCreditRewardAccepted)
			if !matched {
				return nil, toolProtocolError{code: codeInvalidParams, message: rewardResult.(webhook.MinimumCreditRewardRejected).Reason.Description()}
			}
			audience.MinReward = reward.Value
		}
		return audience, nil
	default:
		return nil, toolProtocolError{code: codeInvalidParams, message: "webhook audience must be recipient or marketplace"}
	}
}

func (server Server) callCreateWebhookSubscription(ctx context.Context, subject auth.Subject, credential CallerCredential, arguments json.RawMessage) toolResult {
	var args struct {
		URL                   string   `json:"url"`
		Kinds                 []string `json:"kinds"`
		OrganizationID        string   `json:"organization_id"`
		Audience              string   `json:"audience"`
		FilterTaskType        string   `json:"filter_task_type"`
		FilterMinCreditReward int64    `json:"filter_min_credit_reward"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}

	endpointResult := webhook.NewEndpointURL(args.URL)
	endpoint, endpointMatched := endpointResult.(webhook.EndpointURLAccepted)
	if !endpointMatched {
		return toolProtocolError{code: codeInvalidParams, message: endpointResult.(webhook.EndpointURLRejected).Reason.Description()}
	}

	kinds := make([]event.Kind, 0, len(args.Kinds))
	for _, rawKind := range args.Kinds {
		kindResult := event.ParseKind(rawKind)
		kind, kindMatched := kindResult.(event.KindParsed)
		if !kindMatched {
			return toolProtocolError{code: codeInvalidParams, message: kindResult.(event.KindRejected).Reason.Description()}
		}
		kinds = append(kinds, kind.Value)
	}
	filterResult := webhook.NewKindFilter(kinds)
	filter, filterMatched := filterResult.(webhook.KindFilterAccepted)
	if !filterMatched {
		return toolProtocolError{code: codeInvalidParams, message: filterResult.(webhook.KindFilterRejected).Reason.Description()}
	}

	// Every MCP caller is a credential, so the kind-entitlement rule always
	// applies: subscribing to a kind requires the read scope that could
	// observe it directly, in addition to webhooks_manage.
	if missing := webhook.MissingKindEntitlements(credential.Scopes, filter.Value); len(missing) > 0 {
		return errorResponseToolScopeDenied(missing[0])
	}

	audience, audienceProblem := parseMCPWebhookAudience(args.Audience, args.FilterTaskType, args.FilterMinCreditReward)
	if audienceProblem != nil {
		return audienceProblem
	}

	ownerResult := server.resolveWebhookToolOwner(ctx, subject, args.OrganizationID)
	owner, ownerMatched := ownerResult.(webhookToolOwnerAccepted)
	if !ownerMatched {
		return ownerResult.(webhookToolOwnerRejected).failure
	}

	result := server.services.CreateWebhookSubscription(ctx, owner.owner, endpoint.Value, filter.Value, audience)
	created, createdMatched := result.(webhook.SubscriptionCreated)
	if !createdMatched {
		return toolFailed{code: result.(webhook.CreateRejected).Reason.Code(), message: result.(webhook.CreateRejected).Reason.Description()}
	}
	return marshalPayload(webhookSubscriptionCreatedPayload{
		Subscription: webhookSubscriptionToSummary(created.Value),
		Secret:       created.Secret.String(),
	})
}

// errorResponseToolScopeDenied names the first missing entitlement so the
// caller knows exactly which scope to mint the credential with.
func errorResponseToolScopeDenied(kind event.Kind) toolResult {
	return toolProtocolError{code: codeScopeDenied, message: "credential lacks the " + webhook.RequiredScopeForKind(kind).String() + " scope required to subscribe to " + kind.String()}
}

func (server Server) callListWebhookSubscriptions(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	var args struct {
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	ownerResult := server.resolveWebhookToolOwner(ctx, subject, args.OrganizationID)
	owner, ownerMatched := ownerResult.(webhookToolOwnerAccepted)
	if !ownerMatched {
		return ownerResult.(webhookToolOwnerRejected).failure
	}

	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListWebhookSubscriptions(ctx, owner.owner, page.Probe())
	listed, listedMatched := result.(webhook.SubscriptionsListed)
	if !listedMatched {
		return toolFailed{code: result.(webhook.ListRejected).Reason.Code(), message: result.(webhook.ListRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]webhookSubscriptionSummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, webhookSubscriptionToSummary(listed.Values[index]))
	}
	return marshalPayload(webhookSubscriptionsPayload{Subscriptions: summaries, NextOffset: nextOffset})
}

// webhookToolTargetResult resolves the shared (subscription_id,
// organization_id) argument pair of the per-subscription tools into the
// target id and acting owner.
type webhookToolTargetResult interface {
	webhookToolTargetResult()
}

type webhookToolTargetAccepted struct {
	id    core.WebhookSubscriptionID
	owner webhook.Owner
}

type webhookToolTargetRejected struct {
	failure toolResult
}

func (webhookToolTargetAccepted) webhookToolTargetResult() {}

func (webhookToolTargetRejected) webhookToolTargetResult() {}

func (server Server) resolveWebhookToolTarget(ctx context.Context, subject auth.Subject, arguments json.RawMessage) webhookToolTargetResult {
	var args struct {
		SubscriptionID string `json:"subscription_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return webhookToolTargetRejected{failure: invalidArguments()}
	}
	idResult := core.ParseWebhookSubscriptionID(args.SubscriptionID)
	id, idMatched := idResult.(core.WebhookSubscriptionIDCreated)
	if !idMatched {
		return webhookToolTargetRejected{failure: toolProtocolError{code: codeInvalidParams, message: idResult.(core.WebhookSubscriptionIDRejected).Reason.Description()}}
	}
	ownerResult := server.resolveWebhookToolOwner(ctx, subject, args.OrganizationID)
	owner, ownerMatched := ownerResult.(webhookToolOwnerAccepted)
	if !ownerMatched {
		return webhookToolTargetRejected{failure: ownerResult.(webhookToolOwnerRejected).failure}
	}
	return webhookToolTargetAccepted{id: id.Value, owner: owner.owner}
}

func (server Server) callRevokeWebhookSubscription(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	targetResult := server.resolveWebhookToolTarget(ctx, subject, arguments)
	target, targetMatched := targetResult.(webhookToolTargetAccepted)
	if !targetMatched {
		return targetResult.(webhookToolTargetRejected).failure
	}

	result := server.services.RevokeWebhookSubscription(ctx, target.owner, target.id)
	revoked, revokedMatched := result.(webhook.SubscriptionRevoked)
	if !revokedMatched {
		return toolFailed{code: result.(webhook.RevokeRejected).Reason.Code(), message: result.(webhook.RevokeRejected).Reason.Description()}
	}
	return marshalPayload(webhookSubscriptionToSummary(revoked.Value))
}

func (server Server) callListWebhookDeliveries(ctx context.Context, subject auth.Subject, arguments json.RawMessage) toolResult {
	targetResult := server.resolveWebhookToolTarget(ctx, subject, arguments)
	target, targetMatched := targetResult.(webhookToolTargetAccepted)
	if !targetMatched {
		return targetResult.(webhookToolTargetRejected).failure
	}

	page, pageProblem := parseMCPPageArguments(arguments)
	if pageProblem != nil {
		return pageProblem
	}
	result := server.services.ListWebhookDeliveries(ctx, target.owner, target.id, page.Probe())
	listed, listedMatched := result.(webhook.DeliveriesListed)
	if !listedMatched {
		return toolFailed{code: result.(webhook.ListDeliveriesRejected).Reason.Code(), message: result.(webhook.ListDeliveriesRejected).Reason.Description()}
	}
	visible, nextOffset := core.ProbeListWindow(len(listed.Values), page)
	summaries := make([]webhookDeliverySummary, 0, visible)
	for index := range listed.Values[:visible] {
		summaries = append(summaries, webhookDeliverySummary{
			ID:            listed.Values[index].ID.String(),
			EventCursor:   listed.Values[index].EventCursor.String(),
			State:         listed.Values[index].State.String(),
			AttemptCount:  listed.Values[index].AttemptCount,
			NextAttemptAt: listed.Values[index].NextAttemptAt.UTC().Format(time.RFC3339Nano),
			LastStatus:    listed.Values[index].LastStatus,
		})
	}
	return marshalPayload(webhookDeliveriesPayload{Deliveries: summaries, NextOffset: nextOffset, Total: listed.Total})
}
