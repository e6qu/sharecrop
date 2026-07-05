package wasmdemo

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

func testLabel(t *testing.T, raw string) agent.Label {
	t.Helper()
	result := agent.NewLabel(raw)
	accepted, matched := result.(agent.LabelAccepted)
	if !matched {
		t.Fatalf("new label %q failed", raw)
	}
	return accepted.Value
}

func TestAgentBrowserStoreCreateAndVerify(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	createResult := service.Create(ctx, owner, testLabel(t, "CI token"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil)
	created, matched := createResult.(agent.CredentialCreated)
	if !matched {
		t.Fatalf("create: want CredentialCreated, got %#v", createResult)
	}

	verifyResult := service.Verify(ctx, created.Secret)
	verified, matched := verifyResult.(agent.CredentialVerified)
	if !matched {
		t.Fatalf("verify: want CredentialVerified, got %#v", verifyResult)
	}
	if verified.Subject.ID != owner {
		t.Fatalf("verified subject = %v, want %v", verified.Subject.ID, owner)
	}
}

func TestAgentBrowserStoreVerifyRejectsUnknownSecret(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()

	unknownSecret := agent.NewSecretPlain().(agent.SecretPlainAccepted).Value
	verifyResult := service.Verify(ctx, unknownSecret)
	if _, matched := verifyResult.(agent.VerifyRejected); !matched {
		t.Fatalf("verify unknown secret: want VerifyRejected, got %#v", verifyResult)
	}
}

func TestAgentBrowserStoreVerifyRejectsExpiredCredential(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	past := time.Now().Add(-1 * time.Hour)
	created := service.Create(ctx, owner, testLabel(t, "Expired token"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), &past, nil).(agent.CredentialCreated)

	verifyResult := service.Verify(ctx, created.Secret)
	if _, matched := verifyResult.(agent.VerifyRejected); !matched {
		t.Fatalf("verify expired credential: want VerifyRejected, got %#v", verifyResult)
	}
}

func TestAgentBrowserStoreVerifyRejectsRevokedCredential(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")

	created := service.Create(ctx, owner, testLabel(t, "Revoke me"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil).(agent.CredentialCreated)

	revokeResult := service.Revoke(ctx, owner, created.Value.ID)
	if _, matched := revokeResult.(agent.CredentialRevoked); !matched {
		t.Fatalf("revoke: want CredentialRevoked, got %#v", revokeResult)
	}

	verifyResult := service.Verify(ctx, created.Secret)
	if _, matched := verifyResult.(agent.VerifyRejected); !matched {
		t.Fatalf("verify revoked credential: want VerifyRejected, got %#v", verifyResult)
	}
}

func TestAgentBrowserStoreRevokeRejectsWrongOwner(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	other := testUserID(t, "other")

	created := service.Create(ctx, owner, testLabel(t, "Mine"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil).(agent.CredentialCreated)

	revokeResult := service.Revoke(ctx, other, created.Value.ID)
	if _, matched := revokeResult.(agent.RevokeRejected); !matched {
		t.Fatalf("revoke by wrong owner: want RevokeRejected, got %#v", revokeResult)
	}
}

func TestAgentBrowserStoreListCredentials(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	other := testUserID(t, "other")

	first := service.Create(ctx, owner, testLabel(t, "First"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil).(agent.CredentialCreated)
	second := service.Create(ctx, owner, testLabel(t, "Second"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil).(agent.CredentialCreated)
	service.Create(ctx, other, testLabel(t, "Someone else's"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil, nil)

	listResult := service.List(ctx, owner, testPage(t, 10, 0))
	listed, matched := listResult.(agent.CredentialsListed)
	if !matched {
		t.Fatalf("list: want CredentialsListed, got %#v", listResult)
	}
	if len(listed.Values) != 2 {
		t.Fatalf("listed count = %d, want 2", len(listed.Values))
	}
	// Oldest first, matching internal/db's created_at ordering.
	if listed.Values[0].ID != first.Value.ID || listed.Values[1].ID != second.Value.ID {
		t.Fatalf("list order = [%v, %v], want [%v, %v]", listed.Values[0].ID, listed.Values[1].ID, first.Value.ID, second.Value.ID)
	}
}

func TestAgentBrowserStoreTaskScopedCredential(t *testing.T) {
	store := NewAgentBrowserStore(newTestBrowserStorage())
	service := agent.NewService(store)
	ctx := context.Background()
	owner := testUserID(t, "owner")
	taskIDResult := core.NewTaskID()
	taskID := taskIDResult.(core.TaskIDCreated).Value

	created := service.Create(ctx, owner, testLabel(t, "Task worker"), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead, agent.ScopeSubmissionsWrite}), nil, &taskID).(agent.CredentialCreated)

	verifyResult := service.Verify(ctx, created.Secret).(agent.CredentialVerified)
	if verifyResult.Credential.TaskID == nil || *verifyResult.Credential.TaskID != taskID {
		t.Fatalf("verified credential task id = %v, want %v", verifyResult.Credential.TaskID, taskID)
	}
	if !verifyResult.Credential.MatchesTask(taskID) {
		t.Fatalf("credential should match its own task")
	}
	otherTaskID := core.NewTaskID().(core.TaskIDCreated).Value
	if verifyResult.Credential.MatchesTask(otherTaskID) {
		t.Fatalf("credential should not match a different task")
	}
}
