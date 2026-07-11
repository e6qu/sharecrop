// Package notificationtest holds test-support helpers for
// notification.Notification shared by the notification bridge's codec tests and
// the integration dual-run test, so the two do not carry duplicate comparisons.
package notificationtest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/notification"
)

// NotificationDiff returns a description of the first field in which got and
// want differ, or "" if they are equal. The timestamp is compared by instant.
func NotificationDiff(got, want notification.Notification) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.RecipientID != want.RecipientID:
		return fmt.Sprintf("recipient: %s != %s", got.RecipientID, want.RecipientID)
	case got.ActorID != want.ActorID:
		return fmt.Sprintf("actor: %s != %s", got.ActorID, want.ActorID)
	case got.Kind.String() != want.Kind.String():
		return fmt.Sprintf("kind: %s != %s", got.Kind, want.Kind)
	case got.Subject != want.Subject:
		return fmt.Sprintf("subject: %+v != %+v", got.Subject, want.Subject)
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	case got.Metadata != want.Metadata:
		return fmt.Sprintf("metadata: %+v != %+v", got.Metadata, want.Metadata)
	case !got.CreatedAt.Equal(want.CreatedAt):
		return fmt.Sprintf("created_at: %s != %s", got.CreatedAt, want.CreatedAt)
	default:
		return ""
	}
}
