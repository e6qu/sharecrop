//go:build integration

package integration_test

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/event"
	"github.com/e6qu/sharecrop/internal/ledger"
)

// testEventDraft builds a minimal draft for store mutations whose command or
// signature carries one; store-level tests that assert nothing about
// emissions still have to record a well-formed outbox row.
func testEventDraft(t *testing.T, kind event.Kind, actor core.UserID) event.Draft {
	t.Helper()
	created, matched := event.NewDraft(kind, event.ActorUser{ID: actor}, event.NoSubjectRefs(), event.EmptyMetadata(), event.NewRecipients(actor)).(event.DraftCreated)
	if !matched {
		t.Fatalf("event draft rejected")
	}
	return created.Value
}

// grantDraftActor picks a stable actor for grant-command drafts: the target
// user for user grants, the durable system actor otherwise (the tests using
// these commands assert ledger arithmetic, not event audiences).
func grantDraftActor(t *testing.T, target ledger.GrantTarget) core.UserID {
	t.Helper()
	if user, matched := target.(ledger.GrantToUser); matched {
		return user.ID
	}
	return core.SystemUserID()
}
