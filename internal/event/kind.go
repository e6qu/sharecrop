// Package event is the domain event stream. Domain services emit events for
// every externally meaningful mutation, regardless of the transport that
// triggered it (REST, MCP, or a background sweep). Notifications, webhook
// deliveries, and the browser live-update feed all derive from this one
// stream, so an action taken over MCP produces exactly the same downstream
// effects as the same action taken over REST.
package event

import "github.com/e6qu/sharecrop/internal/core"

// Kind is the closed set of domain event kinds.
type Kind struct {
	value string
}

var (
	KindTaskOpened                 = Kind{value: "task_opened"}
	KindTaskFunded                 = Kind{value: "task_funded"}
	KindTaskCancelled              = Kind{value: "task_cancelled"}
	KindTaskExpired                = Kind{value: "task_expired"}
	KindTaskCommented              = Kind{value: "task_commented"}
	KindSeriesCommented            = Kind{value: "series_commented"}
	KindReservationRequested       = Kind{value: "reservation_requested"}
	KindReservationApproved        = Kind{value: "reservation_approved"}
	KindReservationDeclined        = Kind{value: "reservation_declined"}
	KindReservationCancelled       = Kind{value: "reservation_cancelled"}
	KindReservationExpired         = Kind{value: "reservation_expired"}
	KindSubmissionCreated          = Kind{value: "submission_created"}
	KindSubmissionAccepted         = Kind{value: "submission_accepted"}
	KindSubmissionChangesRequested = Kind{value: "submission_changes_requested"}
	KindSubmissionRejected         = Kind{value: "submission_rejected"}
	KindSubmissionCommented        = Kind{value: "submission_commented"}
	KindPayoutReceived             = Kind{value: "payout_received"}
	KindTipReceived                = Kind{value: "tip_received"}
	KindCollectibleAwarded         = Kind{value: "collectible_awarded"}
)

// AllKinds returns every event kind. Totality tests over the kind set (the
// notification mapping, the webhook scope entitlement map) iterate this.
func AllKinds() []Kind {
	return []Kind{
		KindTaskOpened,
		KindTaskFunded,
		KindTaskCancelled,
		KindTaskExpired,
		KindTaskCommented,
		KindSeriesCommented,
		KindReservationRequested,
		KindReservationApproved,
		KindReservationDeclined,
		KindReservationCancelled,
		KindReservationExpired,
		KindSubmissionCreated,
		KindSubmissionAccepted,
		KindSubmissionChangesRequested,
		KindSubmissionRejected,
		KindSubmissionCommented,
		KindPayoutReceived,
		KindTipReceived,
		KindCollectibleAwarded,
	}
}

func (kind Kind) String() string {
	return kind.value
}

type KindResult interface {
	kindResult()
}

type KindParsed struct {
	Value Kind
}

type KindRejected struct {
	Reason core.DomainError
}

func (KindParsed) kindResult() {}

func (KindRejected) kindResult() {}

// ParseKind converts a raw boundary string into a sealed Kind.
func ParseKind(raw string) KindResult {
	for _, kind := range AllKinds() {
		if kind.value == raw {
			return KindParsed{Value: kind}
		}
	}
	return KindRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "unknown domain event kind")}
}
