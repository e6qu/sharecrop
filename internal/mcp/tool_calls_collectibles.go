package mcp

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/core"
)

type collectibleSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	State          string `json:"state"`
	TransferPolicy string `json:"transfer_policy"`
	OwnerID        string `json:"owner_id"`
	OwnerKind      string `json:"owner_kind"`
	OrganizationID string `json:"organization_id"`
	Art            string `json:"art"`
}

type collectiblesListPayload struct {
	Collectibles []collectibleSummary `json:"collectibles"`
}

type catalogEntryPayload struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	TransferPolicy string `json:"transfer_policy"`
	Art            string `json:"art"`
}

type catalogPayload struct {
	Entries []catalogEntryPayload `json:"entries"`
}

func (collectibleSummary) payloadValue() {}

func (collectiblesListPayload) payloadValue() {}

func (catalogPayload) payloadValue() {}

func collectibleToSummary(value assets.Collectible) collectibleSummary {
	return collectibleSummary{
		ID:             value.ID.String(),
		Name:           value.Name.String(),
		Kind:           value.Kind.String(),
		State:          value.State.String(),
		TransferPolicy: value.Policy.String(),
		OwnerID:        value.OwnerID,
		OwnerKind:      value.OwnerKind,
		OrganizationID: value.OrganizationID,
		Art:            value.Art,
	}
}

func (server Server) callMintCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Name           string `json:"name"`
		Kind           string `json:"kind"`
		TransferPolicy string `json:"transfer_policy"`
		Art            string `json:"art"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	nameResult := assets.NewCollectibleName(args.Name)
	name, nameMatched := nameResult.(assets.CollectibleNameAccepted)
	if !nameMatched {
		return toolProtocolError{code: codeInvalidParams, message: nameResult.(assets.CollectibleNameRejected).Reason.Description()}
	}
	kindResult := assets.ParseCollectibleKind(args.Kind)
	kind, kindMatched := kindResult.(assets.CollectibleKindAccepted)
	if !kindMatched {
		return toolProtocolError{code: codeInvalidParams, message: kindResult.(assets.CollectibleKindRejected).Reason.Description()}
	}
	policyResult := assets.ParseTransferPolicy(args.TransferPolicy)
	policy, policyMatched := policyResult.(assets.TransferPolicyAccepted)
	if !policyMatched {
		return toolProtocolError{code: codeInvalidParams, message: policyResult.(assets.TransferPolicyRejected).Reason.Description()}
	}
	result := server.services.MintCollectible(ctx, assets.CollectibleOwnerKindUser, subject.ID.String(), "", name.Value, kind.Value, policy.Value, args.Art)
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		return toolFailed{message: result.(assets.MintRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(minted.Value))
}

func (server Server) callCollectibleCatalog(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	entries := assets.Catalog()
	payload := catalogPayload{Entries: make([]catalogEntryPayload, 0, len(entries))}
	for index := range entries {
		payload.Entries = append(payload.Entries, catalogEntryPayload{
			Slug:           entries[index].Slug,
			Name:           entries[index].Name,
			Kind:           entries[index].Kind.String(),
			TransferPolicy: entries[index].Policy.String(),
			Art:            entries[index].Art,
		})
	}
	return marshalPayload(payload)
}

func (server Server) callTransferCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		CollectibleID string `json:"collectible_id"`
		RecipientID   string `json:"recipient_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return toolProtocolError{code: codeInvalidParams, message: collectibleIDResult.(core.CollectibleIDRejected).Reason.Description()}
	}
	recipientResult := core.ParseUserID(args.RecipientID)
	recipient, recipientMatched := recipientResult.(core.UserIDCreated)
	if !recipientMatched {
		return toolProtocolError{code: codeInvalidParams, message: recipientResult.(core.UserIDRejected).Reason.Description()}
	}
	result := server.services.TransferCollectible(ctx, subject.ID, recipient.Value, collectibleID.Value)
	gifted, matched := result.(assets.CollectibleGifted)
	if !matched {
		return toolFailed{message: result.(assets.GiftRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(gifted.Value))
}

func (server Server) callListCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	result := server.services.ListCollectibles(ctx, subject.ID, core.DefaultPage())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		return toolFailed{message: result.(assets.ListRejected).Reason.Description()}
	}
	summaries := make([]collectibleSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, collectibleToSummary(listed.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries})
}

