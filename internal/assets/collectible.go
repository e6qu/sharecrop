package assets

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/auth"
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
	// CollectibleStateWithdrawn: an admin removed the instance from its
	// holder. Withdrawn instances are not transferable, tippable, awardable,
	// or escrowable, and are the only instances an admin may hard-delete.
	CollectibleStateWithdrawn = CollectibleState{value: "withdrawn"}
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
	case CollectibleStateWithdrawn.value:
		return CollectibleStateAccepted{Value: CollectibleStateWithdrawn}
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
	ID             core.CollectibleID
	Name           CollectibleName
	Kind           CollectibleKind
	State          CollectibleState
	Policy         TransferPolicy
	OwnerKind      string
	OwnerID        string
	OrganizationID string
	Art            string
	// Catalog ties an awarded copy to its catalog entry; custom mints carry
	// no catalog reference.
	Catalog CatalogRef
	// Edition is the mint sequence number for edition-kind instances.
	Edition EditionRef
	// Issuer is the acting user who minted or awarded the instance;
	// pre-provenance rows carry no issuer.
	Issuer IssuerRef
	// IssuerDisplayName names the issuer for read models. It is resolved on
	// the list read paths (the same convention as task creator names);
	// mutation results and issuerless rows leave it zero.
	IssuerDisplayName auth.DisplayName
	// OwnerDisplayName labels the owner for read models: the user's display
	// name, the organization's name, or the team's name, per OwnerKind. It is
	// resolved on the list read paths; mutation results leave it zero.
	OwnerDisplayName auth.DisplayName
}

// CatalogRef is a collectible instance's link to its catalog entry, or its
// explicit absence (a custom mint).
type CatalogRef interface {
	catalogRef()
}

type NoCatalogRef struct{}

type FromCatalog struct {
	Slug CatalogSlug
}

func (NoCatalogRef) catalogRef() {}

func (FromCatalog) catalogRef() {}

// EditionRef is an edition instance's mint sequence number, or its explicit
// absence (badges and uniques are not numbered).
type EditionRef interface {
	editionRef()
}

type NoEditionNumber struct{}

type EditionNumbered struct {
	Number int64
}

func (NoEditionNumber) editionRef() {}

func (EditionNumbered) editionRef() {}

// IssuerRef is the user who minted or awarded the instance, or its explicit
// absence (rows created before issuer provenance existed).
type IssuerRef interface {
	issuerRef()
}

type NoIssuer struct{}

type IssuedBy struct {
	ID core.UserID
}

func (NoIssuer) issuerRef() {}

func (IssuedBy) issuerRef() {}

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
