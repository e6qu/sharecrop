//go:build integration

package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/jackc/pgx/v5/pgxpool"
)

// drainWebhookPump advances the shared fan-out cursor past every event
// already in the database, so a test's subscriptions only see the events the
// test itself emits afterwards.
func drainWebhookPump(t *testing.T, store db.WebhookStore) {
	t.Helper()
	for {
		result := store.PumpEvents(context.Background())
		completed, matched := result.(db.PumpEventsCompleted)
		if !matched {
			t.Fatalf("pump rejected: %#v", result)
		}
		if completed.ExpandedEvents == 0 {
			return
		}
	}
}

func appendWebhookTestEvent(t *testing.T, pool *pgxpool.Pool, kind event.Kind, actor core.UserID, recipients []core.UserID, organization event.OrganizationRef) event.StoredEvent {
	t.Helper()
	id, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("domain event id rejected")
	}
	subject := event.NoSubjectRefs()
	subject.Organization = organization
	appended, appendMatched := db.NewEventStore(pool).Append(context.Background(), event.Event{
		ID:         id.Value,
		Kind:       kind,
		Actor:      event.ActorUser{ID: actor},
		Subject:    subject,
		Metadata:   event.EmptyMetadata(),
		OccurredAt: time.Now().UTC(),
	}, event.NewRecipients(recipients...)).(event.AppendStoreAccepted)
	if !appendMatched {
		t.Fatalf("append event rejected")
	}
	return appended.Value
}

func createWebhookTestSubscription(t *testing.T, store db.WebhookStore, owner webhook.Owner, kinds ...event.Kind) (webhook.Subscription, webhook.Secret) {
	t.Helper()
	created, matched := webhook.NewService(store).Create(context.Background(), owner,
		webhook.NewEndpointURL("https://receiver.invalid/hooks").(webhook.EndpointURLAccepted).Value,
		webhook.NewKindFilter(kinds).(webhook.KindFilterAccepted).Value,
		webhook.RecipientAudience{},
	).(webhook.SubscriptionCreated)
	if !matched {
		t.Fatalf("create subscription rejected")
	}
	return created.Value, created.Secret
}

func countDeliveries(t *testing.T, pool *pgxpool.Pool, subscriptionID core.WebhookSubscriptionID) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `select count(*) from webhook_deliveries where subscription_id = $1`, subscriptionID.String()).Scan(&count); err != nil {
		t.Fatalf("count deliveries: %v", err)
	}
	return count
}

