// Package assetsbridge is the WASI bridge for internal/assets's Store
// (collectibles): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types (ids, page) are
// serialized by internal/wasibridge/corewire.
package assetsbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- collectible value types (string wrappers) ----

func encodeName(name assets.CollectibleName) string { return name.String() }

func decodeName(raw string) (assets.CollectibleName, error) {
	accepted, matched := assets.NewCollectibleName(raw).(assets.CollectibleNameAccepted)
	if !matched {
		return assets.CollectibleName{}, fmt.Errorf("invalid collectible name %q", raw)
	}
	return accepted.Value, nil
}

func encodeKind(kind assets.CollectibleKind) string { return kind.String() }

func decodeKind(raw string) (assets.CollectibleKind, error) {
	accepted, matched := assets.ParseCollectibleKind(raw).(assets.CollectibleKindAccepted)
	if !matched {
		return assets.CollectibleKind{}, fmt.Errorf("invalid collectible kind %q", raw)
	}
	return accepted.Value, nil
}

func encodeState(state assets.CollectibleState) string { return state.String() }

func decodeState(raw string) (assets.CollectibleState, error) {
	accepted, matched := assets.ParseCollectibleState(raw).(assets.CollectibleStateAccepted)
	if !matched {
		return assets.CollectibleState{}, fmt.Errorf("invalid collectible state %q", raw)
	}
	return accepted.Value, nil
}

func encodePolicy(policy assets.TransferPolicy) string { return policy.String() }

func decodePolicy(raw string) (assets.TransferPolicy, error) {
	accepted, matched := assets.ParseTransferPolicy(raw).(assets.TransferPolicyAccepted)
	if !matched {
		return assets.TransferPolicy{}, fmt.Errorf("invalid transfer policy %q", raw)
	}
	return accepted.Value, nil
}

// ---- assets.Collectible ----

type collectibleWire struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	Policy         string `json:"policy"`
	OwnerKind      string `json:"owner_kind"`
	OwnerID        string `json:"owner_id"`
	OrganizationID string `json:"organization_id"`
	Art            string `json:"art"`
}

func encodeCollectible(collectible assets.Collectible) collectibleWire {
	return collectibleWire{
		ID:             corewire.EncodeCollectibleID(collectible.ID),
		Name:           encodeName(collectible.Name),
		Kind:           encodeKind(collectible.Kind),
		State:          encodeState(collectible.State),
		Policy:         encodePolicy(collectible.Policy),
		OwnerKind:      collectible.OwnerKind,
		OwnerID:        collectible.OwnerID,
		OrganizationID: collectible.OrganizationID,
		Art:            collectible.Art,
	}
}

func decodeCollectible(wire collectibleWire) (assets.Collectible, error) {
	id, err := corewire.DecodeCollectibleID(wire.ID)
	if err != nil {
		return assets.Collectible{}, err
	}
	name, err := decodeName(wire.Name)
	if err != nil {
		return assets.Collectible{}, err
	}
	kind, err := decodeKind(wire.Kind)
	if err != nil {
		return assets.Collectible{}, err
	}
	state, err := decodeState(wire.State)
	if err != nil {
		return assets.Collectible{}, err
	}
	policy, err := decodePolicy(wire.Policy)
	if err != nil {
		return assets.Collectible{}, err
	}
	return assets.Collectible{
		ID:             id,
		Name:           name,
		Kind:           kind,
		State:          state,
		Policy:         policy,
		OwnerKind:      wire.OwnerKind,
		OwnerID:        wire.OwnerID,
		OrganizationID: wire.OrganizationID,
		Art:            wire.Art,
	}, nil
}

func encodeCollectibles(values []assets.Collectible) []collectibleWire {
	encoded := make([]collectibleWire, 0, len(values))
	for index := range values {
		encoded = append(encoded, encodeCollectible(values[index]))
	}
	return encoded
}

