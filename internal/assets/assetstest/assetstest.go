// Package assetstest holds test-support helpers for assets.Collectible, shared
// by the assets bridge's codec tests and the integration dual-run test so the
// two do not carry duplicate comparisons.
package assetstest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/assets"
)

// CollectibleDiff returns a description of the first field in which got and want
// differ, or "" if they are equal.
func CollectibleDiff(got, want assets.Collectible) string {
	switch {
	case got.ID != want.ID:
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	case got.Name.String() != want.Name.String():
		return fmt.Sprintf("name: %s != %s", got.Name, want.Name)
	case got.Kind.String() != want.Kind.String():
		return fmt.Sprintf("kind: %s != %s", got.Kind, want.Kind)
	case got.State.String() != want.State.String():
		return fmt.Sprintf("state: %s != %s", got.State, want.State)
	case got.Policy.String() != want.Policy.String():
		return fmt.Sprintf("policy: %s != %s", got.Policy, want.Policy)
	case got.OwnerKind != want.OwnerKind:
		return fmt.Sprintf("owner_kind: %s != %s", got.OwnerKind, want.OwnerKind)
	case got.OwnerID != want.OwnerID:
		return fmt.Sprintf("owner_id: %s != %s", got.OwnerID, want.OwnerID)
	case got.OrganizationID != want.OrganizationID:
		return fmt.Sprintf("organization_id: %s != %s", got.OrganizationID, want.OrganizationID)
	case got.Art != want.Art:
		return fmt.Sprintf("art: %s != %s", got.Art, want.Art)
	default:
		return ""
	}
}
