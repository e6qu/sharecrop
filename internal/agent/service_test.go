package agent

import (
	"context"
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

func TestParseScopeRoundTrips(t *testing.T) {
	scopes := []Scope{ScopeTasksRead, ScopeTasksWrite, ScopeSubmissionsWrite, ScopeSubmissionsRead, ScopeSubmissionsReview}
	for _, scope := range scopes {
		accepted, matched := ParseScope(scope.String()).(ScopeAccepted)
		if !matched {
			t.Fatalf("ParseScope(%q) rejected", scope.String())
		}
		if accepted.Value != scope {
			t.Fatalf("parsed = %q, want %q", accepted.Value.String(), scope.String())
		}
	}
}

func TestParseScopeRejectsUnknown(t *testing.T) {
	if _, matched := ParseScope("everything").(ScopeRejected); !matched {
		t.Fatalf("unknown scope was accepted")
	}
}

func TestScopeSetDeduplicatesAndChecks(t *testing.T) {
	set := NewScopeSet([]Scope{ScopeTasksRead, ScopeTasksRead, ScopeSubmissionsWrite})
	if len(set.Values()) != 2 {
		t.Fatalf("scope count = %d, want 2", len(set.Values()))
	}
	if _, matched := set.Allows(ScopeTasksRead).(ScopeGranted); !matched {
		t.Fatalf("tasks_read was not granted")
	}
	if _, matched := set.Allows(ScopeSubmissionsReview).(ScopeDenied); !matched {
		t.Fatalf("submissions_review was not denied")
	}
}

func TestNewSecretPlainRoundTripsThroughParse(t *testing.T) {
	created, matched := NewSecretPlain().(SecretPlainAccepted)
	if !matched {
		t.Fatalf("new secret rejected")
	}
	parsed, parsedMatched := ParseSecretPlain(created.Value.String()).(SecretPlainAccepted)
	if !parsedMatched {
		t.Fatalf("parse secret rejected")
	}
	if parsed.Value.Hash().String() != created.Value.Hash().String() {
		t.Fatalf("hash mismatch after parse")
	}
}

func TestParseSecretPlainRejectsForeignToken(t *testing.T) {
	if _, matched := ParseSecretPlain("not-a-sharecrop-agent-token").(SecretPlainRejected); !matched {
		t.Fatalf("foreign token was accepted")
	}
}

func TestNewLabelRejectsBlank(t *testing.T) {
	if _, matched := NewLabel("   ").(LabelRejected); !matched {
		t.Fatalf("blank label accepted")
	}
}

func TestServiceCreateRejectsEmptyScopes(t *testing.T) {
	service := NewService(&memoryStore{})
	result := service.Create(context.Background(), newTestUserID(t), testLabel(t), NewScopeSet(nil))
	if _, matched := result.(CreateRejected); !matched {
		t.Fatalf("empty scopes were accepted")
	}
}

func TestServiceCreateAndVerify(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	created, matched := service.Create(context.Background(), newTestUserID(t), testLabel(t), NewScopeSet([]Scope{ScopeTasksRead})).(CredentialCreated)
	if !matched {
		t.Fatalf("create was rejected")
	}

	verified, verifiedMatched := service.Verify(context.Background(), created.Secret).(CredentialVerified)
	if !verifiedMatched {
		t.Fatalf("verify was rejected")
	}
	if verified.Subject.ID != created.Value.UserID {
		t.Fatalf("verified subject mismatch")
	}
}

func TestServiceVerifyRejectsRevoked(t *testing.T) {
	store := &memoryStore{}
	service := NewService(store)
	created := service.Create(context.Background(), newTestUserID(t), testLabel(t), NewScopeSet([]Scope{ScopeTasksRead})).(CredentialCreated)
	store.revoke(created.Value.ID)

	if _, matched := service.Verify(context.Background(), created.Secret).(VerifyRejected); !matched {
		t.Fatalf("revoked credential verified")
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
	return VerifyStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential was not found")}
}

func (store *memoryStore) ListCredentials(_ context.Context, owner core.UserID, _ core.Page) ListStoreResult {
	values := make([]Credential, 0)
	for index := range store.records {
		if store.records[index].credential.UserID == owner {
			values = append(values, store.records[index].credential)
		}
	}
	return ListStoreListed{Values: values}
}

func (store *memoryStore) RevokeCredential(_ context.Context, owner core.UserID, id core.AgentCredentialID) RevokeStoreResult {
	for index := range store.records {
		if store.records[index].credential.ID == id && store.records[index].credential.UserID == owner {
			store.records[index].credential.State = StateRevoked
			return RevokeStoreRevoked{Value: store.records[index].credential}
		}
	}
	return RevokeStoreRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "agent credential was not found")}
}

func (store *memoryStore) revoke(id core.AgentCredentialID) {
	for index := range store.records {
		if store.records[index].credential.ID == id {
			store.records[index].credential.State = StateRevoked
		}
	}
}

func testLabel(t *testing.T) Label {
	t.Helper()
	accepted, matched := NewLabel("Test agent").(LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	return accepted.Value
}

func newTestUserID(t *testing.T) core.UserID {
	t.Helper()
	created, matched := core.NewUserID().(core.UserIDCreated)
	if !matched {
		t.Fatalf("user id rejected")
	}
	return created.Value
}
