package agentbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/agent/agenttest"
	"github.com/e6qu/sharecrop/internal/core"
)

func sampleCredential(t *testing.T, expiresAt *time.Time) agent.Credential {
	t.Helper()
	id, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	userID, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	label, matched := agent.NewLabel("test-label").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scope, matched := agent.ParseScope("tasks_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	return agent.Credential{
		ID:        id.Value,
		UserID:    userID.Value,
		Label:     label.Value,
		Scopes:    agent.NewScopeSet([]agent.Scope{scope.Value}),
		State:     agent.StateActive,
		ExpiresAt: expiresAt,
		TaskID:    nil,
	}
}

func assertCredentialEqual(t *testing.T, got, want agent.Credential) {
	t.Helper()
	if diff := agenttest.CredentialDiff(got, want); diff != "" {
		t.Errorf("credential mismatch: %s", diff)
	}
}

func TestCredentialRoundTrip(t *testing.T) {
	expires := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	for _, credential := range []agent.Credential{sampleCredential(t, &expires), sampleCredential(t, nil)} {
		restored, err := decodeCredential(encodeCredential(credential))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertCredentialEqual(t, restored, credential)
	}
}

func TestCreateResultRoundTrip(t *testing.T) {
	accepted, err := decodeCreateResult(encodeCreateResult(agent.CreateStoreAccepted{}))
	if err != nil {
		t.Fatalf("decode accepted: %v", err)
	}
	if _, matched := accepted.(agent.CreateStoreAccepted); !matched {
		t.Errorf("accepted result = %T", accepted)
	}
	rejected, err := decodeCreateResult(encodeCreateResult(agent.CreateStoreRejected{
		Reason: core.NewDomainError(core.ErrorCodeConflict, "dup"),
	}))
	if err != nil {
		t.Fatalf("decode rejected: %v", err)
	}
	if typed, matched := rejected.(agent.CreateStoreRejected); !matched || typed.Reason.Code() != core.ErrorCodeConflict {
		t.Errorf("create rejection not preserved: %T", rejected)
	}
}

func TestVerifyRevokeListRoundTrip(t *testing.T) {
	credential := sampleCredential(t, nil)

	verified, err := decodeVerifyResult(encodeVerifyResult(agent.VerifyStoreFound{Value: credential}))
	if err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if typed, matched := verified.(agent.VerifyStoreFound); !matched {
		t.Fatalf("verify result = %T", verified)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	revoked, err := decodeRevokeResult(encodeRevokeResult(agent.RevokeStoreRevoked{Value: credential}))
	if err != nil {
		t.Fatalf("decode revoke: %v", err)
	}
	if typed, matched := revoked.(agent.RevokeStoreRevoked); !matched {
		t.Fatalf("revoke result = %T", revoked)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	listed, err := decodeListResult(encodeListResult(agent.ListStoreListed{Values: []agent.Credential{credential}}))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if typed, matched := listed.(agent.ListStoreListed); !matched || len(typed.Values) != 1 {
		t.Fatalf("list result = %T", listed)
	} else {
		assertCredentialEqual(t, typed.Values[0], credential)
	}
}
