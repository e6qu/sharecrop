//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/webhook"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
)

// dispatchReplicaCount mirrors a small production fleet: three serve replicas
// each running the event dispatch sweep and the webhook pump against the same
// database.
const dispatchReplicaCount = 3

// TestReplicaDispatchAndPumpAreExactlyOnce runs three concurrent "replicas"
// (independent recorder + dispatcher instances over the shared pool) against
// one backlog of recorded events. Every replica sweeps and pumps repeatedly;
// the combined effects must still be exactly once: one notification per
// (event, recipient), one delivery row per (subscription, event), and each
// delivery posted to the receiver exactly once.
func TestReplicaDispatchAndPumpAreExactlyOnce(t *testing.T) {
	pool := newPool(t)
	store := db.NewWebhookStore(pool)
	parkActiveSubscriptions(t, pool)
	dispatchAllRecordedEvents(t, pool)
	parkPendingDeliveries(t, pool)
	receiver := newWebhookReceiver(t, http.StatusOK)

	actor := createUser(t, pool, "replica-actor")
	recipient := createUser(t, pool, "replica-recipient")
	created, createdMatched := webhook.NewService(store).Create(context.Background(), webhook.OwnerUser{ID: recipient},
		webhook.NewEndpointURL(receiver.server.URL+"/hooks").(webhook.EndpointURLAccepted).Value,
		webhook.NewKindFilter([]event.Kind{event.KindCreditGranted}).(webhook.KindFilterAccepted).Value,
		webhook.RecipientAudience{},
	).(webhook.SubscriptionCreated)
	if !createdMatched {
		t.Fatalf("create subscription rejected")
	}

	const backlog = 40
	eventIDs := make([]core.DomainEventID, 0, backlog)
	for range backlog {
		stored := appendWebhookTestEvent(t, pool, event.KindCreditGranted, actor, []core.UserID{recipient}, event.NoOrganization{})
		eventIDs = append(eventIDs, stored.Event.ID)
	}

	// Each replica gets its own store, recorder, and dispatcher instances;
	// only the pgx pool is shared, as it would be between serve processes
	// pointing at one database.
	recorders := make([]event.Recorder, 0, dispatchReplicaCount)
	dispatchers := make([]webhookdispatch.Dispatcher, 0, dispatchReplicaCount)
	for range dispatchReplicaCount {
		recorders = append(recorders, newDBRecorder(pool))
		dispatchers = append(dispatchers, newTestDispatcher(t, pool, receiver, webhookdispatch.AllowEveryAddress()))
	}

	var wg sync.WaitGroup
	for replica := range dispatchReplicaCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			eventStore := db.NewEventStore(pool)
			for range 5 {
				listResult := eventStore.ListRecordedBefore(context.Background(), time.Now().UTC().Add(time.Hour), 500)
				listed, listMatched := listResult.(db.RecordedEventsListed)
				if !listMatched {
					t.Errorf("replica %d: list recorded events rejected", replica)
					return
				}
				recorders[replica].Dispatch(context.Background(), listed.Values...)
				runResult := dispatchers[replica].RunOnce(context.Background())
				if rejected, rejectedMatched := runResult.(webhookdispatch.RunRejected); rejectedMatched {
					t.Errorf("replica %d: pump rejected: %s", replica, rejected.Reason.Description())
					return
				}
			}
		}()
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	// One deterministic final drain: a delivery created by the last sweep of
	// one replica after the last pump of another is still pending. Extra
	// cycles can only surface duplicates, never hide them.
	for range 5 {
		if runDispatcherOnce(t, dispatchers[0]).Attempted == 0 {
			break
		}
	}

	for _, eventID := range eventIDs {
		if row := readEventOutboxRow(t, pool, eventID); row.dispatchState != "dispatched" {
			t.Fatalf("event %s state = %q, want dispatched", eventID.String(), row.dispatchState)
		}
		if got := countNotificationsForEvent(t, pool, eventID, recipient); got != 1 {
			t.Fatalf("event %s produced %d notifications for the recipient, want 1", eventID.String(), got)
		}
	}
	if got := countDeliveries(t, pool, created.Value.ID); got != backlog {
		t.Fatalf("delivery rows = %d, want %d", got, backlog)
	}
	recorded := receiver.recorded()
	if len(recorded) != backlog {
		t.Fatalf("receiver saw %d posts, want %d (each delivery exactly once)", len(recorded), backlog)
	}
	posts := map[string]int{}
	for _, request := range recorded {
		posts[request.id]++
	}
	for id, sends := range posts {
		if sends != 1 {
			t.Fatalf("delivery %s was posted %d times", id, sends)
		}
	}
}

