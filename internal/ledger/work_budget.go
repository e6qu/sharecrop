package ledger

import (
	"github.com/e6qu/sharecrop/internal/core"
)

// DailyPeerTransferCeilingCredits is the per-subject velocity limit on peer
// credit sends: one source account (a user's, or an organization's) may send
// at most this many credits per UTC calendar day, whoever initiates the
// sends. 500 credits is five signup grants - generous for genuine
// person-to-person payments on top of task escrow (which is unaffected),
// while making a stolen session or a sybil ring drain value slowly enough to
// notice. Platform-admin grants (manual_adjustment) are exempt: they are an
// audited administrative action, not a peer transfer.
const DailyPeerTransferCeilingCredits int64 = 500

// SpendOrigin says how the actor of a credit spend (task funding, tips, peer
// sends) authenticated. A human session is never budget-limited; a personal
// agent credential carries the daily spend cap its owner configured, if one
// exists.
// The types live here (not in internal/agent) because internal/agent imports
// this package; internal/agent adapts a credential into these values.
type SpendOrigin interface {
	spendOrigin()
}

// SpendByUser is a signed-in person (or, over REST, an organization credential
// on paths where those are allowed - org credentials cannot pay tips or fund
// with personal credits). No spend budget applies.
type SpendByUser struct{}

// SpendViaWorkCredential is a personal agent credential. Cap is the daily
// credit spend allowance of its work policy; a credential whose policy is
// disabled, or enabled without a spend cap, spends without a daily cap (its
// reach is still limited by the credential's scopes).
type SpendViaWorkCredential struct {
	CredentialID core.AgentCredentialID
	Cap          SpendDayCap
}

func (SpendByUser) spendOrigin() {}

func (SpendViaWorkCredential) spendOrigin() {}

// SpendDayCap is the optional daily credit spend cap of a work credential.
type SpendDayCap interface {
	spendDayCap()
}

type NoSpendDayCap struct{}

type SpendDayCapAtMost struct {
	Limit CreditAmount
}

func (NoSpendDayCap) spendDayCap() {}

func (SpendDayCapAtMost) spendDayCap() {}

// SpendCharge travels inside a spend-carrying store command: the store
// consumes the amount from the credential's UTC-day spend counter atomically
// with the ledger mutation, refusing with budget_exceeded when the cap is
// exhausted.
type SpendCharge interface {
	spendCharge()
}

type NoSpendCharge struct{}

type ChargeSpendBudget struct {
	CredentialID core.AgentCredentialID
	DayLimit     CreditAmount
	Amount       CreditAmount
}

func (NoSpendCharge) spendCharge() {}

func (ChargeSpendBudget) spendCharge() {}

// spendChargeFor derives the store-side charge from a spend origin and the
// credits the operation moves.
func spendChargeFor(origin SpendOrigin, amount CreditAmount) SpendCharge {
	credential, viaCredential := origin.(SpendViaWorkCredential)
	if !viaCredential {
		return NoSpendCharge{}
	}
	capped, hasCap := credential.Cap.(SpendDayCapAtMost)
	if !hasCap {
		return NoSpendCharge{}
	}
	return ChargeSpendBudget{CredentialID: credential.CredentialID, DayLimit: capped.Limit, Amount: amount}
}

// tipSpendChargeFor derives the spend charge of a review's tip selection: only
// a credit tip moves budgeted credits.
func tipSpendChargeFor(origin SpendOrigin, selection TipSelection) SpendCharge {
	tip, hasTip := selection.(CreditTipSelection)
	if !hasTip {
		return NoSpendCharge{}
	}
	return spendChargeFor(origin, tip.Amount)
}