// TestWebhookPumpFanOut proves the pump inserts exactly the right pending
// rows: kind filter AND owner visibility both gate, revoked subscriptions
// never match, and a second pump over the same events inserts nothing.
func TestWebhookPumpFanOut(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)

	recipient := createUser(t, pool, "webhook-recipient")
	bystander := createUser(t, pool, "webhook-bystander")
	organization := createOrganization(t, pool, "webhook-org")
	otherOrganization := createOrganization(t, pool, "webhook-other-org")

	matchingUserSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTaskOpened)
	wrongKindSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTipReceived)
	bystanderSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: bystander}, event.KindTaskOpened)
	matchingOrgSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerOrganization{ID: organization}, event.KindTaskOpened)
	otherOrgSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerOrganization{ID: otherOrganization}, event.KindTaskOpened)
	revokedSub, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTaskOpened)
	if _, matched := store.RevokeSubscription(context.Background(), webhook.OwnerUser{ID: recipient}, revokedSub.ID).(webhook.RevokeStoreRevoked); !matched {
		t.Fatalf("revoke rejected")
	}

	appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.OrganizationSubject{ID: organization})

	pumped, matched := store.PumpEvents(context.Background()).(db.PumpEventsCompleted)
	if !matched {
		t.Fatalf("pump rejected")
	}
	if pumped.ExpandedEvents != 1 || pumped.InsertedDeliveries != 2 {
		t.Fatalf("pump expanded %d events with %d deliveries, want 1 and 2", pumped.ExpandedEvents, pumped.InsertedDeliveries)
	}
	if count := countDeliveries(t, pool, matchingUserSub.ID); count != 1 {
		t.Fatalf("matching user subscription has %d deliveries, want 1", count)
	}
	if count := countDeliveries(t, pool, matchingOrgSub.ID); count != 1 {
		t.Fatalf("matching org subscription has %d deliveries, want 1", count)
	}
	for name, subscription := range map[string]webhook.Subscription{
		"wrong kind": wrongKindSub, "non-recipient": bystanderSub, "other org": otherOrgSub, "revoked": revokedSub,
	} {
		if count := countDeliveries(t, pool, subscription.ID); count != 0 {
			t.Fatalf("%s subscription has %d deliveries, want 0", name, count)
		}
	}

	// Re-running the pump over the same stream inserts nothing new.
	repumped, repumpMatched := store.PumpEvents(context.Background()).(db.PumpEventsCompleted)
	if !repumpMatched || repumped.ExpandedEvents != 0 || repumped.InsertedDeliveries != 0 {
		t.Fatalf("second pump = %#v, want a no-op", repumped)
	}
	if count := countDeliveries(t, pool, matchingUserSub.ID); count != 1 {
		t.Fatalf("second pump duplicated deliveries: %d", count)
	}

	// The owner-facing read model sees the pending delivery; a stranger owner
	// does not.
	listed, listedMatched := store.ListDeliveries(context.Background(), webhook.OwnerUser{ID: recipient}, matchingUserSub.ID, requirePage(t, 10, 0)).(webhook.ListDeliveriesStoreListed)
	if !listedMatched || len(listed.Values) != 1 {
		t.Fatalf("deliveries listing = %#v", listed)
	}
	if listed.Values[0].State != webhook.DeliveryStatePending || listed.Values[0].AttemptCount != 0 {
		t.Fatalf("pending delivery = %#v", listed.Values[0])
	}
	if _, denied := store.ListDeliveries(context.Background(), webhook.OwnerUser{ID: bystander}, matchingUserSub.ID, requirePage(t, 10, 0)).(webhook.ListDeliveriesStoreRejected); !denied {
		t.Fatalf("stranger could read another owner's deliveries")
	}
}

// TestWebhookClaimExclusivity runs two concurrent claimers over one batch of
// due deliveries; FOR UPDATE SKIP LOCKED must hand each row to at most one.
func TestWebhookClaimExclusivity(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)
	parkPendingDeliveries(t, pool)

	recipient := createUser(t, pool, "webhook-claim")
	subscription, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTaskOpened)
	for range 8 {
		appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{})
	}
	drainWebhookPump(t, store)
	if count := countDeliveries(t, pool, subscription.ID); count != 8 {
		t.Fatalf("pending deliveries = %d, want 8", count)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	seen := map[string]int{}
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := store.ClaimDueDeliveries(context.Background(), 6)
			claimed, matched := result.(db.ClaimDueDeliveriesListed)
			if !matched {
				t.Errorf("claim rejected: %#v", result)
				return
			}
			mu.Lock()
			defer mu.Unlock()
			for _, value := range claimed.Values {
				if value.Subscription == subscription.ID {
					seen[value.ID.String()]++
				}
			}
		}()
	}
	wg.Wait()

	for id, claims := range seen {
		if claims != 1 {
			t.Fatalf("delivery %s was claimed %d times", id, claims)
		}
	}
	if len(seen) == 0 {
		t.Fatalf("no deliveries were claimed")
	}
}

