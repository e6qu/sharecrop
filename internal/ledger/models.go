package ledger

import "github.com/e6qu/sharecrop/internal/core"

// TaskReference records whether a ledger entry is tied to a task.
type TaskReference interface {
	taskReference()
}

type NoTaskReference struct{}

type TaskReferenced struct {
	TaskID core.TaskID
}

func (NoTaskReference) taskReference() {}

func (TaskReferenced) taskReference() {}

// LedgerEntry is an append-only record of one credit movement on an account.
type LedgerEntry struct {
	ID      core.LedgerEntryID
	Kind    EntryKind
	Amount  SignedAmount
	TaskRef TaskReference
}

// TaskEscrow holds credits reserved for a task reward.
type TaskEscrow struct {
	TaskID          core.TaskID
	FunderAccountID core.CreditAccountID
	Amount          CreditAmount
	State           EscrowState
}

// DeriveBalance sums the signed amounts of an account's ledger entries.
func DeriveBalance(entries []LedgerEntry) Balance {
	total := int64(0)
	for index := range entries {
		total += entries[index].Amount.Int64()
	}
	return NewBalance(total)
}
