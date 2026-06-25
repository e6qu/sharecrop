package assets

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

// CollectibleKind is the typed nature of a platform collectible.
type CollectibleKind struct {
	value string
}

var (
	CollectibleKindUnique  = CollectibleKind{value: "unique"}
	CollectibleKindEdition = CollectibleKind{value: "edition"}
	CollectibleKindBadge   = CollectibleKind{value: "badge"}
)

type CollectibleKindResult interface {
	collectibleKindResult()
}

type CollectibleKindAccepted struct {
	Value CollectibleKind
}

type CollectibleKindRejected struct {
	Reason core.DomainError
}

func (CollectibleKindAccepted) collectibleKindResult() {}

func (CollectibleKindRejected) collectibleKindResult() {}

func ParseCollectibleKind(raw string) CollectibleKindResult {
	switch raw {
	case CollectibleKindUnique.value:
		return CollectibleKindAccepted{Value: CollectibleKindUnique}
	case CollectibleKindEdition.value:
		return CollectibleKindAccepted{Value: CollectibleKindEdition}
	case CollectibleKindBadge.value:
		return CollectibleKindAccepted{Value: CollectibleKindBadge}
	default:
		return CollectibleKindRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "collectible kind is invalid")}
	}
}

func (kind CollectibleKind) String() string {
	return kind.value
}

// CollectibleState is the lifecycle of a collectible.
type CollectibleState struct {
	value string
}

var (
	CollectibleStateMinted   = CollectibleState{value: "minted"}
	CollectibleStateEscrowed = CollectibleState{value: "escrowed"}
	CollectibleStateAwarded  = CollectibleState{value: "awarded"}
)

type CollectibleStateResult interface {
	collectibleStateResult()
}

type CollectibleStateAccepted struct {
	Value CollectibleState
}

type CollectibleStateRejected struct {
	Reason core.DomainError
}

func (CollectibleStateAccepted) collectibleStateResult() {}

func (CollectibleStateRejected) collectibleStateResult() {}

func ParseCollectibleState(raw string) CollectibleStateResult {
	switch raw {
	case CollectibleStateMinted.value:
		return CollectibleStateAccepted{Value: CollectibleStateMinted}
	case CollectibleStateEscrowed.value:
		return CollectibleStateAccepted{Value: CollectibleStateEscrowed}
	case CollectibleStateAwarded.value:
		return CollectibleStateAccepted{Value: CollectibleStateAwarded}
	default:
		return CollectibleStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "collectible state is invalid")}
	}
}

func (state CollectibleState) String() string {
	return state.value
}

// CollectibleName is a human-readable collectible name.
type CollectibleName struct {
	value string
}

type CollectibleNameResult interface {
	collectibleNameResult()
}

type CollectibleNameAccepted struct {
	Value CollectibleName
}

type CollectibleNameRejected struct {
	Reason core.DomainError
}

func (CollectibleNameAccepted) collectibleNameResult() {}

func (CollectibleNameRejected) collectibleNameResult() {}

func NewCollectibleName(raw string) CollectibleNameResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return CollectibleNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible name is required")}
	}
	if len(trimmed) > 120 {
		return CollectibleNameRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible name is too long")}
	}
	return CollectibleNameAccepted{Value: CollectibleName{value: trimmed}}
}

func (name CollectibleName) String() string {
	return name.value
}

// Collectible is a platform-issued, non-fungible asset. It is owned by a user, a
// team, or an organization (OwnerKind disambiguates the OwnerID).
type Collectible struct {
	ID        core.CollectibleID
	Name      CollectibleName
	Kind      CollectibleKind
	State     CollectibleState
	Policy    TransferPolicy
	OwnerKind string
	OwnerID   string
	Art       string
}

// CollectibleOwnerKindUser/Team/Organization are the owner-kind tags.
const (
	CollectibleOwnerKindUser         = "user"
	CollectibleOwnerKindTeam         = "team"
	CollectibleOwnerKindOrganization = "organization"
)

// ValidCollectibleOwnerKind reports whether a raw owner-kind tag is recognized.
func ValidCollectibleOwnerKind(kind string) bool {
	switch kind {
	case CollectibleOwnerKindUser, CollectibleOwnerKindTeam, CollectibleOwnerKindOrganization:
		return true
	default:
		return false
	}
}
