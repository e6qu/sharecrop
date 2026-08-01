//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/jackc/pgx/v5/pgxpool"
)

func grantStoreCommand(t *testing.T, target ledger.GrantTarget, amount int64, note string, key string) ledger.GrantStoreCommand {
	t.Helper()
	noteAccepted, matched := ledger.NewGrantNote(note).(ledger.GrantNoteAccepted)
	if !matched {
		t.Fatalf("grant note rejected")
	}
	return ledger.GrantStoreCommand{
		EntryID:        newEntryID(t),
		Target:         target,
		Amount:         creditAmount(t, amount),
		Note:           noteAccepted.Value,
		IdempotencyKey: idempotencyKey(t, key),
	}
}

// TestGrantCreditsToUser proves the admin credit grant writes one
// manual_adjustment entry crediting the user's spendable balance, records the
// note, and replays idempotently without double-crediting.
func TestGrantCreditsToUser(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)
	grantee := createUser(t, pool, "credit-grant-user")
	before := mustBalance(t, store, grantee).Spendable()

	command := grantStoreCommand(t, ledger.GrantToUser{ID: grantee}, 250, "manual goodwill grant", "credit-grant-user-1")
	granted, matched := store.GrantCredits(context.Background(), command).(ledger.CreditsGranted)
	if !matched {
		t.Fatalf("grant rejected: %#v", store.GrantCredits(context.Background(), command))
	}
	if len(granted.RecipientUserIDs) != 1 || granted.RecipientUserIDs[0] != grantee {
		t.Fatalf("recipients = %v, want the grantee", granted.RecipientUserIDs)
	}
	if got := mustBalance(t, store, grantee).Spendable(); got != before+250 {
		t.Fatalf("spendable after grant = %d, want %d", got, before+250)
	}

	var kind, note string
	if err := pool.QueryRow(context.Background(),
		"select kind, note from ledger_entries where id = $1", granted.EntryID.String()).Scan(&kind, &note); err != nil {
		t.Fatalf("read grant entry: %v", err)
	}
	if kind != "manual_adjustment" {
		t.Fatalf("entry kind = %q, want manual_adjustment", kind)
	}
	if note != "manual goodwill grant" {
		t.Fatalf("entry note = %q, want the grant note", note)
	}

	// Replaying the same idempotency key (fresh entry id, same command
	// otherwise) returns the original entry and does not credit again.
	replay := grantStoreCommand(t, ledger.GrantToUser{ID: grantee}, 250, "manual goodwill grant", "credit-grant-user-1")
	replayed, replayMatched := store.GrantCredits(context.Background(), replay).(ledger.CreditsGranted)
	if !replayMatched {
		t.Fatalf("grant replay rejected")
	}
	if replayed.EntryID != granted.EntryID {
		t.Fatalf("replayed entry id = %s, want the original %s", replayed.EntryID.String(), granted.EntryID.String())
	}
	if got := mustBalance(t, store, grantee).Spendable(); got != before+250 {
		t.Fatalf("spendable after replay = %d, want %d (no double credit)", got, before+250)
	}

	// The same key with a different amount is a different command and is rejected.
	if _, rejected := store.GrantCredits(context.Background(), grantStoreCommand(t, ledger.GrantToUser{ID: grantee}, 99, "different", "credit-grant-user-1")).(ledger.GrantRejected); !rejected {
		t.Fatalf("mismatched replay was not rejected")
	}
}

func addOrganizationMember(t *testing.T, pool *pgxpool.Pool, organizationID core.OrganizationID, userID core.UserID, role string) {
	t.Helper()
	membershipID, matched := core.NewOrganizationMembershipID().(core.OrganizationMembershipIDCreated)
	if !matched {
		t.Fatalf("membership id rejected")
	}
	if _, err := pool.Exec(context.Background(), `
		insert into organization_memberships (id, organization_id, user_id, status)
		values ($1, $2, $3, 'active')
	`, membershipID.Value.String(), organizationID.String(), userID.String()); err != nil {
		t.Fatalf("insert membership: %v", err)
	}
	if _, err := pool.Exec(context.Background(), `
		insert into organization_membership_roles (membership_id, role)
		values ($1, $2)
	`, membershipID.Value.String(), role); err != nil {
		t.Fatalf("insert membership role: %v", err)
	}
}

// TestGrantCreditsToOrganization proves an organization grant credits the
// organization account and resolves the owner/admin/billing members (not
// plain members) as notification recipients.
func TestGrantCreditsToOrganization(t *testing.T) {
	pool := newPool(t)
	store := db.NewLedgerStore(pool)
	organizationID := createOrganization(t, pool, "credit-grant-org")
	billingUser := createUser(t, pool, "credit-grant-billing")
	plainMember := createUser(t, pool, "credit-grant-member")
	addOrganizationMember(t, pool, organizationID, billingUser, "billing")
	addOrganizationMember(t, pool, organizationID, plainMember, "member")

	var beforeSpendable int64
	if err := pool.QueryRow(context.Background(), `
		select coalesce(sum(ledger_entries.amount), 0)
		from ledger_entries
		join credit_accounts on credit_accounts.id = ledger_entries.account_id
		where credit_accounts.organization_id = $1
	`, organizationID.String()).Scan(&beforeSpendable); err != nil {
		t.Fatalf("read organization balance: %v", err)
	}

	command := grantStoreCommand(t, ledger.GrantToOrganization{ID: organizationID}, 400, "quarterly budget top-up", "credit-grant-org-1")
	granted, matched := store.GrantCredits(context.Background(), command).(ledger.CreditsGranted)
	if !matched {
		t.Fatalf("organization grant rejected")
	}

	recipientSet := map[string]int{}
	for _, recipient := range granted.RecipientUserIDs {
		recipientSet[recipient.String()]++
	}
	if recipientSet[billingUser.String()] != 1 {
		t.Fatalf("recipients %v missed the billing member", granted.RecipientUserIDs)
	}
	if recipientSet[plainMember.String()] != 0 {
		t.Fatalf("recipients %v must not include a plain member", granted.RecipientUserIDs)
	}
	// The organization creator holds the owner role and is included too.
	if len(granted.RecipientUserIDs) != 2 {
		t.Fatalf("recipient count = %d, want 2 (owner + billing)", len(granted.RecipientUserIDs))
	}

	balance, balanceMatched := store.OrganizationBalance(context.Background(), organizationID).(ledger.BalanceFound)
	if !balanceMatched {
		t.Fatalf("organization balance rejected")
	}
	if balance.Value.Spendable() != beforeSpendable+400 {
		t.Fatalf("organization spendable = %d, want %d", balance.Value.Spendable(), beforeSpendable+400)
	}
}
