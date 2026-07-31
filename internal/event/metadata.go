package event

import (
	"strconv"

	"github.com/e6qu/sharecrop/internal/core"
)

// The metadata builders below compose the small display payloads events carry.
// Every value is a UUID string or an integer, so plain concatenation produces
// valid JSON without an encoder dependency in domain packages. The task_id
// field matches what the pre-recorder handler notifications carried, and the
// browser client's "Open task" inbox link reads it.

// TaskMetadata is the metadata for an event about a task.
func TaskMetadata(taskID core.TaskID) Metadata {
	return Metadata{JSON: `{"task_id":"` + taskID.String() + `"}`}
}

// TaskAmountMetadata is the metadata for an event about a credit amount moved
// on a task (funding, payout, tip).
func TaskAmountMetadata(taskID core.TaskID, amount int64) Metadata {
	return Metadata{JSON: `{"task_id":"` + taskID.String() + `","amount":` + strconv.FormatInt(amount, 10) + `}`}
}

// TaskRefundMetadata marks a task_cancelled event that was caused by a refund
// rather than a plain cancel.
func TaskRefundMetadata(taskID core.TaskID) Metadata {
	return Metadata{JSON: `{"task_id":"` + taskID.String() + `","cause":"refund"}`}
}

// TaskCollectibleMetadata is the metadata for a collectible moved on a task
// (a collectible payout or tip).
func TaskCollectibleMetadata(taskID core.TaskID, collectibleID core.CollectibleID) Metadata {
	return Metadata{JSON: `{"task_id":"` + taskID.String() + `","collectible_id":"` + collectibleID.String() + `"}`}
}
