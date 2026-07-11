// Package audittest holds test-support helpers for audit.Event that are shared
// by the audit bridge's codec unit tests and the integration dual-run test, so
// the two do not carry duplicate field-by-field comparisons.
package audittest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/audit"
)

// EventDiff returns a description of the first field in which got and want
// differ, or "" if they are equal. The timestamp is compared by instant so two
// times with different monotonic clocks or zones still match.
func EventDiff(got, want audit.Event) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.ActorUserID != want.ActorUserID:
		return fmt.Sprintf("actor: %s != %s", got.ActorUserID, want.ActorUserID)
	case got.Action.String() != want.Action.String():
		return fmt.Sprintf("action: %s != %s", got.Action, want.Action)
	case got.Subject != want.Subject:
		return fmt.Sprintf("subject: %+v != %+v", got.Subject, want.Subject)
	case got.Metadata != want.Metadata:
		return fmt.Sprintf("metadata: %+v != %+v", got.Metadata, want.Metadata)
	case !got.CreatedAt.Equal(want.CreatedAt):
		return fmt.Sprintf("created_at: %s != %s", got.CreatedAt, want.CreatedAt)
	default:
		return ""
	}
}
