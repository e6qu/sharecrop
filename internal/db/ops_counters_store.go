package db

import (
	"context"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OpsCountersStore is the host-side operations read model: cheap aggregate
// counts over the event outbox, webhook deliveries, and the day's ledger and
// budget activity. Like the lifecycle sweeps, it is a struct-only surface -
// absent from every bridged store interface - so the WASI guest cannot reach
// it; the REST exposure is a separate concern.
type OpsCountersStore struct {
	db  Beginner
	now func() time.Time
}

func NewOpsCountersStore(pool *pgxpool.Pool) OpsCountersStore {
	return NewOpsCountersStoreFromHandle(NewPGX(pool))
}

func NewOpsCountersStoreFromHandle(handle Beginner) OpsCountersStore {
	return OpsCountersStore{db: handle, now: time.Now}
}

// OldestPendingWebhookAge says how long the oldest still-pending webhook
// delivery has been waiting, or that nothing is pending.
type OldestPendingWebhookAge interface {
	oldestPendingWebhookAge()
}

type NoPendingWebhookDeliveries struct{}

type OldestPendingWebhookWaiting struct {
	Age time.Duration
}

func (NoPendingWebhookDeliveries) oldestPendingWebhookAge() {}

func (OldestPendingWebhookWaiting) oldestPendingWebhookAge() {}

// OpsCounters is one consistent snapshot of the operational aggregates.
type OpsCounters struct {
	// OutboxRecordedBacklog counts domain events still awaiting dispatch.
	OutboxRecordedBacklog int64
	// OutboxDispatchFailed counts events retired after exhausting their
	// dispatch attempts.
	OutboxDispatchFailed int64
	// WebhookDeliveriesPending / WebhookDeliveriesDead count delivery rows in
	// those states; OldestPending reports the age of the oldest pending row.
	WebhookDeliveriesPending int64
	WebhookDeliveriesDead    int64
	OldestPending            OldestPendingWebhookAge
	// DaySignupGrants counts signup_grant ledger entries written since the
	// current UTC day began (user and organization grants).
	DaySignupGrants int64
	// DayPeerTransfers counts peer transfers since the UTC day began (one per
	// transfer, not per double-entry row); DayPeerTransferCredits sums the
	// credits they moved.
	DayPeerTransfers       int64
	DayPeerTransferCredits int64
	// DayBudgetRefusals counts work-budget refusals (all dimensions combined) since the
	// UTC day began.
	DayBudgetRefusals int64
}

type OpsCountersResult interface {
	opsCountersResult()
}

type OpsCountersCounted struct {
	Value OpsCounters
}

type OpsCountersUnavailable struct {
	Reason core.DomainError
}

func (OpsCountersCounted) opsCountersResult() {}

func (OpsCountersUnavailable) opsCountersResult() {}

// Count runs the aggregate queries and assembles the snapshot. Each query is
// cheap: the outbox backlog and pending-delivery reads are served by partial
// or state-prefixed indexes, and the day totals scan only the current day's
// rows by timestamp.
func (store OpsCountersStore) Count(ctx context.Context) OpsCountersResult {
	now := store.now()
	counters := OpsCounters{OldestPending: NoPendingWebhookDeliveries{}}

	if err := store.db.QueryRow(ctx, "select count(*) from domain_events where dispatch_state = $1", "recorded").Scan(&counters.OutboxRecordedBacklog); err != nil {
		return opsCountersFailure("count recorded outbox backlog failed")
	}
	if err := store.db.QueryRow(ctx, "select count(*) from domain_events where dispatch_state = $1", "dispatch_failed").Scan(&counters.OutboxDispatchFailed); err != nil {
		return opsCountersFailure("count dispatch-failed outbox events failed")
	}
	if err := store.db.QueryRow(ctx, "select count(*) from webhook_deliveries where state = $1", "pending").Scan(&counters.WebhookDeliveriesPending); err != nil {
		return opsCountersFailure("count pending webhook deliveries failed")
	}
	if err := store.db.QueryRow(ctx, "select count(*) from webhook_deliveries where state = $1", "dead").Scan(&counters.WebhookDeliveriesDead); err != nil {
		return opsCountersFailure("count dead webhook deliveries failed")
	}

	var oldestPending *time.Time
	if err := store.db.QueryRow(ctx, "select min(created_at) from webhook_deliveries where state = $1", "pending").Scan(&oldestPending); err != nil {
		return opsCountersFailure("read oldest pending webhook delivery failed")
	}
	if oldestPending != nil {
		age := now.Sub(*oldestPending)
		if age < 0 {
			age = 0
		}
		counters.OldestPending = OldestPendingWebhookWaiting{Age: age}
	}

	dayStart := utcDayStart(now)
	if err := store.db.QueryRow(ctx, "select count(*) from ledger_entries where kind = $1 and created_at >= $2",
		ledger.EntryKindSignupGrant.String(), dayStart).Scan(&counters.DaySignupGrants); err != nil {
		return opsCountersFailure("count day signup grants failed")
	}
	// A peer transfer writes a debit and a credit row; counting the debits
	// (amount < 0) counts transfers once and sums the credits they moved.
	if err := store.db.QueryRow(ctx, "select count(*), coalesce(sum(-amount), 0) from ledger_entries where kind = $1 and amount < 0 and created_at >= $2",
		ledger.EntryKindPeerTransfer.String(), dayStart).Scan(&counters.DayPeerTransfers, &counters.DayPeerTransferCredits); err != nil {
		return opsCountersFailure("count day peer transfers failed")
	}
	if err := store.db.QueryRow(ctx, "select coalesce(sum(used), 0) from work_day_counters where key = $1 and day = $2",
		workDayKeyBudgetRefusals, utcDay(now)).Scan(&counters.DayBudgetRefusals); err != nil {
		return opsCountersFailure("count day budget refusals failed")
	}

	return OpsCountersCounted{Value: counters}
}

func opsCountersFailure(message string) OpsCountersResult {
	return OpsCountersUnavailable{Reason: core.NewDomainError(core.ErrorCodeUnavailable, message)}
}

// ReadOpsCounters adapts Count to the HTTP server's OpsCountersReader
// boundary (like SavedQueueViewStore, the store satisfies the httpserver
// interface structurally), flattening the sealed oldest-pending option to
// whole seconds: 0 means no pending delivery.
func (store OpsCountersStore) ReadOpsCounters(ctx context.Context) httpserver.OpsCountersReadResult {
	result := store.Count(ctx)
	counted, matched := result.(OpsCountersCounted)
	if !matched {
		return httpserver.OpsCountersReadRejected{Reason: result.(OpsCountersUnavailable).Reason}
	}
	view := httpserver.OpsCountersView{
		OutboxRecordedBacklog:    counted.Value.OutboxRecordedBacklog,
		OutboxDispatchFailed:     counted.Value.OutboxDispatchFailed,
		WebhookDeliveriesPending: counted.Value.WebhookDeliveriesPending,
		WebhookDeliveriesDead:    counted.Value.WebhookDeliveriesDead,
		SignupGrantsToday:        counted.Value.DaySignupGrants,
		PeerTransfersToday:       counted.Value.DayPeerTransfers,
		PeerTransferCreditsToday: counted.Value.DayPeerTransferCredits,
		BudgetRefusalsToday:      counted.Value.DayBudgetRefusals,
	}
	if waiting, pending := counted.Value.OldestPending.(OldestPendingWebhookWaiting); pending {
		view.OldestPendingWebhookAgeSeconds = int64(waiting.Age / time.Second)
	}
	return httpserver.OpsCountersRead{Value: view}
}