// TestWebhookDeliveryMarks walks the store-level state machine: a claimed
// delivery can be retried with an explicit next attempt, ends dead after the
// final attempt, and a delivered row leaves the pending pool for good.
func TestWebhookDeliveryMarks(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)
	parkPendingDeliveries(t, pool)

	recipient := createUser(t, pool, "webhook-marks")
	subscription, _ := createWebhookTestSubscription(t, store, webhook.OwnerUser{ID: recipient}, event.KindTaskOpened)
	appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{})
	appendWebhookTestEvent(t, pool, event.KindTaskOpened, recipient, []core.UserID{recipient}, event.NoOrganization{})
	drainWebhookPump(t, store)

	claimed := claimAllForSubscription(t, store, pool, subscription.ID)
	if len(claimed) != 2 {
		t.Fatalf("claimed %d deliveries, want 2", len(claimed))
	}
	first, second := claimed[0], claimed[1]

	// Retry scheduling records the failure and the caller-computed slot.
	retryAt := time.Now().UTC().Add(30 * time.Second)
	if _, matched := store.MarkFailed(context.Background(), first.ID, 1, "500", db.RetryLater{NextAttemptAt: retryAt}).(db.MarkDeliveryRecorded); !matched {
		t.Fatalf("mark failed (retry) rejected")
	}
	deliveries := listDeliveriesByID(t, store, recipient, subscription.ID)
	if deliveries[first.ID.String()].State != webhook.DeliveryStatePending || deliveries[first.ID.String()].AttemptCount != 1 || deliveries[first.ID.String()].LastStatus != "500" {
		t.Fatalf("retried delivery = %#v", deliveries[first.ID.String()])
	}

	// The final attempt ends the walk dead.
	if _, matched := store.MarkFailed(context.Background(), first.ID, 6, "timeout", db.Dead{}).(db.MarkDeliveryRecorded); !matched {
		t.Fatalf("mark failed (dead) rejected")
	}
	deliveries = listDeliveriesByID(t, store, recipient, subscription.ID)
	if deliveries[first.ID.String()].State != webhook.DeliveryStateDead {
		t.Fatalf("dead delivery = %#v", deliveries[first.ID.String()])
	}
	// A dead delivery cannot be marked again.
	if _, matched := store.MarkDelivered(context.Background(), first.ID, 7, "200").(db.MarkDeliveryRejected); !matched {
		t.Fatalf("dead delivery accepted a delivered mark")
	}

	// Success removes the row from the pending pool.
	if _, matched := store.MarkDelivered(context.Background(), second.ID, 1, "200").(db.MarkDeliveryRecorded); !matched {
		t.Fatalf("mark delivered rejected")
	}
	deliveries = listDeliveriesByID(t, store, recipient, subscription.ID)
	if deliveries[second.ID.String()].State != webhook.DeliveryStateDelivered || deliveries[second.ID.String()].LastStatus != "200" {
		t.Fatalf("delivered delivery = %#v", deliveries[second.ID.String()])
	}
}

// claimAllForSubscription forces every pending delivery of one subscription
// due and claims until no new rows appear, returning each of that
// subscription's rows exactly once (an unmarked claimed row goes back to
// pending, so repeat claims are deduplicated by id).
func claimAllForSubscription(t *testing.T, store db.WebhookStore, pool *pgxpool.Pool, subscriptionID core.WebhookSubscriptionID) []db.ClaimedWebhookDelivery {
	t.Helper()
	seen := map[string]int{}
	values := make([]db.ClaimedWebhookDelivery, 0)
	for range 10 {
		forceDeliveriesDue(t, pool, subscriptionID)
		result := store.ClaimDueDeliveries(context.Background(), 10)
		claimed, matched := result.(db.ClaimDueDeliveriesListed)
		if !matched {
			t.Fatalf("claim rejected: %#v", result)
		}
		progressed := false
		for _, value := range claimed.Values {
			if value.Subscription != subscriptionID || seen[value.ID.String()] > 0 {
				continue
			}
			seen[value.ID.String()]++
			values = append(values, value)
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return values
}

// parkPendingDeliveries pushes every already-pending delivery in the shared
// database out of the due window, so a test's claims only compete with the
// deliveries the test itself creates (the integration database persists
// across tests and runs).
func parkPendingDeliveries(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		update webhook_deliveries set next_attempt_at = now() + interval '12 hours'
		where state = 'pending'
	`); err != nil {
		t.Fatalf("park pending deliveries: %v", err)
	}
}

// forceDeliveriesDue rewinds next_attempt_at so pending rows are immediately
// claimable, standing in for the passage of scheduled retry time.
func forceDeliveriesDue(t *testing.T, pool *pgxpool.Pool, subscriptionID core.WebhookSubscriptionID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		update webhook_deliveries set next_attempt_at = now() - interval '1 second'
		where subscription_id = $1 and state = 'pending'
	`, subscriptionID.String()); err != nil {
		t.Fatalf("force deliveries due: %v", err)
	}
}

func listDeliveriesByID(t *testing.T, store db.WebhookStore, owner core.UserID, subscriptionID core.WebhookSubscriptionID) map[string]webhook.Delivery {
	t.Helper()
	listed, matched := store.ListDeliveries(context.Background(), webhook.OwnerUser{ID: owner}, subscriptionID, requirePage(t, 50, 0)).(webhook.ListDeliveriesStoreListed)
	if !matched {
		t.Fatalf("list deliveries rejected")
	}
	byID := map[string]webhook.Delivery{}
	for _, value := range listed.Values {
		byID[value.ID.String()] = value
	}
	return byID
}

func insertPublicVisibility(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), `
		insert into task_visibility_scopes (task_id, visibility_kind, scope_key)
		values ($1, 'public', 'public')
	`, taskID.String()); err != nil {
		t.Fatalf("insert public visibility: %v", err)
	}
}

