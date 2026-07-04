package orgcred

import (
	"context"
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
)

func TestServiceCreateRejectsEmptyScopes(t *testing.T) {
	service := NewService(&memoryStore{})
	result := service.Create(context.Background(), newTestOrganizationID(t), testLabel(t), agent.NewScopeSet(nil), nil)
	if _, matched := result.(CreateRejected); !matched {
		t.Fatalf("empty scopes were accepted")
	}
}

func TestServiceCreateAndVerify(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	organizationID := newTestOrganizationID(t)
	created, matched := service.Create(context.Background(), organizationID, testLabel(t), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil).(CredentialCreated)
	if !matched {
		t.Fatalf("create was rejected")
	}

	verified, verifiedMatched := service.Verify(context.Background(), created.Secret).(CredentialVerified)
	if !verifiedMatched {
		t.Fatalf("verify was rejected")
	}
	if verified.Subject.ID != organizationID {
		t.Fatalf("verified subject mismatch")
	}
}

func TestServiceVerifyRejectsRevoked(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	organizationID := newTestOrganizationID(t)
	created := service.Create(context.Background(), organizationID, testLabel(t), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), nil).(CredentialCreated)
	store.revoke(created.Value.ID)

	if _, matched := service.Verify(context.Background(), created.Secret).(VerifyRejected); !matched {
		t.Fatalf("revoked credential verified")
	}
}

func TestServiceVerifyRejectsExpiredCredential(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	past := time.Now().Add(-time.Hour)
	created, matched := service.Create(context.Background(), newTestOrganizationID(t), testLabel(t), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), &past).(CredentialCreated)
	if !matched {
		t.Fatalf("create was rejected")
	}

	if _, matched := service.Verify(context.Background(), created.Secret).(VerifyRejected); !matched {
		t.Fatalf("expired credential verified")
	}
}

func TestServiceVerifyAcceptsNotYetExpiredCredential(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	future := time.Now().Add(time.Hour)
	created, matched := service.Create(context.Background(), newTestOrganizationID(t), testLabel(t), agent.NewScopeSet([]agent.Scope{agent.ScopeTasksRead}), &future).(CredentialCreated)
	if !matched {
		t.Fatalf("create was rejected")
	}

	if _, matched := service.Verify(context.Background(), created.Secret).(CredentialVerified); !matched {
		t.Fatalf("not-yet-expired credential was rejected")
	}
}

func TestSecretHasDistinctPrefixFromAgentCredential(t *testing.T) {
	created, matched := NewSecretPlain().(SecretPlainAccepted)
	if !matched {
		t.Fatalf("new secret rejected")
	}
	if !HasSecretPrefix(created.Value.String()) {
		t.Fatalf("org secret did not report its own prefix")
	}
	agentSecret, matched := agent.NewSecretPlain().(agent.SecretPlainAccepted)
	if !matched {
		t.Fatalf("new agent secret rejected")
	}
	if HasSecretPrefix(agentSecret.Value.String()) {
		t.Fatalf("agent secret was misidentified as an org secret")
	}
}

type storedCredential struct {
	credential Credential
	hash       string
}

type memoryStore struct {
	records []storedCredential
}

func (store *memoryStore) CreateCredential(_ context.Context, credential Credential, hash SecretHash) CreateStoreResult {
	store.records = append(store.records, storedCredential{credential: credential, hash: hash.String()})
	return CreateStoreAccepted{}
}

func (store *memoryStore) VerifyCredential(_ context.Context, hash SecretHash) VerifyStoreResult {
	for index := range store.records {
		if store.records[index].hash == hash.String() {
			return VerifyStoreFound{Value: store.records[index].credential}
		}
	}
	return VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential was not found")}
}

func (store *memoryStore) ListCredentials(_ context.Context, organizationID core.OrganizationID, _ core.Page) ListStoreResult {
	values := make([]Credential, 0)
	for index := range store.records {
		if store.records[index].credential.OrganizationID == organizationID {
			values = append(values, store.records[index].credential)
		}
	}
	return ListStoreListed{Values: values}
}

func (store *memoryStore) RevokeCredential(_ context.Context, organizationID core.OrganizationID, id core.OrgCredentialID) RevokeStoreResult {
	for index := range store.records {
		if store.records[index].credential.ID == id && store.records[index].credential.OrganizationID == organizationID {
			store.records[index].credential.State = agent.StateRevoked
			return RevokeStoreRevoked{Value: store.records[index].credential}
		}
	}
	return RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "org credential was not found")}
}

func (store *memoryStore) revoke(id core.OrgCredentialID) {
	for index := range store.records {
		if store.records[index].credential.ID == id {
			store.records[index].credential.State = agent.StateRevoked
		}
	}
}

func testLabel(t *testing.T) agent.Label {
	t.Helper()
	accepted, matched := agent.NewLabel("Test org token").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	return accepted.Value
}

func newTestOrganizationID(t *testing.T) core.OrganizationID {
	t.Helper()
	created, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	return created.Value
}
