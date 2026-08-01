package db

import (
	"context"
	"errors"
	"time"

	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WebhookStore persists webhook subscriptions and deliveries. The bridged
// webhook.Store surface (create/list/revoke/list-deliveries) runs everywhere
// the app mux runs; the claim/mark methods further down are host-only (used
// by internal/webhookdispatch) and deliberately NOT part of the bridged
// interface, so the WASI guest can never reach them. Delivery rows are
// created by the event dispatch step (expandWebhookDeliveries, called from
// db.EventStore.Dispatch).
//
// The signing secret is stored AS WRITTEN, never hashed: the dispatcher must
// compute an HMAC-SHA256 over each delivery body with the original secret,
// which is impossible from a hash.
type WebhookStore struct {
	db Beginner
}

func NewWebhookStore(pool *pgxpool.Pool) WebhookStore {
	return NewWebhookStoreFromHandle(NewPGX(pool))
}

func NewWebhookStoreFromHandle(handle Beginner) WebhookStore {
	return WebhookStore{db: handle}
}

func webhookOwnerColumns(owner webhook.Owner) (kind string, userID *string, organizationID *string) {
	switch typed := owner.(type) {
	case webhook.OwnerUser:
		value := typed.ID.String()
		return "user", &value, nil
	case webhook.OwnerOrganization:
		value := typed.ID.String()
		return "organization", nil, &value
	default:
		return "", nil, nil
	}
}

// webhookAudienceColumns flattens the audience union into its three columns.
func webhookAudienceColumns(audience webhook.Audience) (kind string, taskType *string, minReward *int64) {
	marketplace, matched := audience.(webhook.MarketplaceAudience)
	if !matched {
		return "recipient", nil, nil
	}
	if typed, typedMatched := marketplace.TaskType.(webhook.MarketplaceTaskTypeIs); typedMatched {
		value := typed.Value.String()
		taskType = &value
	}
	if reward, rewardMatched := marketplace.MinReward.(webhook.MinimumCreditReward); rewardMatched {
		value := reward.Amount()
		minReward = &value
	}
	return "marketplace", taskType, minReward
}

func (store WebhookStore) CreateSubscription(ctx context.Context, subscription webhook.Subscription, secret webhook.Secret) webhook.CreateStoreResult {
	ownerKind, ownerUserID, ownerOrganizationID := webhookOwnerColumns(subscription.Owner)
	if ownerKind == "" {
		return webhook.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook subscription owner is invalid")}
	}
	audienceKind, filterTaskType, filterMinReward := webhookAudienceColumns(subscription.Audience)

	tx, err := store.db.Begin(ctx)
	if err != nil {
		return webhook.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin create webhook subscription transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	_, err = tx.Exec(ctx, `
		insert into webhook_subscriptions (id, owner_kind, owner_user_id, owner_organization_id, url, secret, state, created_at, audience, filter_task_type, filter_min_credit_reward)
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, subscription.ID.String(), ownerKind, ownerUserID, ownerOrganizationID, subscription.URL.String(), secret.String(), subscription.State.String(), subscription.CreatedAt, audienceKind, filterTaskType, filterMinReward)
	if err != nil {
		return webhook.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert webhook subscription failed")}
	}

	for _, kind := range subscription.Kinds.Values() {
		if _, err := tx.Exec(ctx, "insert into webhook_subscription_kinds (subscription_id, kind) values ($1, $2)", subscription.ID.String(), kind.String()); err != nil {
			return webhook.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "insert webhook subscription kind failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return webhook.CreateStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit create webhook subscription transaction failed")}
	}
	return webhook.CreateStoreAccepted{}
}

func webhookSubscriptionSelectSQL() string {
	return `
		select webhook_subscriptions.id::text, webhook_subscriptions.owner_kind,
			webhook_subscriptions.owner_user_id::text, webhook_subscriptions.owner_organization_id::text,
			webhook_subscriptions.url, webhook_subscriptions.state, webhook_subscriptions.created_at,
			webhook_subscriptions.audience,
			coalesce(webhook_subscriptions.filter_task_type, ''),
			coalesce(webhook_subscriptions.filter_min_credit_reward, 0),
			coalesce(array_remove(array_agg(webhook_subscription_kinds.kind), null), '{}')::text as kinds
		from webhook_subscriptions
		left join webhook_subscription_kinds on webhook_subscription_kinds.subscription_id = webhook_subscriptions.id
	`
}

// webhookOwnerPredicate matches subscriptions owned by the given owner. The
// two placeholders are the owner kind and the owner id.
const webhookOwnerPredicate = `
	webhook_subscriptions.owner_kind = $1
	and coalesce(webhook_subscriptions.owner_user_id::text, webhook_subscriptions.owner_organization_id::text) = $2
`

func webhookOwnerIdentity(owner webhook.Owner) (kind string, id string) {
	switch typed := owner.(type) {
	case webhook.OwnerUser:
		return "user", typed.ID.String()
	case webhook.OwnerOrganization:
		return "organization", typed.ID.String()
	default:
		return "", ""
	}
}

func (store WebhookStore) ListSubscriptions(ctx context.Context, owner webhook.Owner, page core.Page) webhook.ListStoreResult {
	ownerKind, ownerID := webhookOwnerIdentity(owner)
	rows, err := store.db.Query(ctx, webhookSubscriptionSelectSQL()+`
		where `+webhookOwnerPredicate+`
		group by webhook_subscriptions.id
		order by webhook_subscriptions.created_at, webhook_subscriptions.id
		limit $3 offset $4
	`, ownerKind, ownerID, page.Limit(), page.Offset())
	if err != nil {
		return webhook.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list webhook subscriptions failed")}
	}
	defer rows.Close()

	values := make([]webhook.Subscription, 0)
	for rows.Next() {
		parsed, parseErr := scanWebhookSubscription(rows)
		if parseErr != nil {
			return webhook.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan webhook subscription failed")}
		}
		values = append(values, parsed)
	}
	if rows.Err() != nil {
		return webhook.ListStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read webhook subscriptions failed")}
	}
	return webhook.ListStoreListed{Values: values}
}

func (store WebhookStore) RevokeSubscription(ctx context.Context, owner webhook.Owner, id core.WebhookSubscriptionID) webhook.RevokeStoreResult {
	ownerKind, ownerID := webhookOwnerIdentity(owner)
	tag, err := store.db.Exec(ctx, `
		update webhook_subscriptions
		set state = 'revoked', state_recorded_at = now()
		where id = $3 and state = 'active' and `+webhookOwnerPredicate+`
	`, ownerKind, ownerID, id.String())
	if err != nil {
		return webhook.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "revoke webhook subscription failed")}
	}
	if tag == 0 {
		return webhook.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "active webhook subscription was not found")}
	}

	rows, err := store.db.Query(ctx, webhookSubscriptionSelectSQL()+`
		where webhook_subscriptions.id = $1
		group by webhook_subscriptions.id
	`, id.String())
	if err != nil {
		return webhook.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read revoked webhook subscription failed")}
	}
	defer rows.Close()
	if !rows.Next() {
		return webhook.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read revoked webhook subscription failed")}
	}
	parsed, parseErr := scanWebhookSubscription(rows)
	if parseErr != nil {
		return webhook.RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan revoked webhook subscription failed")}
	}
	return webhook.RevokeStoreRevoked{Value: parsed}
}

func (store WebhookStore) ListDeliveries(ctx context.Context, owner webhook.Owner, id core.WebhookSubscriptionID, page core.Page) webhook.ListDeliveriesStoreResult {
	ownerKind, ownerID := webhookOwnerIdentity(owner)
	var owned int64
	err := store.db.QueryRow(ctx, `
		select count(*) from webhook_subscriptions
		where id = $3 and `+webhookOwnerPredicate+`
	`, ownerKind, ownerID, id.String()).Scan(&owned)
	if err != nil {
		return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read webhook subscription failed")}
	}
	if owned == 0 {
		return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeNotFound, "webhook subscription was not found")}
	}

	var total int64
	if err := store.db.QueryRow(ctx,
		"select count(*) from webhook_deliveries where subscription_id = $1",
		id.String()).Scan(&total); err != nil {
		return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "count webhook deliveries failed")}
	}

	rows, err := store.db.Query(ctx, `
		select id::text, event_seq, state, attempt_count, next_attempt_at, last_status
		from webhook_deliveries
		where subscription_id = $1
		order by event_seq, id
		limit $2 offset $3
	`, id.String(), page.Limit(), page.Offset())
	if err != nil {
		return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "list webhook deliveries failed")}
	}
	defer rows.Close()

	values := make([]webhook.Delivery, 0)
	for rows.Next() {
		var rawID, rawState, lastStatus string
		var eventSeq, attemptCount int64
		var nextAttemptAt time.Time
		if err := rows.Scan(&rawID, &eventSeq, &rawState, &attemptCount, &nextAttemptAt, &lastStatus); err != nil {
			return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan webhook delivery failed")}
		}
		idResult, idMatched := core.ParseWebhookDeliveryID(rawID).(core.WebhookDeliveryIDCreated)
		if !idMatched {
			return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "webhook delivery id is invalid")}
		}
		stateResult, stateMatched := webhook.ParseDeliveryState(rawState).(webhook.DeliveryStateAccepted)
		if !stateMatched {
			return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "webhook delivery state is invalid")}
		}
		values = append(values, webhook.Delivery{
			ID:            idResult.Value,
			EventCursor:   event.CursorFromSequence(eventSeq),
			State:         stateResult.Value,
			AttemptCount:  attemptCount,
			NextAttemptAt: nextAttemptAt,
			LastStatus:    lastStatus,
		})
	}
	if rows.Err() != nil {
		return webhook.ListDeliveriesStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "read webhook deliveries failed")}
	}
	return webhook.ListDeliveriesStoreListed{Values: values, Total: total}
}

// scanWebhookSubscription decodes one row selected with
// webhookSubscriptionSelectSQL.
func scanWebhookSubscription(rows Rows) (webhook.Subscription, error) {
	var rawID, ownerKind string
	var rawOwnerUser, rawOwnerOrganization *string
	var rawURL, rawState string
	var createdAt time.Time
	var rawAudience, rawFilterTaskType string
	var rawFilterMinReward int64
	var rawKinds StringArray
	if err := rows.Scan(&rawID, &ownerKind, &rawOwnerUser, &rawOwnerOrganization, &rawURL, &rawState, &createdAt, &rawAudience, &rawFilterTaskType, &rawFilterMinReward, &rawKinds); err != nil {
		return webhook.Subscription{}, err
	}

	idResult, idMatched := core.ParseWebhookSubscriptionID(rawID).(core.WebhookSubscriptionIDCreated)
	if !idMatched {
		return webhook.Subscription{}, ErrNoRows
	}

	var owner webhook.Owner
	switch {
	case ownerKind == "user" && rawOwnerUser != nil:
		parsed, matched := core.ParseUserID(*rawOwnerUser).(core.UserIDCreated)
		if !matched {
			return webhook.Subscription{}, ErrNoRows
		}
		owner = webhook.OwnerUser{ID: parsed.Value}
	case ownerKind == "organization" && rawOwnerOrganization != nil:
		parsed, matched := core.ParseOrganizationID(*rawOwnerOrganization).(core.OrganizationIDCreated)
		if !matched {
			return webhook.Subscription{}, ErrNoRows
		}
		owner = webhook.OwnerOrganization{ID: parsed.Value}
	default:
		return webhook.Subscription{}, ErrNoRows
	}

	urlResult, urlMatched := webhook.NewEndpointURL(rawURL).(webhook.EndpointURLAccepted)
	if !urlMatched {
		return webhook.Subscription{}, ErrNoRows
	}
	stateResult, stateMatched := webhook.ParseState(rawState).(webhook.StateAccepted)
	if !stateMatched {
		return webhook.Subscription{}, ErrNoRows
	}

	kinds := make([]event.Kind, 0, len(rawKinds))
	for _, rawKind := range rawKinds {
		parsed, matched := event.ParseKind(rawKind).(event.KindParsed)
		if !matched {
			return webhook.Subscription{}, ErrNoRows
		}
		kinds = append(kinds, parsed.Value)
	}
	filterResult, filterMatched := webhook.NewKindFilter(kinds).(webhook.KindFilterAccepted)
	if !filterMatched {
		return webhook.Subscription{}, ErrNoRows
	}

	audience, audienceErr := parseWebhookAudience(rawAudience, rawFilterTaskType, rawFilterMinReward)
	if audienceErr != nil {
		return webhook.Subscription{}, audienceErr
	}

	return webhook.Subscription{
		ID:        idResult.Value,
		Owner:     owner,
		URL:       urlResult.Value,
		Kinds:     filterResult.Value,
		Audience:  audience,
		State:     stateResult.Value,
		CreatedAt: createdAt,
	}, nil
}

// parseWebhookAudience rebuilds the audience union from its three columns.
func parseWebhookAudience(rawAudience string, rawTaskType string, rawMinReward int64) (webhook.Audience, error) {
	switch rawAudience {
	case "recipient":
		return webhook.RecipientAudience{}, nil
	case "marketplace":
		audience := webhook.NewMarketplaceAudience()
		if rawTaskType != "" {
			parsed, matched := task.ParseTaskType(rawTaskType).(task.TaskTypeAccepted)
			if !matched {
				return nil, ErrNoRows
			}
			audience.TaskType = webhook.MarketplaceTaskTypeIs{Value: parsed.Value}
		}
		if rawMinReward != 0 {
			reward, matched := webhook.NewMinimumCreditReward(rawMinReward).(webhook.MinimumCreditRewardAccepted)
			if !matched {
				return nil, ErrNoRows
			}
			audience.MinReward = reward.Value
		}
		return audience, nil
	default:
		return nil, ErrNoRows
	}
}

// ---- host-only claim/mark methods (internal/webhookdispatch) ----

// expandWebhookDeliveries inserts the pending delivery rows for one event.
// It runs inside the event dispatch transaction (db.EventStore.Dispatch):
// delivery expansion is a dispatch effect of each recorded event, executed
// inline after the mutation commits and replayed by the lifecycle runner's
// dispatch sweep after a crash.
// The matching happens in SQL so the pump never loads subscription or
// recipient rows: a subscription matches when it is active, its kind filter
// contains the event's kind, and its owner can see the event (user owners
// through domain_event_recipients, organization owners through the event's
// organization_id).
func expandWebhookDeliveries(ctx context.Context, tx Tx, sequence int64) (int, error) {
	inserted := 0
	rows, err := tx.Query(ctx, `
		select webhook_subscriptions.id::text
		from webhook_subscriptions
		join webhook_subscription_kinds
			on webhook_subscription_kinds.subscription_id = webhook_subscriptions.id
		join domain_events on domain_events.seq = $1
		where webhook_subscriptions.state = 'active'
			and webhook_subscription_kinds.kind = domain_events.kind
			and (
				(webhook_subscriptions.audience = 'recipient' and (
					(webhook_subscriptions.owner_kind = 'user' and exists (
						select 1 from domain_event_recipients
						where domain_event_recipients.event_seq = domain_events.seq
							and domain_event_recipients.user_id = webhook_subscriptions.owner_user_id
					))
					or (webhook_subscriptions.owner_kind = 'organization'
						and domain_events.organization_id = webhook_subscriptions.owner_organization_id)
				))
				or (webhook_subscriptions.audience = 'marketplace'
					and domain_events.kind = 'task_opened'
					and exists (
						select 1 from tasks
						join task_visibility_scopes on task_visibility_scopes.task_id = tasks.id
						where tasks.id = domain_events.task_id
							and tasks.state = 'open'
							and task_visibility_scopes.visibility_kind = 'public'
							and (webhook_subscriptions.filter_task_type is null
								or tasks.task_type = webhook_subscriptions.filter_task_type)
							and (webhook_subscriptions.filter_min_credit_reward is null
								or coalesce(tasks.reward_credit_amount, 0) >= webhook_subscriptions.filter_min_credit_reward)
					))
			)
	`, sequence)
	if err != nil {
		return 0, err
	}
	subscriptionIDs := make([]string, 0)
	for rows.Next() {
		var subscriptionID string
		if err := rows.Scan(&subscriptionID); err != nil {
			rows.Close()
			return 0, err
		}
		subscriptionIDs = append(subscriptionIDs, subscriptionID)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return 0, rowsErr
	}

	for _, subscriptionID := range subscriptionIDs {
		idResult, idMatched := core.NewWebhookDeliveryID().(core.WebhookDeliveryIDCreated)
		if !idMatched {
			return inserted, errors.New("webhook delivery id generation failed")
		}
		count, err := tx.Exec(ctx, `
			insert into webhook_deliveries (id, subscription_id, event_seq, state, attempt_count, next_attempt_at)
			values ($1, $2, $3, 'pending', 0, now())
			on conflict (subscription_id, event_seq) do nothing
		`, idResult.Value.String(), subscriptionID, sequence)
		if err != nil {
			return inserted, err
		}
		inserted += int(count)
	}
	return inserted, nil
}

// ClaimedWebhookDelivery is one due delivery locked for a dispatch attempt,
// joined with everything the dispatcher needs: where to POST, what secret to
// sign with, and the full stored event for the body.
type ClaimedWebhookDelivery struct {
	ID           core.WebhookDeliveryID
	Subscription core.WebhookSubscriptionID
	URL          webhook.EndpointURL
	Secret       webhook.Secret
	AttemptCount int64
	Event        event.StoredEvent
}

type ClaimDueDeliveriesResult interface {
	claimDueDeliveriesResult()
}

type ClaimDueDeliveriesListed struct {
	Values []ClaimedWebhookDelivery
}

type ClaimDueDeliveriesRejected struct {
	Reason core.DomainError
}

func (ClaimDueDeliveriesListed) claimDueDeliveriesResult() {}

func (ClaimDueDeliveriesRejected) claimDueDeliveriesResult() {}

// ClaimDueDeliveries locks up to limit due pending deliveries with
// FOR UPDATE SKIP LOCKED so concurrent dispatchers never claim the same row,
// and pushes their next_attempt_at forward by the caller-supplied hold so
// another replica cannot re-claim a row while a slow batch is still working
// through it. The hold must cover the dispatcher's worst-case batch time
// (the dispatcher derives it from its own constants); after a crash the
// stranded rows become claimable again once the hold lapses. Delivery is
// therefore at-least-once: a dispatcher that crashes after a successful POST
// but before MarkDelivered leaves the row pending, and a later claim POSTs
// it again — receivers must dedupe on the delivery id header. The due
// predicate uses SQL now(): the claim horizon is database time, while the
// dispatcher computes retry backoff from its own injected clock.
func (store WebhookStore) ClaimDueDeliveries(ctx context.Context, limit int, hold time.Duration) ClaimDueDeliveriesResult {
	tx, err := store.db.Begin(ctx)
	if err != nil {
		return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "begin webhook claim transaction failed")}
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The delivery body is built from this row, so the claim resolves the
	// read-time enrichment the feed read also carries: the acting user's
	// display name and the referenced task's title.
	rows, err := tx.Query(ctx, `
		select webhook_deliveries.id::text, webhook_deliveries.attempt_count,
			webhook_subscriptions.id::text, webhook_subscriptions.url, webhook_subscriptions.secret,
			`+eventColumns+`,
			`+displayNameSQL("actor_user")+`,
			coalesce(subject_task.title, '')
		from webhook_deliveries
		join webhook_subscriptions on webhook_subscriptions.id = webhook_deliveries.subscription_id
		join domain_events on domain_events.seq = webhook_deliveries.event_seq
		join users as actor_user on actor_user.id = domain_events.actor_user_id
		left join tasks as subject_task on subject_task.id = domain_events.task_id
		where webhook_deliveries.state = 'pending' and webhook_deliveries.next_attempt_at <= now()
		order by webhook_deliveries.next_attempt_at
		limit $1
		for update of webhook_deliveries skip locked
	`, limit)
	if err != nil {
		return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "claim webhook deliveries failed")}
	}

	values := make([]ClaimedWebhookDelivery, 0)
	for rows.Next() {
		claimed, scanErr := scanClaimedWebhookDelivery(rows)
		if scanErr != nil {
			rows.Close()
			return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "scan claimed webhook delivery failed")}
		}
		values = append(values, claimed)
	}
	rowsErr := rows.Err()
	rows.Close()
	if rowsErr != nil {
		return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "iterate claimed webhook deliveries failed")}
	}

	// Push the claimed rows out of the due window for the full hold so they
	// are not reclaimed while this dispatcher is still working through the
	// batch; the dispatcher marks each one delivered or failed with a real
	// backoff long before the hold lapses.
	for _, claimed := range values {
		if _, err := tx.Exec(ctx, `
			update webhook_deliveries set next_attempt_at = now() + make_interval(secs => $2)
			where id = $1
		`, claimed.ID.String(), hold.Seconds()); err != nil {
			return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "hold claimed webhook delivery failed")}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return ClaimDueDeliveriesRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "commit webhook claim transaction failed")}
	}
	return ClaimDueDeliveriesListed{Values: values}
}

func scanClaimedWebhookDelivery(rows Rows) (ClaimedWebhookDelivery, error) {
	var rawDeliveryID, rawSubscriptionID, rawURL, rawSecret string
	var attemptCount int64
	var sequence int64
	var rawEventID, rawKind, actorKind, rawActorUser string
	var rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible *string
	var rawMetadata string
	var occurredAt time.Time
	var rawActorName, rawTaskTitle string
	if err := rows.Scan(&rawDeliveryID, &attemptCount, &rawSubscriptionID, &rawURL, &rawSecret,
		&sequence, &rawEventID, &rawKind, &actorKind, &rawActorUser,
		&rawTask, &rawSubmission, &rawReservation, &rawSeries, &rawOrganization, &rawCollectible,
		&rawMetadata, &occurredAt, &rawActorName, &rawTaskTitle); err != nil {
		return ClaimedWebhookDelivery{}, err
	}

	deliveryID, deliveryMatched := core.ParseWebhookDeliveryID(rawDeliveryID).(core.WebhookDeliveryIDCreated)
	if !deliveryMatched {
		return ClaimedWebhookDelivery{}, ErrNoRows
	}
	subscriptionID, subscriptionMatched := core.ParseWebhookSubscriptionID(rawSubscriptionID).(core.WebhookSubscriptionIDCreated)
	if !subscriptionMatched {
		return ClaimedWebhookDelivery{}, ErrNoRows
	}
	endpoint, endpointMatched := webhook.NewEndpointURL(rawURL).(webhook.EndpointURLAccepted)
	if !endpointMatched {
		return ClaimedWebhookDelivery{}, ErrNoRows
	}
	secret, secretMatched := webhook.ParseSecret(rawSecret).(webhook.SecretAccepted)
	if !secretMatched {
		return ClaimedWebhookDelivery{}, ErrNoRows
	}
	stored, storedErr := parseStoredEventColumns(sequence, rawEventID, rawKind, actorKind, rawActorUser,
		rawTask, rawSubmission, rawReservation, rawSeries, rawOrganization, rawCollectible, rawMetadata, occurredAt)
	if storedErr != nil {
		return ClaimedWebhookDelivery{}, storedErr
	}
	nameResult, nameMatched := auth.NewDisplayName(rawActorName).(auth.DisplayNameAccepted)
	if !nameMatched {
		return ClaimedWebhookDelivery{}, ErrNoRows
	}
	stored.ActorName = event.ActorNamed{DisplayName: nameResult.Value}
	if rawTaskTitle != "" {
		stored.TaskTitle = event.TaskTitled{Title: rawTaskTitle}
	}

	return ClaimedWebhookDelivery{
		ID:           deliveryID.Value,
		Subscription: subscriptionID.Value,
		URL:          endpoint.Value,
		Secret:       secret.Value,
		AttemptCount: attemptCount,
		Event:        stored,
	}, nil
}

type MarkDeliveryResult interface {
	markDeliveryResult()
}

type MarkDeliveryRecorded struct{}

type MarkDeliveryRejected struct {
	Reason core.DomainError
}

func (MarkDeliveryRecorded) markDeliveryResult() {}

func (MarkDeliveryRejected) markDeliveryResult() {}

// MarkDelivered records a successful attempt: the delivery leaves the
// pending pool for good.
func (store WebhookStore) MarkDelivered(ctx context.Context, id core.WebhookDeliveryID, attemptCount int64, status string) MarkDeliveryResult {
	tag, err := store.db.Exec(ctx, `
		update webhook_deliveries
		set state = 'delivered', attempt_count = $2, last_status = $3, state_recorded_at = now()
		where id = $1 and state = 'pending'
	`, id.String(), attemptCount, status)
	if err != nil {
		return MarkDeliveryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark webhook delivery delivered failed")}
	}
	if tag == 0 {
		return MarkDeliveryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "pending webhook delivery was not found")}
	}
	return MarkDeliveryRecorded{}
}

// FailureOutcome says where a failed attempt leaves the delivery: scheduled
// for a later retry, or dead after the final attempt.
type FailureOutcome interface {
	failureOutcome()
}

type RetryLater struct {
	NextAttemptAt time.Time
}

type Dead struct{}

func (RetryLater) failureOutcome() {}

func (Dead) failureOutcome() {}

// MarkFailed records a failed attempt. For RetryLater the caller supplies
// next_attempt_at explicitly (the dispatcher computes it from its injected
// clock and backoff schedule); Dead ends the walk.
func (store WebhookStore) MarkFailed(ctx context.Context, id core.WebhookDeliveryID, attemptCount int64, status string, outcome FailureOutcome) MarkDeliveryResult {
	var tag int64
	var err error
	switch typed := outcome.(type) {
	case RetryLater:
		tag, err = store.db.Exec(ctx, `
			update webhook_deliveries
			set attempt_count = $2, last_status = $3, next_attempt_at = $4, state_recorded_at = now()
			where id = $1 and state = 'pending'
		`, id.String(), attemptCount, status, typed.NextAttemptAt)
	case Dead:
		tag, err = store.db.Exec(ctx, `
			update webhook_deliveries
			set state = 'dead', attempt_count = $2, last_status = $3, state_recorded_at = now()
			where id = $1 and state = 'pending'
		`, id.String(), attemptCount, status)
	default:
		return MarkDeliveryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "webhook failure outcome is invalid")}
	}
	if err != nil {
		return MarkDeliveryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "mark webhook delivery failed attempt failed")}
	}
	if tag == 0 {
		return MarkDeliveryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "pending webhook delivery was not found")}
	}
	return MarkDeliveryRecorded{}
}
