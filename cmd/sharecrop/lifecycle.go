package main

import (
	"context"
	"errors"
	"log/slog"
	"math/rand/v2"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/event"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/runner"
	"github.com/e6qu/sharecrop/internal/webhookdispatch"
)

// Sweep cadences for the lifecycle runner. Reservation and task expiry run
// every minute (a released reservation or expired task should surface within
// a minute of its instant passing); privacy retention is an hourly
// housekeeping pass; rate-limit buckets and idle MCP sessions are reclaimed
// on slower cycles because the request path already ignores stale rows; the
// webhook pump runs at webhookdispatch.PumpInterval.
const (
	reservationExpirySweepInterval = time.Minute
	taskExpirySweepInterval        = time.Minute
	privacyRetentionSweepInterval  = time.Hour
	rateLimitEvictionSweepInterval = 5 * time.Minute
	mcpSessionSweepInterval        = 10 * time.Minute

	// The event dispatch sweep is the crash-recovery half of the outbox: it
	// re-dispatches events whose mutation committed but whose inline
	// dispatch was lost. It runs on a short cycle so lost notifications and
	// webhook deliveries surface quickly, and only touches recorded rows
	// older than a grace period so it never races a mutation whose inline
	// dispatch is still in flight.
	eventDispatchSweepInterval = 30 * time.Second
	eventDispatchGracePeriod   = 10 * time.Second
	// eventDispatchSweepBatch bounds one recovery pass; the next tick
	// continues where it left off.
	eventDispatchSweepBatch = 200
)

// buildLifecycleRunner assembles the host-side lifecycle runner over the
// serve process's Postgres pool and event recorder. It runs in both native
// and WASI-hosted modes: the sweeps are host-side either way. runMCPStdio
// gets no runner.
func buildLifecycleRunner(logger *slog.Logger, pool *pgxpool.Pool, recorder event.Recorder) *runner.Runner {
	taskStore := db.NewTaskStore(pool)
	privacyStore := db.NewPrivacyStore(pool)
	mcpSessions := db.NewMCPSessionStore(pool)
	ipLimiter := db.NewRateLimiter(pool, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec)
	subjectLimiter := db.NewRateLimiter(pool, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec)
	dispatcher := webhookdispatch.New(db.NewWebhookStore(pool), webhookdispatch.StrictDialPolicy(), time.Now, rand.NewPCG(rand.Uint64(), rand.Uint64()))

	return runner.New(logger, []runner.Sweep{
		{
			Name:     "reservation_expiry",
			Interval: reservationExpirySweepInterval,
			Run: func(ctx context.Context) error {
				return sweepDueReservations(ctx, taskStore, recorder)
			},
		},
		{
			Name:     "task_expiry",
			Interval: taskExpirySweepInterval,
			Run: func(ctx context.Context) error {
				return sweepDueTasks(ctx, taskStore, recorder)
			},
		},
		{
			Name:     "privacy_retention",
			Interval: privacyRetentionSweepInterval,
			Run: func(ctx context.Context) error {
				result := privacyStore.RunRetention(ctx, core.SystemUserID())
				if rejected, matched := result.(httpserver.PrivacyRetentionRejected); matched {
					return errors.New(rejected.Reason.Description())
				}
				return nil
			},
		},
		{
			Name:     "rate_limit_eviction",
			Interval: rateLimitEvictionSweepInterval,
			Run: func(ctx context.Context) error {
				if err := ipLimiter.EvictExpired(ctx); err != nil {
					return err
				}
				return subjectLimiter.EvictExpired(ctx)
			},
		},
		{
			Name:     "mcp_session_eviction",
			Interval: mcpSessionSweepInterval,
			Run: func(ctx context.Context) error {
				_, err := mcpSessions.DeleteExpiredMCPSessions(ctx, time.Now().UTC().Add(-httpserver.MCPSessionTTL))
				return err
			},
		},
		{
			Name:     "event_dispatch",
			Interval: eventDispatchSweepInterval,
			Run: func(ctx context.Context) error {
				return sweepRecordedEvents(ctx, db.NewEventStore(pool), recorder)
			},
		},
		{
			Name:     "webhook_pump",
			Interval: webhookdispatch.PumpInterval,
			Run: func(ctx context.Context) error {
				result := dispatcher.RunOnce(ctx)
				if rejected, matched := result.(webhookdispatch.RunRejected); matched {
					return errors.New(rejected.Reason.Description())
				}
				return nil
			},
		},
	})
}

// sweepRecordedEvents dispatches stale recorded events: rows the outbox
// committed whose inline dispatch (inbox fan-out, webhook expansion) was
// lost to a crash. Dispatch is idempotent per event, so racing a slow inline
// dispatch is harmless.
func sweepRecordedEvents(ctx context.Context, eventStore db.EventStore, recorder event.Recorder) error {
	cutoff := time.Now().UTC().Add(-eventDispatchGracePeriod)
	result := eventStore.ListRecordedBefore(ctx, cutoff, eventDispatchSweepBatch)
	listed, matched := result.(db.RecordedEventsListed)
	if !matched {
		return errors.New(result.(db.ListRecordedRejected).Reason.Description())
	}
	recorder.Dispatch(ctx, listed.Values...)
	return nil
}

// sweepDueReservations releases expired reservations and emits one
// reservation_expired event per released row, notifying the reservation
// holder and the task owner as the system actor.
func sweepDueReservations(ctx context.Context, taskStore db.TaskStore, recorder event.Recorder) error {
	result := taskStore.ExpireDueReservations(ctx)
	completed, matched := result.(db.ExpireDueReservationsCompleted)
	if !matched {
		return errors.New(result.(db.ExpireDueReservationsRejected).Reason.Description())
	}
	for _, row := range completed.Values {
		subject := event.NoSubjectRefs()
		subject.Task = event.TaskSubject{ID: row.TaskID}
		subject.Reservation = event.ReservationSubject{ID: row.ReservationID}
		_ = recorder.Emit(ctx, event.EmitCommand{
			Kind:       event.KindReservationExpired,
			Actor:      event.ActorSystem{},
			Subject:    subject,
			Metadata:   event.TaskMetadata(row.TaskID),
			Recipients: event.NewRecipients(row.HolderID, row.TaskOwnerID),
		})
	}
	return nil
}

// sweepDueTasks expires open tasks past their expiration instant (refunding
// escrow and releasing reservations inside the store transaction) and emits
// one task_expired event per expired task, notifying the owner and every
// released reservation holder as the system actor.
func sweepDueTasks(ctx context.Context, taskStore db.TaskStore, recorder event.Recorder) error {
	result := taskStore.ExpireDueTasks(ctx)
	completed, matched := result.(db.ExpireDueTasksCompleted)
	if !matched {
		return errors.New(result.(db.ExpireDueTasksRejected).Reason.Description())
	}
	for _, row := range completed.Values {
		subject := event.NoSubjectRefs()
		subject.Task = event.TaskSubject{ID: row.TaskID}
		recipients := append([]core.UserID{row.OwnerID}, row.ReleasedHolders...)
		_ = recorder.Emit(ctx, event.EmitCommand{
			Kind:       event.KindTaskExpired,
			Actor:      event.ActorSystem{},
			Subject:    subject,
			Metadata:   event.TaskMetadata(row.TaskID),
			Recipients: event.NewRecipients(recipients...),
		})
	}
	return nil
}