// TestReplicaExpirySweepsAreExactlyOnce races two lifecycle-runner replicas
// over the same due tasks and due reservations. Each due task must be
// reported by exactly one replica, refund escrow exactly once (the
// expire:<task_id> idempotency key), and record exactly one task_expired
// event; each due reservation must be released and reported exactly once.
func TestReplicaExpirySweepsAreExactlyOnce(t *testing.T) {
	pool := newPool(t)
	ledgerStore := db.NewLedgerStore(pool)
	dispatchAllRecordedEvents(t, pool)

	owner := createUser(t, pool, "expiry-race-owner")
	worker := createUser(t, pool, "expiry-race-worker")

	const expiringTasks = 5
	taskIDs := make([]core.TaskID, 0, expiringTasks)
	for index := range expiringTasks {
		taskID := insertTask(t, pool, owner, "draft", 10)
		key := "fund-expiry-race-" + taskID.String()
		if _, matched := ledgerStore.FundTask(context.Background(), fundCommand(t, owner, taskID, 10, key)).(ledger.TaskFunded); !matched {
			t.Fatalf("fund expiring task %d rejected", index)
		}
		setTaskState(t, pool, taskID, "open")
		insertActiveReservation(t, pool, taskID, worker, false)
		if _, err := pool.Exec(context.Background(), "update tasks set expires_at = now() - interval '1 hour' where id = $1", taskID.String()); err != nil {
			t.Fatalf("set task expiry: %v", err)
		}
		taskIDs = append(taskIDs, taskID)
	}

	const expiringReservations = 4
	reservationTaskIDs := make([]core.TaskID, 0, expiringReservations)
	for range expiringReservations {
		taskID := insertTaskWithRewardKind(t, pool, owner, "open", "none")
		insertActiveReservation(t, pool, taskID, worker, true)
		reservationTaskIDs = append(reservationTaskIDs, taskID)
	}

	var mu sync.Mutex
	reportedTasks := map[string]int{}
	reportedReservations := map[string]int{}
	var wg sync.WaitGroup
	for replica := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Replicate the lifecycle runner's sweep bodies: expire, then emit
			// one event per reported row as the system actor.
			replicaStore := db.NewTaskStore(pool)
			recorder := newDBRecorder(pool)
			taskResult := replicaStore.ExpireDueTasks(context.Background())
			tasksCompleted, tasksMatched := taskResult.(db.ExpireDueTasksCompleted)
			if !tasksMatched {
				t.Errorf("replica %d: task expiry sweep rejected", replica)
				return
			}
			reservationResult := replicaStore.ExpireDueReservations(context.Background())
			reservationsCompleted, reservationsMatched := reservationResult.(db.ExpireDueReservationsCompleted)
			if !reservationsMatched {
				t.Errorf("replica %d: reservation expiry sweep rejected", replica)
				return
			}
			mu.Lock()
			for _, row := range tasksCompleted.Values {
				reportedTasks[row.TaskID.String()]++
			}
			for _, row := range reservationsCompleted.Values {
				reportedReservations[row.ReservationID.String()]++
			}
			mu.Unlock()
			for _, row := range tasksCompleted.Values {
				subject := event.NoSubjectRefs()
				subject.Task = event.TaskSubject{ID: row.TaskID}
				recipients := append([]core.UserID{row.OwnerID}, row.ReleasedHolders...)
				_ = recorder.Emit(context.Background(), event.EmitCommand{
					Kind: event.KindTaskExpired, Actor: event.ActorSystem{}, Subject: subject,
					Metadata: event.TaskMetadata(row.TaskID), Recipients: event.NewRecipients(recipients...),
				})
			}
			for _, row := range reservationsCompleted.Values {
				subject := event.NoSubjectRefs()
				subject.Task = event.TaskSubject{ID: row.TaskID}
				subject.Reservation = event.ReservationSubject{ID: row.ReservationID}
				_ = recorder.Emit(context.Background(), event.EmitCommand{
					Kind: event.KindReservationExpired, Actor: event.ActorSystem{}, Subject: subject,
					Metadata: event.TaskMetadata(row.TaskID), Recipients: event.NewRecipients(row.HolderID, row.TaskOwnerID),
				})
			}
		}()
	}
	wg.Wait()
	if t.Failed() {
		return
	}

	for _, taskID := range taskIDs {
		if got := reportedTasks[taskID.String()]; got != 1 {
			t.Fatalf("task %s was reported by %d sweeps, want exactly 1", taskID.String(), got)
		}
		var refunds int
		if err := pool.QueryRow(context.Background(), `
			select count(*) from ledger_entries
			where task_id = $1 and kind = 'task_refund' and idempotency_key = $2
		`, taskID.String(), "expire:"+taskID.String()).Scan(&refunds); err != nil {
			t.Fatalf("count refunds: %v", err)
		}
		if refunds != 1 {
			t.Fatalf("task %s has %d expiry refunds, want exactly 1", taskID.String(), refunds)
		}
		if got := countEventsByKindForTask(t, pool, taskID, event.KindTaskExpired); got != 1 {
			t.Fatalf("task %s has %d task_expired events, want 1", taskID.String(), got)
		}
	}
	balance := mustBalance(t, ledgerStore, owner)
	if balance.Spendable() != 100 || balance.Allocated() != 0 {
		t.Fatalf("owner balance after racing expiry = %d/%d, want the full 100/0 back", balance.Spendable(), balance.Allocated())
	}

	totalReservationReports := 0
	for reservationID, reports := range reportedReservations {
		if reports != 1 {
			t.Fatalf("reservation %s was reported by %d sweeps, want exactly 1", reservationID, reports)
		}
		totalReservationReports += reports
	}
	// Our own reservation-only tasks each carry exactly one reservation_expired
	// event (the shared database may hold other due reservations; they are
	// counted in reportedReservations but asserted per task only for ours).
	for _, taskID := range reservationTaskIDs {
		if got := countEventsByKindForTask(t, pool, taskID, event.KindReservationExpired); got != 1 {
			t.Fatalf("task %s has %d reservation_expired events, want 1", taskID.String(), got)
		}
	}
	if totalReservationReports < expiringReservations {
		t.Fatalf("sweeps reported %d released reservations, want at least the %d this test created", totalReservationReports, expiringReservations)
	}
}
