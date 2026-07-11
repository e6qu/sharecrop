package orgcredbridge

import (
	"testing"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/orgcred/orgcredtest"
)

func sampleCredential(t *testing.T, expiresAt *time.Time) orgcred.Credential {
	t.Helper()
	id, matched := core.NewOrgCredentialID().(core.OrgCredentialIDCreated)
	if !matched {
		t.Fatalf("org credential id rejected")
	}
	orgID, matched := core.NewOrganizationID().(core.OrganizationIDCreated)
	if !matched {
		t.Fatalf("organization id rejected")
	}
	label, matched := agent.NewLabel("org-cred").(agent.LabelAccepted)
	if !matched {
		t.Fatalf("label rejected")
	}
	scope, matched := agent.ParseScope("org_read").(agent.ScopeAccepted)
	if !matched {
		t.Fatalf("scope rejected")
	}
	return orgcred.Credential{
		ID:             id.Value,
		OrganizationID: orgID.Value,
		Label:          label.Value,
		Scopes:         agent.NewScopeSet([]agent.Scope{scope.Value}),
		State:          agent.StateActive,
		ExpiresAt:      expiresAt,
	}
}

func assertCredentialEqual(t *testing.T, got, want orgcred.Credential) {
	t.Helper()
	if diff := orgcredtest.CredentialDiff(got, want); diff != "" {
		t.Errorf("credential mismatch: %s", diff)
	}
}

func TestCredentialRoundTrip(t *testing.T) {
	expires := time.Date(2026, 7, 12, 12, 0, 0, 0, time.UTC)
	for _, credential := range []orgcred.Credential{sampleCredential(t, &expires), sampleCredential(t, nil)} {
		restored, err := decodeCredential(encodeCredential(credential))
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertCredentialEqual(t, restored, credential)
	}
}

func TestVerifyRevokeListRoundTrip(t *testing.T) {
	credential := sampleCredential(t, nil)

	verified, err := decodeVerifyResult(encodeVerifyResult(orgcred.VerifyStoreFound{Value: credential}))
	if err != nil {
		t.Fatalf("decode verify: %v", err)
	}
	if typed, matched := verified.(orgcred.VerifyStoreFound); !matched {
		t.Fatalf("verify result = %T", verified)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	revoked, err := decodeRevokeResult(encodeRevokeResult(orgcred.RevokeStoreRevoked{Value: credential}))
	if err != nil {
		t.Fatalf("decode revoke: %v", err)
	}
	if typed, matched := revoked.(orgcred.RevokeStoreRevoked); !matched {
		t.Fatalf("revoke result = %T", revoked)
	} else {
		assertCredentialEqual(t, typed.Value, credential)
	}

	listed, err := decodeListResult(encodeListResult(orgcred.ListStoreListed{Values: []orgcred.Credential{credential}}))
	if err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if typed, matched := listed.(orgcred.ListStoreListed); !matched || len(typed.Values) != 1 {
		t.Fatalf("list result = %T", listed)
	} else {
		assertCredentialEqual(t, typed.Values[0], credential)
	}
}
