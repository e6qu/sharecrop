//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/db"
)

func TestAgentCredentialCreateVerifyListRevoke(t *testing.T) {
	pool := newPool(t)
	owner := createUser(t, pool, "agent-owner")
	store := db.NewAgentStore(pool)

	secret := mustSecret(t)
	credential := agent.Credential{
		ID:     newAgentCredentialID(t),
		UserID: owner,
		Label:  mustLabel(t, "Integration agent"),
		Scopes: agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeSubmissionsWrite}),
		State:  agent.StateActive,
	}

	if _, matched := store.CreateCredential(context.Background(), credential, secret.Hash()).(agent.CreateStoreAccepted); !matched {
		t.Fatalf("create credential rejected")
	}

	verified, matched := store.VerifyCredential(context.Background(), secret.Hash()).(agent.VerifyStoreFound)
	if !matched {
		t.Fatalf("verify credential rejected")
	}
	if verified.Value.UserID != owner {
		t.Fatalf("verified owner mismatch")
	}
	if len(verified.Value.Scopes.Values()) != 2 {
		t.Fatalf("scope count = %d, want 2", len(verified.Value.Scopes.Values()))
	}
	if _, granted := verified.Value.Scopes.Allows(agent.ScopeTasksRead).(agent.ScopeGranted); !granted {
		t.Fatalf("tasks_read scope missing after round trip")
	}

	listed, listedMatched := store.ListCredentials(context.Background(), owner, core.DefaultPage()).(agent.ListStoreListed)
	if !listedMatched {
		t.Fatalf("list credentials rejected")
	}
	if len(listed.Values) != 1 {
		t.Fatalf("credential count = %d, want 1", len(listed.Values))
	}

	revoked, revokedMatched := store.RevokeCredential(context.Background(), owner, credential.ID).(agent.RevokeStoreRevoked)
	if !revokedMatched {
		t.Fatalf("revoke credential rejected")
	}
	if revoked.Value.State != agent.StateRevoked {
		t.Fatalf("revoked state = %q, want revoked", revoked.Value.State.String())
	}

	// A revoked credential is still found by hash so the service can reject it explicitly.
	afterRevoke, afterMatched := store.VerifyCredential(context.Background(), secret.Hash()).(agent.VerifyStoreFound)
	if !afterMatched {
		t.Fatalf("verify after revoke rejected")
	}
	if afterRevoke.Value.State != agent.StateRevoked {
		t.Fatalf("state after revoke = %q, want revoked", afterRevoke.Value.State.String())
	}

	// Revoking again is rejected because there is no active credential.
	if _, matched := store.RevokeCredential(context.Background(), owner, credential.ID).(agent.RevokeStoreRejected); !matched {
		t.Fatalf("second revoke was not rejected")
	}
}

func mustSecret(t *testing.T) agent.SecretPlain {
	t.Helper()
	accepted, matched := agent.NewSecretPlain().(agent.SecretPlainAccepted)
	if !matched {
		t.Fatalf("secret rejected")
	}
	return accepted.Value
}

func mustLabel(t *testing.T, raw string) agent.Label {
	t.Helper()
	accepted, matched := agent.NewLabel(raw).(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	return accepted.Value
}

func newAgentCredentialID(t *testing.T) core.AgentCredentialID {
	t.Helper()
	created, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	return created.Value
}
