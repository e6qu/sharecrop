package ledger

import (
	"testing"

	"github.com/e6qu/sharecrop/internal/core"
)

func testWorkCredentialID(t *testing.T) core.AgentCredentialID {
	t.Helper()
	created, matched := core.NewAgentCredentialID().(core.AgentCredentialIDCreated)
	if !matched {
		t.Fatalf("agent credential id rejected")
	}
	return created.Value
}

func mustAmount(t *testing.T, value int64) CreditAmount {
	t.Helper()
	accepted, matched := NewCreditAmount(value).(CreditAmountAccepted)
	if !matched {
		t.Fatalf("credit amount rejected: %d", value)
	}
	return accepted.Value
}

func TestSpendChargeForUserAndUncappedCredentialIsNone(t *testing.T) {
	amount := mustAmount(t, 40)
	if _, matched := spendChargeFor(SpendByUser{}, amount).(NoSpendCharge); !matched {
		t.Fatalf("user spend produced a charge")
	}
	uncapped := SpendViaWorkCredential{CredentialID: testWorkCredentialID(t), Cap: NoSpendDayCap{}}
	if _, matched := spendChargeFor(uncapped, amount).(NoSpendCharge); !matched {
		t.Fatalf("uncapped credential spend produced a charge")
	}
}

func TestSpendChargeForCappedCredentialCarriesLimitAndAmount(t *testing.T) {
	credentialID := testWorkCredentialID(t)
	origin := SpendViaWorkCredential{CredentialID: credentialID, Cap: SpendDayCapAtMost{Limit: mustAmount(t, 100)}}
	charge, matched := spendChargeFor(origin, mustAmount(t, 40)).(ChargeSpendBudget)
	if !matched {
		t.Fatalf("capped credential spend produced no charge")
	}
	if charge.CredentialID != credentialID || charge.DayLimit.Int64() != 100 || charge.Amount.Int64() != 40 {
		t.Fatalf("charge = %+v, want credential %s, limit 100, amount 40", charge, credentialID.String())
	}
}

func TestTipSpendChargeForOnlyChargesCreditTips(t *testing.T) {
	origin := SpendViaWorkCredential{CredentialID: testWorkCredentialID(t), Cap: SpendDayCapAtMost{Limit: mustAmount(t, 100)}}
	if _, matched := tipSpendChargeFor(origin, NoTipSelection{}).(NoSpendCharge); !matched {
		t.Fatalf("no-tip review produced a spend charge")
	}
	charge, matched := tipSpendChargeFor(origin, CreditTipSelection{Amount: mustAmount(t, 7)}).(ChargeSpendBudget)
	if !matched {
		t.Fatalf("credit tip produced no charge")
	}
	if charge.Amount.Int64() != 7 {
		t.Fatalf("tip charge amount = %d, want 7", charge.Amount.Int64())
	}
}

// TestDailyPeerTransferCeilingIsFiveGrants pins the velocity constant to its
// documented rationale (five signup grants per day).
func TestDailyPeerTransferCeilingIsFiveGrants(t *testing.T) {
	if DailyPeerTransferCeilingCredits != 5*SignupGrantAmount().Int64() {
		t.Fatalf("DailyPeerTransferCeilingCredits = %d, want %d", DailyPeerTransferCeilingCredits, 5*SignupGrantAmount().Int64())
	}
}
