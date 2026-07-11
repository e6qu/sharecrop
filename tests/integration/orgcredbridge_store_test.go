//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/orgcred/orgcredtest"
	"github.com/e6qu/sharecrop/internal/wasibridge/orgcredbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestOrgCredentialBridgeDualRun exercises the organization-credential store
// through both the direct-db path and the compiled wasip1 guest + host bridge.
// It shares agent's Label/ScopeSet/State codecs (via agentwire), so it also
// checks that the extracted shared codecs work for a second store.
func TestOrgCredentialBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewOrgCredentialStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return orgcredbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := orgcredbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	organizationID := createOrganization(t, pool, "orgcredbridge")
	page := requirePage(t, 50, 0)

	credential := buildOrgCredential(t, organizationID)
	hash := orgcred.SecretHashFromString("orgcred-hash-" + credential.ID.String())

	t.Run("create through the bridge, then verify matches a direct call", func(t *testing.T) {
		if _, matched := bridgeStore.CreateCredential(ctx, credential, hash).(orgcred.CreateStoreAccepted); !matched {
			t.Fatalf("bridge CreateCredential did not accept")
		}
		viaBridge := requireOrgCredentialVerified(t, bridgeStore.VerifyCredential(ctx, hash))
		direct := requireOrgCredentialVerified(t, dbStore.VerifyCredential(ctx, hash))
		if diff := orgcredtest.CredentialDiff(viaBridge, direct); diff != "" {
			t.Errorf("verify mismatch: %s", diff)
		}
		if viaBridge.ID != credential.ID {
			t.Errorf("verified id = %s, want %s", viaBridge.ID, credential.ID)
		}
	})

	t.Run("list matches a direct call", func(t *testing.T) {
		viaBridge := requireOrgCredentialsListed(t, bridgeStore.ListCredentials(ctx, organizationID, page))
		direct := requireOrgCredentialsListed(t, dbStore.ListCredentials(ctx, organizationID, page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("list counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := orgcredtest.CredentialDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("list mismatch: %s", diff)
		}
	})

	t.Run("revoke through the bridge", func(t *testing.T) {
		revoked, matched := bridgeStore.RevokeCredential(ctx, organizationID, credential.ID).(orgcred.RevokeStoreRevoked)
		if !matched {
			t.Fatalf("bridge RevokeCredential did not report revoked")
		}
		if revoked.Value.ID != credential.ID {
			t.Errorf("revoked id = %s, want %s", revoked.Value.ID, credential.ID)
		}
		if revoked.Value.State.String() != agent.StateRevoked.String() {
			t.Errorf("revoked state = %s, want revoked", revoked.Value.State)
		}
	})
}

func buildOrgCredential(t *testing.T, organizationID core.OrganizationID) orgcred.Credential {
	t.Helper()
	id, matched := core.NewOrgCredentialID().(core.OrgCredentialIDCreated)
	if !matched {
		t.Fatalf("org credential id rejected")
	}
	label, matched := agent.NewLabel("integration-orgcred").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scope, matched := agent.ParseScope("org_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	return orgcred.Credential{
		ID:             id.Value,
		OrganizationID: organizationID,
		Label:          label.Value,
		Scopes:         agent.NewScopeSet([]agent.Scope{scope.Value}),
		State:          agent.StateActive,
	}
}

func requireOrgCredentialVerified(t *testing.T, result orgcred.VerifyStoreResult) orgcred.Credential {
	t.Helper()
	found, matched := result.(orgcred.VerifyStoreFound)
	if !matched {
		t.Fatalf("verify result = %T, want VerifyStoreFound", result)
	}
	return found.Value
}

func requireOrgCredentialsListed(t *testing.T, result orgcred.ListStoreResult) []orgcred.Credential {
	t.Helper()
	listed, matched := result.(orgcred.ListStoreListed)
	if !matched {
		t.Fatalf("list result = %T, want ListStoreListed", result)
	}
	return listed.Values
}
