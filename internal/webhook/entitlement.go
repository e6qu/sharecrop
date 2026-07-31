package webhook

import (
	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/event"
)

// RequiredScopeForKind maps each domain event kind to the read scope a
// CREDENTIAL caller must hold to subscribe to it. A webhook delivery carries
// the same facts the matching read endpoint exposes, so subscribing to a
// kind requires the scope that could read it directly: task and reservation
// lifecycle map to tasks_read, submission lifecycle to submissions_read,
// credit movements to ledger_read, and collectible awards to
// collectibles_read. Browser-session users are never scope-checked.
//
// The map is total over event.AllKinds(); a unit test enforces totality so a
// new kind cannot ship without an entitlement decision.
func RequiredScopeForKind(kind event.Kind) agent.Scope {
	switch kind {
	case event.KindTaskOpened, event.KindTaskCancelled, event.KindTaskExpired, event.KindTaskCommented, event.KindSeriesCommented:
		return agent.ScopeTasksRead
	case event.KindReservationRequested, event.KindReservationApproved, event.KindReservationDeclined, event.KindReservationCancelled, event.KindReservationExpired:
		return agent.ScopeTasksRead
	case event.KindSubmissionCreated, event.KindSubmissionAccepted, event.KindSubmissionChangesRequested, event.KindSubmissionRejected, event.KindSubmissionCommented:
		return agent.ScopeSubmissionsRead
	case event.KindTaskFunded, event.KindPayoutReceived, event.KindTipReceived:
		return agent.ScopeLedgerRead
	case event.KindCollectibleAwarded:
		return agent.ScopeCollectiblesRead
	default:
		// Unreachable for a sealed Kind constructed through ParseKind or the
		// package variables; the totality test walks AllKinds() so a newly
		// added kind fails there before this branch could matter. Defaulting
		// to the strictest existing gate keeps the fallback safe regardless.
		return agent.ScopePlatformAdmin
	}
}

// MissingKindEntitlements reports which kinds in the filter the given scope
// set is not entitled to subscribe to. An empty result means the credential
// may subscribe to every requested kind.
func MissingKindEntitlements(scopes agent.ScopeSet, kinds KindFilter) []event.Kind {
	missing := make([]event.Kind, 0)
	for _, kind := range kinds.Values() {
		required := RequiredScopeForKind(kind)
		if _, granted := scopes.Allows(required).(agent.ScopeGranted); !granted {
			missing = append(missing, kind)
		}
	}
	return missing
}