func decodeCollectibles(wires []collectibleWire) ([]assets.Collectible, error) {
	values := make([]assets.Collectible, 0, len(wires))
	for index := range wires {
		value, err := decodeCollectible(wires[index])
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

// ---- command structs ----

type awardCommandWire struct {
	OrganizationID  string `json:"organization_id"`
	CollectibleID   string `json:"collectible_id"`
	RecipientUserID string `json:"recipient_user_id"`
}

func encodeAwardCommand(command assets.AwardOrganizationCollectibleStoreCommand) awardCommandWire {
	return awardCommandWire{
		OrganizationID:  corewire.EncodeOrganizationID(command.OrganizationID),
		CollectibleID:   corewire.EncodeCollectibleID(command.CollectibleID),
		RecipientUserID: corewire.EncodeUserID(command.RecipientUserID),
	}
}

func decodeAwardCommand(wire awardCommandWire) (assets.AwardOrganizationCollectibleStoreCommand, error) {
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	recipientUserID, err := corewire.DecodeUserID(wire.RecipientUserID)
	if err != nil {
		return assets.AwardOrganizationCollectibleStoreCommand{}, err
	}
	return assets.AwardOrganizationCollectibleStoreCommand{
		OrganizationID:  organizationID,
		CollectibleID:   collectibleID,
		RecipientUserID: recipientUserID,
	}, nil
}

type giftCommandWire struct {
	FromUserID    string `json:"from_user_id"`
	ToUserID      string `json:"to_user_id"`
	CollectibleID string `json:"collectible_id"`
}

func encodeGiftCommand(command assets.GiftStoreCommand) giftCommandWire {
	return giftCommandWire{
		FromUserID:    corewire.EncodeUserID(command.FromUserID),
		ToUserID:      corewire.EncodeUserID(command.ToUserID),
		CollectibleID: corewire.EncodeCollectibleID(command.CollectibleID),
	}
}

func decodeGiftCommand(wire giftCommandWire) (assets.GiftStoreCommand, error) {
	fromUserID, err := corewire.DecodeUserID(wire.FromUserID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	toUserID, err := corewire.DecodeUserID(wire.ToUserID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.GiftStoreCommand{}, err
	}
	return assets.GiftStoreCommand{FromUserID: fromUserID, ToUserID: toUserID, CollectibleID: collectibleID}, nil
}

type fundCommandWire struct {
	FunderUserID  string `json:"funder_user_id"`
	TaskID        string `json:"task_id"`
	CollectibleID string `json:"collectible_id"`
}

func encodeFundCommand(command assets.FundRewardStoreCommand) fundCommandWire {
	return fundCommandWire{
		FunderUserID:  corewire.EncodeUserID(command.FunderUserID),
		TaskID:        corewire.EncodeTaskID(command.TaskID),
		CollectibleID: corewire.EncodeCollectibleID(command.CollectibleID),
	}
}

func decodeFundCommand(wire fundCommandWire) (assets.FundRewardStoreCommand, error) {
	funderUserID, err := corewire.DecodeUserID(wire.FunderUserID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	collectibleID, err := corewire.DecodeCollectibleID(wire.CollectibleID)
	if err != nil {
		return assets.FundRewardStoreCommand{}, err
	}
	return assets.FundRewardStoreCommand{FunderUserID: funderUserID, TaskID: taskID, CollectibleID: collectibleID}, nil
}

type refundCommandWire struct {
	RequesterUserID string `json:"requester_user_id"`
	TaskID          string `json:"task_id"`
}

func encodeRefundCommand(command assets.RefundRewardStoreCommand) refundCommandWire {
	return refundCommandWire{
		RequesterUserID: corewire.EncodeUserID(command.RequesterUserID),
		TaskID:          corewire.EncodeTaskID(command.TaskID),
	}
}

func decodeRefundCommand(wire refundCommandWire) (assets.RefundRewardStoreCommand, error) {
	requesterUserID, err := corewire.DecodeUserID(wire.RequesterUserID)
	if err != nil {
		return assets.RefundRewardStoreCommand{}, err
	}
	taskID, err := corewire.DecodeTaskID(wire.TaskID)
	if err != nil {
		return assets.RefundRewardStoreCommand{}, err
	}
	return assets.RefundRewardStoreCommand{RequesterUserID: requesterUserID, TaskID: taskID}, nil
}

// ---- result unions ----

type acceptedRejectedWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateResult(result assets.CreateStoreResult) acceptedRejectedWire {
	switch typed := result.(type) {
	case assets.CreateStoreAccepted:
		return acceptedRejectedWire{Variant: "accepted"}
	case assets.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return acceptedRejectedWire{Variant: "rejected", Error: &reason}
	default:
		return acceptedRejectedWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeCreateResult(wire acceptedRejectedWire) (assets.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return assets.CreateStoreAccepted{}, nil
	case "rejected":
		return assets.CreateStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}

// collectiblesResultWire backs the list and refund results, which each carry a
// collectible slice on success.
type collectiblesResultWire struct {
	Variant      string                  `json:"variant"`
	Collectibles []collectibleWire       `json:"collectibles,omitempty"`
	Error        *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result assets.ListStoreResult) collectiblesResultWire {
	switch typed := result.(type) {
	case assets.ListStoreListed:
		return collectiblesResultWire{Variant: "listed", Collectibles: encodeCollectibles(typed.Values)}
	case assets.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectiblesResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectiblesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeListResult(wire collectiblesResultWire) (assets.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		values, err := decodeCollectibles(wire.Collectibles)
		if err != nil {
			return nil, err
		}
		return assets.ListStoreListed{Values: values}, nil
	case "rejected":
		return assets.ListStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func encodeRefundRewardResult(result assets.RefundRewardResult) collectiblesResultWire {
	switch typed := result.(type) {
	case assets.RewardRefunded:
		return collectiblesResultWire{Variant: "refunded", Collectibles: encodeCollectibles(typed.Values)}
	case assets.RefundRewardRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectiblesResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectiblesResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeRefundRewardResult(wire collectiblesResultWire) (assets.RefundRewardResult, error) {
	switch wire.Variant {
	case "refunded":
		values, err := decodeCollectibles(wire.Collectibles)
		if err != nil {
			return nil, err
		}
		return assets.RewardRefunded{Values: values}, nil
	case "rejected":
		return assets.RefundRewardRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown refund result variant %q", wire.Variant)
	}
}

// collectibleResultWire backs the fund and gift results, which each carry a
// single collectible on success.
type collectibleResultWire struct {
	Variant     string                  `json:"variant"`
	Collectible *collectibleWire        `json:"collectible,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func encodeFundRewardResult(result assets.FundRewardResult) collectibleResultWire {
	switch typed := result.(type) {
	case assets.RewardFunded:
		collectible := encodeCollectible(typed.Value)
		return collectibleResultWire{Variant: "funded", Collectible: &collectible}
	case assets.FundRewardRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectibleResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectibleResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeFundRewardResult(wire collectibleResultWire) (assets.FundRewardResult, error) {
	switch wire.Variant {
	case "funded":
		collectible, err := decodeCollectiblePayload(wire.Collectible)
		if err != nil {
			return nil, err
		}
		return assets.RewardFunded{Value: collectible}, nil
	case "rejected":
		return assets.FundRewardRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown fund result variant %q", wire.Variant)
	}
}

func encodeGiftResult(result assets.GiftResult) collectibleResultWire {
	switch typed := result.(type) {
	case assets.CollectibleGifted:
		collectible := encodeCollectible(typed.Value)
		return collectibleResultWire{Variant: "gifted", Collectible: &collectible}
	case assets.GiftRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return collectibleResultWire{Variant: "rejected", Error: &reason}
	default:
		return collectibleResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeGiftResult(wire collectibleResultWire) (assets.GiftResult, error) {
	switch wire.Variant {
	case "gifted":
		collectible, err := decodeCollectiblePayload(wire.Collectible)
		if err != nil {
			return nil, err
		}
		return assets.CollectibleGifted{Value: collectible}, nil
	case "rejected":
		return assets.GiftRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown gift result variant %q", wire.Variant)
	}
}

type taskHeldResultWire struct {
	Variant string                  `json:"variant"`
	IDs     []string                `json:"ids,omitempty"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeTaskHeldResult(result assets.TaskHeldCollectiblesResult) taskHeldResultWire {
	switch typed := result.(type) {
	case assets.TaskHeldCollectiblesFound:
		ids := make([]string, 0, len(typed.IDs))
		for index := range typed.IDs {
			ids = append(ids, corewire.EncodeCollectibleID(typed.IDs[index]))
		}
		return taskHeldResultWire{Variant: "found", IDs: ids}
	case assets.TaskHeldCollectiblesRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return taskHeldResultWire{Variant: "rejected", Error: &reason}
	default:
		return taskHeldResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown assets result %T", result))}
	}
}

func decodeTaskHeldResult(wire taskHeldResultWire) (assets.TaskHeldCollectiblesResult, error) {
	switch wire.Variant {
	case "found":
		ids := make([]core.CollectibleID, 0, len(wire.IDs))
		for index := range wire.IDs {
			id, err := corewire.DecodeCollectibleID(wire.IDs[index])
			if err != nil {
				return nil, err
			}
			ids = append(ids, id)
		}
		return assets.TaskHeldCollectiblesFound{IDs: ids}, nil
	case "rejected":
		return assets.TaskHeldCollectiblesRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown task-held result variant %q", wire.Variant)
	}
}

func decodeCollectiblePayload(wire *collectibleWire) (assets.Collectible, error) {
	if wire == nil {
		return assets.Collectible{}, fmt.Errorf("result is missing its collectible")
	}
	return decodeCollectible(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "assets bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