func (server Server) callFundCollectibleReward(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	var args struct {
		CollectibleID string `json:"collectible_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	collectibleIDResult := core.ParseCollectibleID(args.CollectibleID)
	collectibleID, collectibleMatched := collectibleIDResult.(core.CollectibleIDCreated)
	if !collectibleMatched {
		return toolProtocolError{code: codeInvalidParams, message: collectibleIDResult.(core.CollectibleIDRejected).Reason.Description()}
	}
	result := server.services.FundCollectibleReward(ctx, subject.ID, taskID, collectibleID.Value)
	funded, matched := result.(assets.RewardFunded)
	if !matched {
		return toolFailed{message: result.(assets.FundRewardRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(funded.Value))
}

func (server Server) callRefundCollectibleReward(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	taskID, problem := parseTaskID(arguments)
	if problem != nil {
		return problem
	}
	result := server.services.RefundCollectibleReward(ctx, subject.ID, taskID)
	refunded, matched := result.(assets.RewardRefunded)
	if !matched {
		return toolFailed{message: result.(assets.RefundRewardRejected).Reason.Description()}
	}
	summaries := make([]collectibleSummary, 0, len(refunded.Values))
	for index := range refunded.Values {
		summaries = append(summaries, collectibleToSummary(refunded.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries})
}

func (server Server) callListOrganizationCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	organizationID, problem := parseOrganizationID(arguments)
	if problem != nil {
		return problem
	}
	return server.listCollectiblesByOwner(ctx, assets.CollectibleOwnerKindOrganization, organizationID.String())
}

func (server Server) callListTeamCollectibles(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	teamID, problem := parseTeamID(arguments)
	if problem != nil {
		return problem
	}
	return server.listCollectiblesByOwner(ctx, assets.CollectibleOwnerKindTeam, teamID.String())
}

func (server Server) listCollectiblesByOwner(ctx context.Context, ownerKind string, ownerID string) toolResult {
	if strings.TrimSpace(ownerID) == "" {
		return toolProtocolError{code: codeInvalidParams, message: "owner id is required"}
	}
	result := server.services.ListCollectiblesByOwner(ctx, ownerKind, ownerID, core.DefaultPage())
	listed, matched := result.(assets.CollectiblesListed)
	if !matched {
		return toolFailed{message: result.(assets.ListRejected).Reason.Description()}
	}
	summaries := make([]collectibleSummary, 0, len(listed.Values))
	for index := range listed.Values {
		summaries = append(summaries, collectibleToSummary(listed.Values[index]))
	}
	return marshalPayload(collectiblesListPayload{Collectibles: summaries})
}

func (server Server) callAwardCollectible(ctx context.Context, subject auth.UserSubject, arguments json.RawMessage) toolResult {
	var args struct {
		Slug           string `json:"slug"`
		RecipientKind  string `json:"recipient_kind"`
		RecipientID    string `json:"recipient_id"`
		OrganizationID string `json:"organization_id"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return invalidArguments()
	}
	recipientKind := strings.TrimSpace(args.RecipientKind)
	if recipientKind == "" {
		recipientKind = assets.CollectibleOwnerKindUser
	}
	if !assets.ValidCollectibleOwnerKind(recipientKind) {
		return toolProtocolError{code: codeInvalidParams, message: "recipient kind must be user, team, or organization"}
	}
	if strings.TrimSpace(args.RecipientID) == "" {
		return toolProtocolError{code: codeInvalidParams, message: "recipient id is required"}
	}
	result := server.services.AwardCollectible(ctx, strings.TrimSpace(args.Slug), recipientKind, strings.TrimSpace(args.RecipientID), strings.TrimSpace(args.OrganizationID))
	minted, matched := result.(assets.CollectibleMinted)
	if !matched {
		return toolFailed{message: result.(assets.MintRejected).Reason.Description()}
	}
	return marshalPayload(collectibleToSummary(minted.Value))
}
