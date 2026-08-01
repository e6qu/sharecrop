//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	// marketplaceScaleSubscriptions is the fan-out width for the scale test.
	// 300 forces the dispatcher through many claim waves (the claim batch is
	// 10 rows, so a full drain needs 30 claims) while staying CI-friendly:
	// the fan-out expansion and the drain both finish in seconds against a
	// local Postgres and a loopback receiver.
	marketplaceScaleSubscriptions = 300
	// dispatcherRunOnceCapacity mirrors webhookdispatch's claimBatchSize (10)
	// times maxClaimBatchesPerRun (50): the most deliveries one RunOnce
	// attempts before yielding the sweep slot.
	dispatcherRunOnceCapacity = 500
	// fanOutScaleCeiling bounds the whole fan-out-and-drain wall time; the
	// scenario normally finishes in a few seconds, so hitting the ceiling
	// means claiming or delivery has regressed badly.
	fanOutScaleCeiling = 60 * time.Second
)

func countDeliveriesForEvent(t *testing.T, pool *pgxpool.Pool, stored event.StoredEvent, state string) int {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), `
		select count(*) from webhook_deliveries
		where event_seq = (select seq from domain_events where id = $1) and state = $2
	`, stored.Event.ID.String(), state).Scan(&count); err != nil {
		t.Fatalf("count deliveries for event: %v", err)
	}
	return count
}

// TestMarketplaceFanOutDrainsAtScale proves the fan-out path end to end at a
// width beyond one claim batch: N marketplace subscriptions all watching
// task_opened, one public task opens, the dispatch step creates exactly N
// pending deliveries, and repeated RunOnce cycles drain the whole backlog to
// delivered within the expected number of cycles. The receiver's log doubles
// as the no-double-send proof: every delivery id arrives exactly once.
func TestMarketplaceFanOutDrainsAtScale(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)
	receiver := newWebhookReceiver(t, http.StatusOK)
	started := time.Now()

	creator := createUser(t, pool, "scale-creator")
	subscriber := createUser(t, pool, "scale-subscriber")
	service := webhook.NewService(store)
	endpoint, endpointMatched := webhook.NewEndpointURL(receiver.server.URL + "/hooks").(webhook.EndpointURLAccepted)
	if !endpointMatched {
		t.Fatalf("receiver endpoint url rejected")
	}
	kinds, kindsMatched := webhook.NewKindFilter([]event.Kind{event.KindTaskOpened}).(webhook.KindFilterAccepted)
	if !kindsMatched {
		t.Fatalf("kind filter rejected")
	}
	for range marketplaceScaleSubscriptions {
		if _, matched := service.Create(context.Background(), webhook.OwnerUser{ID: subscriber},
			endpoint.Value, kinds.Value, webhook.NewMarketplaceAudience()).(webhook.SubscriptionCreated); !matched {
			t.Fatalf("create marketplace subscription rejected")
		}
	}
	// The shared database persists across tests; do not leave a 300-wide
	// active fan-out target behind for later events.
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `
			update webhook_subscriptions set state = 'revoked', state_recorded_at = now()
			where owner_user_id = $1 and state = 'active'
		`, subscriber.String())
	})

	taskID := insertTask(t, pool, creator, "open", 50)
	insertPublicVisibility(t, pool, taskID)
	stored := appendTaskOpenedEvent(t, pool, taskID, creator)
	dispatchStoredEvent(t, pool, stored)
	if got := countDeliveriesForEvent(t, pool, stored, "pending"); got != marketplaceScaleSubscriptions {
		t.Fatalf("dispatch created %d pending deliveries, want %d", got, marketplaceScaleSubscriptions)
	}

	dispatcher := newTestDispatcher(t, pool, receiver, webhookdispatch.AllowEveryAddress())
	expectedCycles := (marketplaceScaleSubscriptions + dispatcherRunOnceCapacity - 1) / dispatcherRunOnceCapacity
	cycles := 0
	delivered := 0
	for {
		completed := runDispatcherOnce(t, dispatcher)
		if completed.Attempted == 0 {
			break
		}
		delivered += completed.Delivered
		cycles++
		if cycles > expectedCycles {
			t.Fatalf("backlog still draining after %d cycles, want at most %d", cycles, expectedCycles)
		}
		if time.Since(started) > fanOutScaleCeiling {
			t.Fatalf("fan-out drain exceeded the %v ceiling", fanOutScaleCeiling)
		}
	}
	if delivered != marketplaceScaleSubscriptions {
		t.Fatalf("drained %d deliveries, want %d", delivered, marketplaceScaleSubscriptions)
	}
	if got := countDeliveriesForEvent(t, pool, stored, "delivered"); got != marketplaceScaleSubscriptions {
		t.Fatalf("delivered rows = %d, want %d", got, marketplaceScaleSubscriptions)
	}

	// Each delivery id was posted exactly once: no delivery was claimed (and
	// therefore sent) twice.
	recorded := receiver.recorded()
	if len(recorded) != marketplaceScaleSubscriptions {
		t.Fatalf("receiver saw %d requests, want %d", len(recorded), marketplaceScaleSubscriptions)
	}
	seen := map[string]int{}
	for _, request := range recorded {
		seen[request.id]++
	}
	for id, sends := range seen {
		if sends != 1 {
			t.Fatalf("delivery %s was sent %d times", id, sends)
		}
	}
	if elapsed := time.Since(started); elapsed > fanOutScaleCeiling {
		t.Fatalf("scenario took %v, want under %v", elapsed, fanOutScaleCeiling)
	}
}
