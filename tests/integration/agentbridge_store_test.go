//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/agent/agenttest"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
)

// TestAgentBridgeDualRun exercises the agent credential store through both the
// direct-db path and the compiled wasip1 guest + host bridge: create, verify,
// list, revoke.
func TestAgentBridgeDualRun(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t)
	dbStore := db.NewAgentStore(pool)

	guestWASM, err := compileWASIGuest(t, "github.com/e6qu/sharecrop/cmd/sharecrop-wasi-store-guest")
	if err != nil {
		t.Fatalf("compile store guest: %v", err)
	}
	host, err := rpc.NewHost(ctx, guestWASM, func(ctx context.Context, method string, args []byte) ([]byte, error) {
		return agentbridge.Dispatch(ctx, dbStore, method, args)
	})
	if err != nil {
		t.Fatalf("new host: %v", err)
	}
	t.Cleanup(func() { _ = host.Close(ctx) })
	bridgeStore := agentbridge.NewGuestStore(func(method string, args []byte) ([]byte, error) {
		return host.Call(ctx, method, args)
	})

	owner := createUser(t, pool, "agentbridge")
	page := requirePage(t, 50, 0)

	credential := buildAgentCredential(t, owner)
	hash := agent.SecretHashFromString("agent-hash-" + credential.ID.String())

	t.Run("create through the bridge, then verify matches a direct call", func(t *testing.T) {
		if _, matched := bridgeStore.CreateCredential(ctx, credential, hash).(agent.CreateStoreAccepted); !matched {
			t.Fatalf("bridge CreateCredential did not accept")
		}
		viaBridge := requireCredentialVerified(t, bridgeStore.VerifyCredential(ctx, hash))
		direct := requireCredentialVerified(t, dbStore.VerifyCredential(ctx, hash))
		if diff := agenttest.CredentialDiff(viaBridge, direct); diff != "" {
			t.Errorf("verify mismatch: %s", diff)
		}
		if viaBridge.ID != credential.ID {
			t.Errorf("verified id = %s, want %s", viaBridge.ID, credential.ID)
		}
	})

	t.Run("list matches a direct call", func(t *testing.T) {
		viaBridge := requireCredentialsListed(t, bridgeStore.ListCredentials(ctx, owner, page))
		direct := requireCredentialsListed(t, dbStore.ListCredentials(ctx, owner, page))
		if len(viaBridge) != len(direct) || len(viaBridge) != 1 {
			t.Fatalf("list counts: bridge %d, direct %d, want 1", len(viaBridge), len(direct))
		}
		if diff := agenttest.CredentialDiff(viaBridge[0], direct[0]); diff != "" {
			t.Errorf("list mismatch: %s", diff)
		}
	})

	t.Run("revoke through the bridge", func(t *testing.T) {
		revoked, matched := bridgeStore.RevokeCredential(ctx, owner, credential.ID).(agent.RevokeStoreRevoked)
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

func buildAgentCredential(t *testing.T, owner core.UserID) agent.Credential {
	t.Helper()
	id, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	label, matched := agent.NewLabel("integration-agent").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scope, matched := agent.ParseScope("tasks_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	return agent.Credential{
		ID:     id.Value,
		UserID: owner,
		Label:  label.Value,
		Scopes: agent.NewScopeSet([]agent.Scope{scope.Value}),
		State:  agent.StateActive,
	}
}

func requireCredentialVerified(t *testing.T, result agent.VerifyStoreResult) agent.Credential {
	t.Helper()
	found, matched := result.(agent.VerifyStoreFound)
	if !matched {
		t.Fatalf("verify result = %T, want VerifyStoreFound", result)
	}
	return found.Value
}

func requireCredentialsListed(t *testing.T, result agent.ListStoreResult) []agent.Credential {
	t.Helper()
	listed, matched := result.(agent.ListStoreListed)
	if !matched {
		t.Fatalf("list result = %T, want ListStoreListed", result)
	}
	return listed.Values
}
