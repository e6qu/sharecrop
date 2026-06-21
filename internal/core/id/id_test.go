package id

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewGeneratesVersion7(t *testing.T) {
	created, matched := New().(IDCreated)
	if !matched {
		t.Fatalf("New did not create an id")
	}

	parsed, err := uuid.Parse(created.Value.String())
	if err != nil {
		t.Fatalf("parse generated id: %v", err)
	}
	if parsed.Version() != 7 {
		t.Fatalf("uuid version = %d, want 7", parsed.Version())
	}
	if parsed.Variant() != uuid.RFC4122 {
		t.Fatalf("uuid variant = %v, want RFC4122", parsed.Variant())
	}
}

func TestNewGeneratesTimeOrderedIDs(t *testing.T) {
	previous := ""
	for index := 0; index < 100; index++ {
		created, matched := New().(IDCreated)
		if !matched {
			t.Fatalf("New did not create an id at index %d", index)
		}
		current := created.Value.String()
		if previous != "" && current <= previous {
			t.Fatalf("id at index %d (%s) is not greater than the previous id (%s); UUIDv7 ids must be time-ordered", index, current, previous)
		}
		previous = current
	}
}

func TestParseRejectsNonUUID(t *testing.T) {
	if _, matched := Parse("not-a-uuid").(IDRejected); !matched {
		t.Fatalf("Parse accepted a non-uuid string")
	}
}
