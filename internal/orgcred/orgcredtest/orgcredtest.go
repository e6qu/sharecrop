// Package orgcredtest holds test-support helpers for orgcred.Credential, shared
// by the orgcred bridge's codec tests and the integration dual-run test. The
// fields it shares with an agent credential (label, scopes, state, expiry) are
// compared by agenttest.SharedFieldsDiff, so the two comparators don't diverge.
package orgcredtest

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/agent/agenttest"
	"github.com/e6qu/sharecrop/internal/orgcred"
)

// CredentialDiff returns a description of the first field in which got and want
// differ, or "" if they are equal.
func CredentialDiff(got, want orgcred.Credential) string {
	if got.ID != want.ID {
		return fmt.Sprintf("id: %s != %s", got.ID, want.ID)
	}
	if got.OrganizationID != want.OrganizationID {
		return fmt.Sprintf("organization id: %s != %s", got.OrganizationID, want.OrganizationID)
	}
	return agenttest.SharedFieldsDiff(got.Label, want.Label, got.Scopes, want.Scopes, got.State, want.State, got.ExpiresAt, want.ExpiresAt)
}
