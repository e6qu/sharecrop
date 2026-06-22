package task

import "github.com/e6qu/sharecrop/internal/core"

type CreditRewardAmount struct {
	value int64
}

type CreditRewardAmountResult interface {
	creditRewardAmountResult()
}

type CreditRewardAmountAccepted struct {
	Value CreditRewardAmount
}

type CreditRewardAmountRejected struct {
	Reason core.DomainError
}

func (CreditRewardAmountAccepted) creditRewardAmountResult() {}

func (CreditRewardAmountRejected) creditRewardAmountResult() {}

func NewCreditRewardAmount(value int64) CreditRewardAmountResult {
	if value <= 0 {
		return CreditRewardAmountRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "credit reward amount must be positive")}
	}
	return CreditRewardAmountAccepted{Value: CreditRewardAmount{value: value}}
}

func (amount CreditRewardAmount) Int64() int64 {
	return amount.value
}

type CollectibleRewardCount struct {
	value int
}

type CollectibleRewardCountResult interface {
	collectibleRewardCountResult()
}

type CollectibleRewardCountAccepted struct {
	Value CollectibleRewardCount
}

type CollectibleRewardCountRejected struct {
	Reason core.DomainError
}

func (CollectibleRewardCountAccepted) collectibleRewardCountResult() {}

func (CollectibleRewardCountRejected) collectibleRewardCountResult() {}

func NewCollectibleRewardCount(value int) CollectibleRewardCountResult {
	if value <= 0 {
		return CollectibleRewardCountRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "collectible reward count must be positive")}
	}
	return CollectibleRewardCountAccepted{Value: CollectibleRewardCount{value: value}}
}

func (count CollectibleRewardCount) Int() int {
	return count.value
}