func setTaskType(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, taskType string) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), "update tasks set task_type = $2 where id = $1", taskID.String(), taskType); err != nil {
		t.Fatalf("set task type: %v", err)
	}
}

func appendTaskOpenedEvent(t *testing.T, pool *pgxpool.Pool, taskID core.TaskID, actor core.UserID) event.StoredEvent {
	t.Helper()
	id, matched := core.NewDomainEventID().(core.DomainEventIDCreated)
	if !matched {
		t.Fatalf("domain event id rejected")
	}
	subject := event.NoSubjectRefs()
	subject.Task = event.TaskSubject{ID: taskID}
	appended, appendMatched := db.NewEventStore(pool).Append(context.Background(), event.Event{
		ID:         id.Value,
		Kind:       event.KindTaskOpened,
		Actor:      event.ActorUser{ID: actor},
		Subject:    subject,
		Metadata:   event.EmptyMetadata(),
		OccurredAt: time.Now().UTC(),
	}, event.NewRecipients(actor)).(event.AppendStoreAccepted)
	if !appendMatched {
		t.Fatalf("append task_opened event rejected")
	}
	return appended.Value
}

func createMarketplaceSubscription(t *testing.T, store db.WebhookStore, owner webhook.Owner, audience webhook.MarketplaceAudience) webhook.Subscription {
	t.Helper()
	created, matched := webhook.NewService(store).Create(context.Background(), owner,
		webhook.NewEndpointURL("https://receiver.invalid/marketplace").(webhook.EndpointURLAccepted).Value,
		webhook.NewKindFilter([]event.Kind{event.KindTaskOpened}).(webhook.KindFilterAccepted).Value,
		audience,
	).(webhook.SubscriptionCreated)
	if !matched {
		t.Fatalf("create marketplace subscription rejected")
	}
	return created.Value
}

// TestWebhookMarketplaceFanOut proves the marketplace audience: a stranger's
// subscription receives every public open task_opened event (no recipient
// relationship required), the optional task-type and minimum-reward filters
// gate expansion, private tasks never match, and a rogue marketplace row
// listening for another kind never matches. Recipient subscriptions keep
// their existing recipient-gated behavior (TestWebhookPumpFanOut).
func TestWebhookMarketplaceFanOut(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)

	creator := createUser(t, pool, "marketplace-creator")
	stranger := createUser(t, pool, "marketplace-stranger")

	publicTask := insertTask(t, pool, creator, "open", 50)
	insertPublicVisibility(t, pool, publicTask)
	setTaskType(t, pool, publicTask, "code_review")

	privateTask := insertTask(t, pool, creator, "open", 50)
	insertTaskUserVisibility(t, pool, privateTask, creator)

	plainSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, webhook.NewMarketplaceAudience())

	matchingType := webhook.NewMarketplaceAudience()
	matchingType.TaskType = webhook.MarketplaceTaskTypeIs{Value: task.TaskTypeCodeReview}
	typeMatchSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, matchingType)

	otherType := webhook.NewMarketplaceAudience()
	otherType.TaskType = webhook.MarketplaceTaskTypeIs{Value: task.TaskTypeQATesting}
	typeMissSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, otherType)

	affordable := webhook.NewMarketplaceAudience()
	affordable.MinReward = webhook.NewMinimumCreditReward(40).(webhook.MinimumCreditRewardAccepted).Value
	rewardMatchSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, affordable)

	expensive := webhook.NewMarketplaceAudience()
	expensive.MinReward = webhook.NewMinimumCreditReward(51).(webhook.MinimumCreditRewardAccepted).Value
	rewardMissSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, expensive)

	// A rogue marketplace row listening for a non-task_opened kind (the
	// service refuses to create one; this simulates drift) must never match.
	rogueSub := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, webhook.NewMarketplaceAudience())
	if _, err := pool.Exec(context.Background(),
		"update webhook_subscription_kinds set kind = 'task_funded' where subscription_id = $1", rogueSub.ID.String()); err != nil {
		t.Fatalf("rewrite rogue subscription kind: %v", err)
	}

	appendTaskOpenedEvent(t, pool, publicTask, creator)
	appendTaskOpenedEvent(t, pool, privateTask, creator)

	pumped, matched := store.PumpEvents(context.Background()).(db.PumpEventsCompleted)
	if !matched {
		t.Fatalf("pump rejected")
	}
	if pumped.ExpandedEvents != 2 {
		t.Fatalf("pump expanded %d events, want 2", pumped.ExpandedEvents)
	}

	for name, expectation := range map[string]struct {
		subscription webhook.Subscription
		want         int
	}{
		"unfiltered marketplace sub":   {plainSub, 1},
		"matching task type filter":    {typeMatchSub, 1},
		"non-matching task type":       {typeMissSub, 0},
		"reward at or above the floor": {rewardMatchSub, 1},
		"reward below the floor":       {rewardMissSub, 0},
		"rogue non-task_opened kind":   {rogueSub, 0},
	} {
		if count := countDeliveries(t, pool, expectation.subscription.ID); count != expectation.want {
			t.Fatalf("%s has %d deliveries, want %d", name, count, expectation.want)
		}
	}
}

