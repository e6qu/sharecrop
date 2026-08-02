package db

import (
	"context"
	"fmt"
	"time"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/ledger"
)

// This file holds the UTC-calendar-day budget counters behind agent work
// budgets and the peer-transfer velocity ceiling. Counters live in
// work_day_counters keyed by (key, day): consuming inside the mutating
// transaction makes budget checks atomic with the mutation they gate, so
// concurrent requests cannot overshoot a limit.

const workDayLayout = "2006-01-02"

// Day-counter key prefixes. One key namespace per enforced dimension.
const (
	workDayKeyAgentTasks     = "agent_tasks:"     // + credential id: reservations and direct submissions per day
	workDayKeyAgentSpend     = "agent_spend:"     // + credential id: credits spent per day via the credential
	workDayKeyPeerSend       = "peer_send:"       // + source credit-account id: peer-transfer velocity
	workDayKeyBudgetRefusals = "budget_refusals"  // global count of refused budget consumptions per day
	workDayRetention         = 7 * 24 * time.Hour // refusal totals stay readable for a week before eviction
)

// utcDay renders an instant's UTC calendar day, the budget window key.
func utcDay(now time.Time) string {
	return now.UTC().Format(workDayLayout)
}

// utcDayStart is the instant a UTC calendar day begins, for day-total
// aggregate queries over timestamped rows.
func utcDayStart(now time.Time) time.Time {
	day := now.UTC()
	return time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
}

type dayCounterChargeResult interface {
	dayCounterChargeResult()
}

type dayCounterCharged struct{}

type dayCounterExhausted struct {
	// usedBefore is the consumption already recorded for the day before this
	// refused charge.
	usedBefore int64
}

type dayCounterChargeFailed struct{}

func (dayCounterCharged) dayCounterChargeResult() {}

func (dayCounterExhausted) dayCounterChargeResult() {}

func (dayCounterChargeFailed) dayCounterChargeResult() {}

// chargeWorkDayCounter consumes amount from the (key, day) counter inside the
// caller's transaction, refusing when the consumption would exceed limit. The
// upsert takes a row lock, so two concurrent transactions charging the same
// key serialize and cannot jointly overshoot; a refused charge is undone by
// the caller's rollback.
func chargeWorkDayCounter(ctx context.Context, tx Tx, key string, day string, amount int64, limit int64) dayCounterChargeResult {
	var used int64
	err := tx.QueryRow(ctx, `
		insert into work_day_counters (key, day, used)
		values ($1, $2, $3)
		on conflict (key, day) do update set used = work_day_counters.used + excluded.used
		returning used
	`, key, day, amount).Scan(&used)
	if err != nil {
		return dayCounterChargeFailed{}
	}
	if used > limit {
		return dayCounterExhausted{usedBefore: used - amount}
	}
	return dayCounterCharged{}
}

// recordBudgetRefusal bumps the global per-day refusal total in its own
// statement (outside the refused, soon-rolled-back transaction), best-effort:
// the counter feeds the ops read model, and failing to record it must not
// change the caller's refusal.
func recordBudgetRefusal(ctx context.Context, querier Querier, day string) {
	_, _ = querier.Exec(ctx, `
		insert into work_day_counters (key, day, used)
		values ($1, $2, 1)
		on conflict (key, day) do update set used = work_day_counters.used + 1
	`, workDayKeyBudgetRefusals, day)
}

// budgetExceededError builds the machine-parseable budget_exceeded refusal
// with the human message for one exhausted dimension.
func budgetExceededError(message string) core.DomainError {
	return core.NewDomainError(core.ErrorCodeBudgetExceeded, message)
}

// dailyTaskBudgetMessage words a refused daily-task consumption:
// "daily task budget exhausted (5 of 5 used today, resets at 00:00 UTC)".
func dailyTaskBudgetMessage(usedBefore int64, limit int64) string {
	return fmt.Sprintf("daily task budget exhausted (%d of %d used today, resets at 00:00 UTC)", usedBefore, limit)
}

func concurrentReservationMessage(active int64, limit int64) string {
	return fmt.Sprintf("concurrent reservation budget exhausted (%d of %d reservations active)", active, limit)
}

func dailySpendBudgetMessage(spentBefore int64, limit int64, amount int64) string {
	return fmt.Sprintf("daily credit spend budget exhausted (%d of %d credits spent today, this %d-credit spend exceeds it, resets at 00:00 UTC)", spentBefore, limit, amount)
}

func peerTransferVelocityMessage(sentBefore int64, limit int64, amount int64) string {
	return fmt.Sprintf("daily peer transfer ceiling reached (%d of %d credits sent today, this %d-credit send exceeds it, resets at 00:00 UTC)", sentBefore, limit, amount)
}

// applySpendCharge consumes a work-seeking credential's daily credit spend
// budget inside the caller's ledger transaction. NoSpendCharge is a no-op.
// The credential row is locked first so concurrent spends through the same
// credential serialize and cannot jointly overshoot the cap.
func applySpendCharge(ctx context.Context, tx Tx, charge ledger.SpendCharge) *core.DomainError {
	budgeted, matched := charge.(ledger.ChargeSpendBudget)
	if !matched {
		return nil
	}
	var lockedID string
	if err := tx.QueryRow(ctx, "select id::text from agent_credentials where id = $1 for update", budgeted.CredentialID.String()).Scan(&lockedID); err != nil {
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "lock agent credential for spend budget failed")
		return &reason
	}
	amount := budgeted.Amount.Int64()
	limit := budgeted.DayLimit.Int64()
	chargeResult := chargeWorkDayCounter(ctx, tx, workDayKeyAgentSpend+budgeted.CredentialID.String(), utcDay(time.Now()), amount, limit)
	switch typed := chargeResult.(type) {
	case dayCounterCharged:
		return nil
	case dayCounterExhausted:
		reason := budgetExceededError(dailySpendBudgetMessage(typed.usedBefore, limit, amount))
		return &reason
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "consume daily spend budget failed")
		return &reason
	}
}

// applyPeerTransferVelocity consumes the per-subject daily peer-transfer
// ceiling for the debited account inside the transfer transaction. It applies
// to every sender - user sessions included - because it is an anti-abuse
// velocity limit, not an agent work budget; platform-admin grants use a
// different path and are exempt.
func applyPeerTransferVelocity(ctx context.Context, tx Tx, sourceAccountID string, amount int64) *core.DomainError {
	chargeResult := chargeWorkDayCounter(ctx, tx, workDayKeyPeerSend+sourceAccountID, utcDay(time.Now()), amount, ledger.DailyPeerTransferCeilingCredits)
	switch typed := chargeResult.(type) {
	case dayCounterCharged:
		return nil
	case dayCounterExhausted:
		reason := budgetExceededError(peerTransferVelocityMessage(typed.usedBefore, ledger.DailyPeerTransferCeilingCredits, amount))
		return &reason
	default:
		reason := core.NewDomainError(core.ErrorCodeInvalidState, "consume peer transfer velocity budget failed")
		return &reason
	}
}

// EvictStaleWorkDayCounters removes day counters older than the retention
// window so budget rows cannot accumulate without bound. The lifecycle runner
// calls it on a fixed cadence.
func (store OpsCountersStore) EvictStaleWorkDayCounters(ctx context.Context) error {
	cutoff := utcDay(store.now().Add(-workDayRetention))
	_, err := store.db.Exec(ctx, "delete from work_day_counters where day < $1", cutoff)
	return err
}
