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

// TaskFund records the credits currently allocated to a task. A TaskFund exists
// only while the task holds those credits; awarding or refunding the task
// removes it.
type TaskFund struct {
	TaskID          core.TaskID
	FunderAccountID core.CreditAccountID
	CreditAmount    CreditAmount
}

// DeriveSpendable sums the signed amounts of an account's ledger entries, i.e.
// the spendable section of its wallet.
func DeriveSpendable(entries []LedgerEntry) int64 {
	total := int64(0)
	for index := range entries {
		total += entries[index].Amount.Int64()
	}
	return total
}
