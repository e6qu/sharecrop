package assets

import "github.com/e6qu/sharecrop/internal/core"

// TransferPolicy is the typed rule governing how an asset may move. Policies are
// variants rather than booleans so new rules can be added without ambiguity.
type TransferPolicy struct {
	value string
}

var (
	TransferPolicyNonTransferableExceptPayout = TransferPolicy{value: "non_transferable_except_payout"}
	TransferPolicyTransferableBetweenUsers    = TransferPolicy{value: "transferable_between_users"}
	TransferPolicyTransferableWithinOrg       = TransferPolicy{value: "transferable_within_organization"}
	TransferPolicyIssuerControlled            = TransferPolicy{value: "issuer_controlled"}
)

type TransferPolicyResult interface {
	transferPolicyResult()
}

type TransferPolicyAccepted struct {
	Value TransferPolicy
}

type TransferPolicyRejected struct {
	Reason core.DomainError
}

func (TransferPolicyAccepted) transferPolicyResult() {}

func (TransferPolicyRejected) transferPolicyResult() {}

func ParseTransferPolicy(raw string) TransferPolicyResult {
	switch raw {
	case TransferPolicyNonTransferableExceptPayout.value:
		return TransferPolicyAccepted{Value: TransferPolicyNonTransferableExceptPayout}
	case TransferPolicyTransferableBetweenUsers.value:
		return TransferPolicyAccepted{Value: TransferPolicyTransferableBetweenUsers}
	case TransferPolicyTransferableWithinOrg.value:
		return TransferPolicyAccepted{Value: TransferPolicyTransferableWithinOrg}
	case TransferPolicyIssuerControlled.value:
		return TransferPolicyAccepted{Value: TransferPolicyIssuerControlled}
	default:
		return TransferPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "transfer policy is invalid")}
	}
}

func (policy TransferPolicy) String() string {
	return policy.value
}

// RewardCheck is the typed outcome of asking whether a policy permits awarding
// an asset as a task reward payout.
type RewardCheck interface {
	rewardCheck()
}

type RewardAllowed struct{}

type RewardDenied struct {
	Reason core.DomainError
}

func (RewardAllowed) rewardCheck() {}

func (RewardDenied) rewardCheck() {}

// AllowsTip reports whether a collectible under this policy may be voluntarily
// gifted to a worker as a review tip. A tip is a free transfer between users, so
// only the transferable policies permit it; non-transferable-except-payout is
// limited to the reward-payout movement and issuer-controlled needs issuer
// consent the platform does not model yet.
func AllowsTip(policy TransferPolicy) RewardCheck {
	switch policy {
	case TransferPolicyTransferableBetweenUsers,
		TransferPolicyTransferableWithinOrg:
		return RewardAllowed{}
	case TransferPolicyNonTransferableExceptPayout:
		return RewardDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "this collectible can only move as a reward payout, not a tip")}
	case TransferPolicyIssuerControlled:
		return RewardDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "issuer-controlled assets cannot be tipped")}
	default:
		return RewardDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer policy does not permit tipping")}
	}
}

// AllowsRewardPayout reports whether a collectible under this policy may be
// awarded to a worker when their submission is accepted. Every current policy
// permits the task-payout movement; issuer-controlled assets require explicit
// issuer consent that the platform does not model yet, so they are denied.
func AllowsRewardPayout(policy TransferPolicy) RewardCheck {
	switch policy {
	case TransferPolicyNonTransferableExceptPayout,
		TransferPolicyTransferableBetweenUsers,
		TransferPolicyTransferableWithinOrg:
		return RewardAllowed{}
	case TransferPolicyIssuerControlled:
		return RewardDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "issuer-controlled assets cannot be awarded as task rewards yet")}
	default:
		return RewardDenied{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "transfer policy does not permit reward payout")}
	}
}
