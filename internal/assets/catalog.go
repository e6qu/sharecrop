package assets

import (
	"strings"

	"github.com/e6qu/sharecrop/internal/core"
)

// The collectible catalog lives in the database (collectible_catalog_entries,
// seeded by migration 000051 with the 25 default entries). This file holds
// the catalog's domain types and the fixed art registry.

// SpriteSlugs is the fixed art registry: the pixel-sprite slugs the client
// can render (Sharecrop.Sprites in web/elm mirrors this list by hand — the
// slugs are part of the API contract and must stay in sync; an unknown slug
// renders as a neutral placeholder). Catalog entries may only reference art
// from this registry.
var SpriteSlugs = []string{
	"harvest-star", "golden-sickle", "seedling", "sun-token", "rain-drop",
	"wheat-sheaf", "red-barn", "scarecrow", "honey-pot", "pumpkin",
	"apple", "carrot", "beehive", "windmill", "tractor",
	"silver-plow", "golden-egg", "prize-cow", "lucky-clover", "full-moon-harvest",
	"cornucopia", "first-harvest-trophy", "founders-seed", "rainbow-field", "golden-combine",
}

// KnownSpriteSlug reports whether the art slug names a sprite in the fixed
// registry.
func KnownSpriteSlug(slug string) bool {
	for _, known := range SpriteSlugs {
		if known == slug {
			return true
		}
	}
	return false
}

// CatalogSlug is a catalog entry's stable identifier: lowercase letters,
// digits, and single dashes.
type CatalogSlug struct {
	value string
}

type CatalogSlugResult interface {
	catalogSlugResult()
}

type CatalogSlugAccepted struct {
	Value CatalogSlug
}

type CatalogSlugRejected struct {
	Reason core.DomainError
}

func (CatalogSlugAccepted) catalogSlugResult() {}

func (CatalogSlugRejected) catalogSlugResult() {}

func NewCatalogSlug(raw string) CatalogSlugResult {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return CatalogSlugRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "catalog slug is required")}
	}
	if len(trimmed) > 64 {
		return CatalogSlugRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "catalog slug is too long")}
	}
	for index := 0; index < len(trimmed); index++ {
		character := trimmed[index]
		isLower := character >= 'a' && character <= 'z'
		isDigit := character >= '0' && character <= '9'
		if !isLower && !isDigit && character != '-' {
			return CatalogSlugRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "catalog slug may contain only lowercase letters, digits, and dashes")}
		}
	}
	if strings.HasPrefix(trimmed, "-") || strings.HasSuffix(trimmed, "-") || strings.Contains(trimmed, "--") {
		return CatalogSlugRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "catalog slug dashes must separate words")}
	}
	return CatalogSlugAccepted{Value: CatalogSlug{value: trimmed}}
}

func (slug CatalogSlug) String() string {
	return slug.value
}

// CatalogEntryState is the catalog entry lifecycle: available (awardable) or
// withdrawn (no longer awardable; existing instances are unaffected).
type CatalogEntryState struct {
	value string
}

var (
	CatalogEntryStateAvailable = CatalogEntryState{value: "available"}
	CatalogEntryStateWithdrawn = CatalogEntryState{value: "withdrawn"}
)

type CatalogEntryStateResult interface {
	catalogEntryStateResult()
}

type CatalogEntryStateAccepted struct {
	Value CatalogEntryState
}

type CatalogEntryStateRejected struct {
	Reason core.DomainError
}

func (CatalogEntryStateAccepted) catalogEntryStateResult() {}

func (CatalogEntryStateRejected) catalogEntryStateResult() {}

func ParseCatalogEntryState(raw string) CatalogEntryStateResult {
	switch raw {
	case CatalogEntryStateAvailable.value:
		return CatalogEntryStateAccepted{Value: CatalogEntryStateAvailable}
	case CatalogEntryStateWithdrawn.value:
		return CatalogEntryStateAccepted{Value: CatalogEntryStateWithdrawn}
	default:
		return CatalogEntryStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "catalog entry state is invalid")}
	}
}

func (state CatalogEntryState) String() string {
	return state.value
}

// EditionCap bounds how many instances of a catalog entry may ever be
// minted: explicitly uncapped (badges), or capped at a positive limit
// (uniques are capped at 1, editions at their declared run size).
type EditionCap interface {
	editionCap()
}

type NoEditionCap struct{}

type EditionCapOf struct {
	Limit int64
}

func (NoEditionCap) editionCap() {}

func (EditionCapOf) editionCap() {}

// CatalogEntry is one catalog collectible template. Awarding mints a fresh
// instance owned by the recipient (many holders can carry the same badge;
// editions are numbered against the cap; uniques allow one live instance).
type CatalogEntry struct {
	Slug   CatalogSlug
	Name   CollectibleName
	Kind   CollectibleKind
	Policy TransferPolicy
	// Art names a sprite from the fixed art registry (SpriteSlugs).
	Art   string
	State CatalogEntryState
	Cap   EditionCap
}

// CatalogListing pairs one catalog entry with its live-instance count for
// catalog reads.
type CatalogListing struct {
	Entry CatalogEntry
	// LiveInstanceCount counts the entry's instances that are not withdrawn
	// (the number circulating right now; for uniques this is 0 or 1).
	LiveInstanceCount int64
}

type CatalogListResult interface {
	catalogListResult()
}

type CatalogListed struct {
	Values []CatalogListing
}

type CatalogListRejected struct {
	Reason core.DomainError
}

func (CatalogListed) catalogListResult() {}

func (CatalogListRejected) catalogListResult() {}

type CatalogMutationResult interface {
	catalogMutationResult()
}

type CatalogMutated struct {
	Value CatalogEntry
}

type CatalogMutationRejected struct {
	Reason core.DomainError
}

func (CatalogMutated) catalogMutationResult() {}

func (CatalogMutationRejected) catalogMutationResult() {}
