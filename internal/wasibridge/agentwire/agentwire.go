// Package agentwire holds the JSON wire codecs for the agent value types that
// both the agent and orgcred bridges share: Label, State, ScopeSet, and the
// accept/reject CreateStoreResult (orgcred's is a type alias of agent's). Their
// own Credential structs differ, so those stay in each bridge package; this is
// only the shared vocabulary.
package agentwire

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// EncodeLabel / DecodeLabel carry an agent.Label.
func EncodeLabel(label agent.Label) string { return label.String() }

func DecodeLabel(raw string) (agent.Label, error) {
	accepted, matched := agent.NewLabel(raw).(agent.LabelAccepted)
	if !matched {
		return agent.Label{}, fmt.Errorf("invalid agent label %q", raw)
	}
	return accepted.Value, nil
}

// EncodeState / DecodeState carry an agent.State.
func EncodeState(state agent.State) string { return state.String() }

func DecodeState(raw string) (agent.State, error) {
	accepted, matched := agent.ParseState(raw).(agent.StateAccepted)
	if !matched {
		return agent.State{}, fmt.Errorf("invalid agent state %q", raw)
	}
	return accepted.Value, nil
}

// EncodeScopeSet / DecodeScopeSet carry an agent.ScopeSet as a string list.
func EncodeScopeSet(scopes agent.ScopeSet) []string {
	values := scopes.Values()
	encoded := make([]string, 0, len(values))
	for index := range values {
		encoded = append(encoded, values[index].String())
	}
	return encoded
}

func DecodeScopeSet(raw []string) (agent.ScopeSet, error) {
	scopes := make([]agent.Scope, 0, len(raw))
	for _, value := range raw {
		accepted, matched := agent.ParseScope(value).(agent.ScopeAccepted)
		if !matched {
			return agent.ScopeSet{}, fmt.Errorf("invalid agent scope %q", value)
		}
		scopes = append(scopes, accepted.Value)
	}
	return agent.NewScopeSet(scopes), nil
}

// CreateResultWire is the wire form of agent.CreateStoreResult, which orgcred
// aliases - so both bridges serialize it through here.
type CreateResultWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func EncodeCreateStoreResult(result agent.CreateStoreResult) CreateResultWire {
	switch typed := result.(type) {
	case agent.CreateStoreAccepted:
		return CreateResultWire{Variant: "accepted"}
	case agent.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return CreateResultWire{Variant: "rejected", Error: &reason}
	default:
		reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, fmt.Sprintf("unknown create result %T", result)))
		return CreateResultWire{Variant: "rejected", Error: &reason}
	}
}

func DecodeCreateStoreResult(wire CreateResultWire) (agent.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return agent.CreateStoreAccepted{}, nil
	case "rejected":
		if wire.Error == nil {
			return nil, fmt.Errorf("rejected create result is missing its error")
		}
		return agent.CreateStoreRejected{Reason: domainwire.DecodeDomainError(*wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}
