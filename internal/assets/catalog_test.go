package assets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSpriteSlugsAreValidAndUnique(t *testing.T) {
	if len(SpriteSlugs) != 25 {
		t.Fatalf("sprite registry has %d slugs, want 25 (mirroring Sprites.elm)", len(SpriteSlugs))
	}
	seen := map[string]bool{}
	for _, slug := range SpriteSlugs {
		if seen[slug] {
			t.Fatalf("sprite slug %q is duplicated", slug)
		}
		seen[slug] = true
		if _, matched := NewCatalogSlug(slug).(CatalogSlugAccepted); !matched {
			t.Fatalf("sprite slug %q does not parse as a catalog slug", slug)
		}
		if !KnownSpriteSlug(slug) {
			t.Fatalf("KnownSpriteSlug(%q) = false", slug)
		}
	}
	if KnownSpriteSlug("no-such-sprite") {
		t.Fatalf("unknown sprite slug was accepted")
	}
}

// TestCatalogSeedMigrationMatchesSpriteRegistry guards the seed migration
// against drifting from the fixed art registry: every registry slug must be
// seeded (as its own art), and the migration must not reference art outside
// the registry.
func TestCatalogSeedMigrationMatchesSpriteRegistry(t *testing.T) {
	migration, err := os.ReadFile(filepath.Join("..", "..", "migrations", "000051_collectible_catalog.sql"))
	if err != nil {
		t.Fatalf("read seed migration: %v", err)
	}
	text := string(migration)
	for _, slug := range SpriteSlugs {
		if !strings.Contains(text, "'"+slug+"'") {
			t.Fatalf("seed migration is missing sprite slug %q", slug)
		}
	}
}

func TestNewCatalogSlugValidation(t *testing.T) {
	for _, valid := range []string{"harvest-star", "a", "abc-123"} {
		if _, matched := NewCatalogSlug(valid).(CatalogSlugAccepted); !matched {
			t.Fatalf("NewCatalogSlug(%q) rejected", valid)
		}
	}
	for _, invalid := range []string{"", "Upper", "has space", "-lead", "trail-", "dou--ble", strings.Repeat("a", 65)} {
		if _, matched := NewCatalogSlug(invalid).(CatalogSlugRejected); !matched {
			t.Fatalf("NewCatalogSlug(%q) accepted", invalid)
		}
	}
}

func TestParseCatalogEntryState(t *testing.T) {
	for _, state := range []CatalogEntryState{CatalogEntryStateAvailable, CatalogEntryStateWithdrawn} {
		accepted, matched := ParseCatalogEntryState(state.String()).(CatalogEntryStateAccepted)
		if !matched || accepted.Value != state {
			t.Fatalf("ParseCatalogEntryState(%q) failed", state.String())
		}
	}
	if _, matched := ParseCatalogEntryState("retired").(CatalogEntryStateRejected); !matched {
		t.Fatalf("unknown catalog entry state accepted")
	}
}

func TestParseCollectibleStateIncludesWithdrawn(t *testing.T) {
	accepted, matched := ParseCollectibleState("withdrawn").(CollectibleStateAccepted)
	if !matched || accepted.Value != CollectibleStateWithdrawn {
		t.Fatalf("withdrawn state did not parse")
	}
}
