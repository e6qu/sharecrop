//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrgCredentialCreateVerifyListRevoke(t *testing.T) {
	pool := newPool(t)
	organizationID := createOrganization(t, pool, "orgcred-owner")
	store := db.NewOrgCredentialStore(pool)

	secret := mustOrgSecret(t)
	credential := orgcred.Credential{
		ID:             newOrgCredentialID(t),
		OrganizationID: organizationID,
		Label:          mustLabel(t, "Integration org token"),
		Scopes:         agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeOrgManage}),
		State:          agent.StateActive,
	}

	if _, matched := store.CreateCredential(context.Background(), credential, secret.Hash()).(orgcred.CreateStoreAccepted); !matched {
		t.Fatalf("create credential rejected")
	}

	verified, matched := store.VerifyCredential(context.Background(), secret.Hash()).(orgcred.VerifyStoreFound)
	if !matched {
		t.Fatalf("verify credential rejected")
	}
	if verified.Value.OrganizationID != organizationID {
		t.Fatalf("verified organization mismatch")
	}
	if len(verified.Value.Scopes.Values()) != 2 {
		t.Fatalf("scope count = %d, want 2", len(verified.Value.Scopes.Values()))
	}
	if _, granted := verified.Value.Scopes.Allows(agent.ScopeOrgManage).(agent.ScopeGranted); !granted {
		t.Fatalf("org_manage scope missing after round trip")
	}

	listed, listedMatched := store.ListCredentials(context.Background(), organizationID, core.DefaultPage()).(orgcred.ListStoreListed)
	if !listedMatched {
		t.Fatalf("list credentials rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("credential count = %d, want 1", len(listed.Values))
	}

	revoked, revokedMatched := store.RevokeCredential(context.Background(), organizationID, credential.ID).(orgcred.RevokeStoreRevoked)
	if !revokedMatched {
		t.Fatalf("revoke credential rejected")
	}
	if revoked.Value.State != agent.StateRevoked {
		t.Fatalf("revoked state = %q, want revoked", revoked.Value.State.String())
	}

	// A revoked credential is still found by hash so the service can reject it explicitly.
	afterRevoke, afterMatched := store.VerifyCredential(context.Background(), secret.Hash()).(orgcred.VerifyStoreFound)
	if !afterMatched {
		t.Fatalf("verify after revoke rejected")
	}
	if afterRevoke.Value.State != agent.StateRevoked {
		t.Fatalf("state after revoke = %q, want revoked", afterRevoke.Value.State.String())
	}

	// Revoking again is rejected because there is no active credential.
	if _, matched := store.RevokeCredential(context.Background(), organizationID, credential.ID).(orgcred.RevokeStoreRejected); !matched {
		t.Fatalf("second revoke was not rejected")
	}
}

// createOrganization provisions a fresh organization (with a fresh creator
// user) via the real org service, for tests that just need a valid
// organization id to attach fixtures to.
func createOrganization(t *testing.T, pool *pgxpool.Pool, prefix string) core.OrganizationID {
	t.Helper()
	creator := createUser(t, pool, prefix)
	nameResult := org.NewOrganizationName(prefix + " org")
	name, matched := nameResult.(org.OrganizationNameAccepted)
	if !matched {
		t.Fatalf("organization name rejected")
	}
	service := org.NewService(db.NewOrgStore(pool))
	created, matched := service.CreateOrganization(context.Background(), auth.UserSubject{ID: creator}, name.Value).(org.OrganizationCreated)
	if !matched {
		t.Fatalf("create organization rejected")
	}
	return created.Value.ID
}

func mustOrgSecret(t *testing.T) orgcred.SecretPlain {
	t.Helper()
	accepted, matched := orgcred.NewSecretPlain().(orgcred.SecretPlainAccepted)
	if !matched {
		t.Fatalf("secret rejected")
	}
	return accepted.Value
}

func newOrgCredentialID(t *testing.T) core.OrgCredentialID {
	t.Helper()
	created, matched := core.NewOrgCredentialID().(core.OrgCredentialIDCreated)
	if !matched {
		t.Fatalf("org credential id rejected")
	}
	return created.Value
}
