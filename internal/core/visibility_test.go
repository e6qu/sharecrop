package core

import "testing"

func TestParseVisibilityKind(t *testing.T) {
	result := ParseVisibilityKind("organization")

	parsed, matched := result.(VisibilityKindParsed)
	if !matched {
		t.Fatalf("result = %T, want VisibilityKindParsed", result)
	}

	if parsed.Value.String() != "organization" {
		t.Fatalf("visibility = %q, want organization", parsed.Value.String())
	}
}

func TestParseVisibilityKindRejectsUnknownValue(t *testing.T) {
	result := ParseVisibilityKind("hidden")

	_, matched := result.(VisibilityKindRejected)
	if !matched {
		t.Fatalf("result = %T, want VisibilityKindRejected", result)
	}
}

func TestVisibilityScopeVariantsCompile(t *testing.T) {
	user, matched := NewUserID().(UserIDCreated)
	if !matched {
		t.Fatalf("new user id did not create")
	}

	var scope VisibilityScope = UserVisibility{UserID: user.Value}

	_, matched = scope.(UserVisibility)
	if !matched {
		t.Fatalf("scope = %T, want UserVisibility", scope)
	}
}