// TestWebhookMarketplaceIgnoresClosedTasks proves a marketplace delivery is
// only expanded while the task is still open and public at pump time.
func TestWebhookMarketplaceIgnoresClosedTasks(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	drainWebhookPump(t, store)

	creator := createUser(t, pool, "marketplace-closed-creator")
	stranger := createUser(t, pool, "marketplace-closed-stranger")
	closedTask := insertTask(t, pool, creator, "open", 50)
	insertPublicVisibility(t, pool, closedTask)

	subscription := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: stranger}, webhook.NewMarketplaceAudience())

	appendTaskOpenedEvent(t, pool, closedTask, creator)
	setTaskState(t, pool, closedTask, "cancelled")

	if _, matched := store.PumpEvents(context.Background()).(db.PumpEventsCompleted); !matched {
		t.Fatalf("pump rejected")
	}
	if count := countDeliveries(t, pool, subscription.ID); count != 0 {
		t.Fatalf("cancelled task produced %d marketplace deliveries, want 0", count)
	}
}

// TestWebhookAudienceRoundTrip proves the audience and its filters survive
// the store round trip.
func TestWebhookAudienceRoundTrip(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	owner := createUser(t, pool, "marketplace-roundtrip")

	audience := webhook.NewMarketplaceAudience()
	audience.TaskType = webhook.MarketplaceTaskTypeIs{Value: task.TaskTypeSecurityReview}
	audience.MinReward = webhook.NewMinimumCreditReward(75).(webhook.MinimumCreditRewardAccepted).Value
	created := createMarketplaceSubscription(t, store, webhook.OwnerUser{ID: owner}, audience)

	listed, matched := store.ListSubscriptions(context.Background(), webhook.OwnerUser{ID: owner}, core.DefaultPage()).(webhook.ListStoreListed)
	if !matched || len(listed.Values) != 1 {
		t.Fatalf("list subscriptions rejected or wrong count")
	}
	restored, restoredMatched := listed.Values[0].Audience.(webhook.MarketplaceAudience)
	if !restoredMatched {
		t.Fatalf("audience = %T, want MarketplaceAudience", listed.Values[0].Audience)
	}
	taskTypeFilter, taskTypeMatched := restored.TaskType.(webhook.MarketplaceTaskTypeIs)
	if !taskTypeMatched || taskTypeFilter.Value != task.TaskTypeSecurityReview {
		t.Fatalf("task type filter did not round-trip: %#v", restored.TaskType)
	}
	rewardFilter, rewardMatched := restored.MinReward.(webhook.MinimumCreditReward)
	if !rewardMatched || rewardFilter.Amount() != 75 {
		t.Fatalf("minimum reward filter did not round-trip: %#v", restored.MinReward)
	}
	if listed.Values[0].ID != created.ID {
		t.Fatalf("listed subscription id mismatch")
	}
}
