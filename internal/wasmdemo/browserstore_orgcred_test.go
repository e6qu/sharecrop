package wasmdemo

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
)

func testOrganizationID(t *testing.T, label string) core.OrganizationID {
	t.Helper()
	result := core.NewOrganizationID()
	created, matched := result.(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("new organization id for %q failed", label)
	}
	return created.Value
}

func TestOrgCredentialBrowserStoreCreateAndVerify(t *testing.T) {
	store := NewOrgCredentialBrowserStore(newTestBrowserStorage())
	service := orgcred.NewService(store)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "org")

	createResult := service.Create(ctx, organizationID, testLabel(t, "Org CI token"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil)
	created, matched := createResult.(orgcred.CredentialCreated)
	if !matched {
		t.Fatalf("create: want CredentialCreated, got %#v", createResult)
	}

	verifyResult := service.Verify(ctx, created.Secret)
	verified, matched := verifyResult.(orgcred.CredentialVerified)
	if !matched {
		t.Fatalf("verify: want CredentialVerified, got %#v", verifyResult)
	}
	if verified.Subject.ID != organizationID {
		t.Fatalf("verified subject = %v, want %v", verified.Subject.ID, organizationID)
	}
}

func TestOrgCredentialBrowserStoreVerifyRejectsRevoked(t *testing.T) {
	store := NewOrgCredentialBrowserStore(newTestBrowserStorage())
	service := orgcred.NewService(store)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "org")

	created := service.Create(ctx, organizationID, testLabel(t, "Revoke me"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil).(orgcred.CredentialCreated)

	revokeResult := service.Revoke(ctx, organizationID, created.Value.ID)
	if _, matched := revokeResult.(orgcred.CredentialRevoked); !matched {
		t.Fatalf("revoke: want CredentialRevoked, got %#v", revokeResult)
	}

	verifyResult := service.Verify(ctx, created.Secret)
	if _, matched := verifyResult.(orgcred.VerifyRejected); !matched {
		t.Fatalf("verify revoked credential: want VerifyRejected, got %#v", verifyResult)
	}
}

func TestOrgCredentialBrowserStoreRevokeRejectsWrongOrganization(t *testing.T) {
	store := NewOrgCredentialBrowserStore(newTestBrowserStorage())
	service := orgcred.NewService(store)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "org")
	otherOrganizationID := testOrganizationID(t, "other org")

	created := service.Create(ctx, organizationID, testLabel(t, "Mine"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil).(orgcred.CredentialCreated)

	revokeResult := service.Revoke(ctx, otherOrganizationID, created.Value.ID)
	if _, matched := revokeResult.(orgcred.RevokeRejected); !matched {
		t.Fatalf("revoke by wrong organization: want RevokeRejected, got %#v", revokeResult)
	}
}

func TestOrgCredentialBrowserStoreListCredentials(t *testing.T) {
	store := NewOrgCredentialBrowserStore(newTestBrowserStorage())
	service := orgcred.NewService(store)
	ctx := context.Background()
	organizationID := testOrganizationID(t, "org")
	otherOrganizationID := testOrganizationID(t, "other org")

	first := service.Create(ctx, organizationID, testLabel(t, "First"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil).(orgcred.CredentialCreated)
	second := service.Create(ctx, organizationID, testLabel(t, "Second"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil).(orgcred.CredentialCreated)
	service.Create(ctx, otherOrganizationID, testLabel(t, "Someone else's org"), agent.NewScopeSet([]agent.Scope{agent.ScopeOrgRead}), nil)

	listResult := service.List(ctx, organizationID, testPage(t, 10, 0))
	listed, matched := listResult.(orgcred.CredentialsListed)
	if !matched {
		t.Fatalf("list: want CredentialsListed, got %#v", listResult)
	}
	if len(listed.Values) != 2 {
		t.Fatalf("listed count = %d, want 2", len(listed.Values))
	}
	if listed.Values[0].ID != first.Value.ID || listed.Values[1].ID != second.Value.ID {
		t.Fatalf("list order = [%v, %v], want [%v, %v]", listed.Values[0].ID, listed.Values[1].ID, first.Value.ID, second.Value.ID)
	}
}
