// Package agentbridge is the WASI bridge for internal/agent's Store (agent MCP
// credentials): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types (ids, page,
// time) are serialized by internal/wasibridge/corewire.
//
// Like auth, the credential SecretHash is an opaque stored string, so it
// round-trips through agent.SecretHashFromString rather than by re-hashing.
package agentbridge

import (
	"fmt"
	"time"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- value types ----

func encodeSecretHash(hash agent.SecretHash) string { return hash.String() }

func decodeSecretHash(raw string) (agent.SecretHash, error) {
	return agent.SecretHashFromString(raw), nil
}

func encodeLabel(label agent.Label) string { return label.String() }

func decodeLabel(raw string) (agent.Label, error) {
	accepted, matched := agent.NewLabel(raw).(agent.LabelAccepted)
	if !matched {
		return agent.Label{}, fmt.Errorf("invalid agent label %q", raw)
	}
	return accepted.Value, nil
}

func encodeState(state agent.State) string { return state.String() }

func decodeState(raw string) (agent.State, error) {
	accepted, matched := agent.ParseState(raw).(agent.StateAccepted)
	if !matched {
		return agent.State{}, fmt.Errorf("invalid agent state %q", raw)
	}
	return accepted.Value, nil
}

func encodeScopeSet(scopes agent.ScopeSet) []string {
	values := scopes.Values()
	encoded := make([]string, 0, len(values))
	for index := range values {
		encoded = append(encoded, values[index].String())
	}
	return encoded
}

func decodeScopeSet(raw []string) (agent.ScopeSet, error) {
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

// Nullable pointers cross the wire as a string that is empty when nil.

func encodeTimePtr(value *time.Time) string {
	if value == nil {
		return ""
	}
	return corewire.EncodeTime(*value)
}

func decodeTimePtr(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := corewire.DecodeTime(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func encodeTaskIDPtr(value *core.TaskID) string {
	if value == nil {
		return ""
	}
	return corewire.EncodeTaskID(*value)
}

func decodeTaskIDPtr(raw string) (*core.TaskID, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := corewire.DecodeTaskID(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

// ---- agent.Credential ----

type credentialWire struct {
	ID        string   `json:"id"`
	UserID    string   `json:"user_id"`
	Label     string   `json:"label"`
	Scopes    []string `json:"scopes"`
	State     string   `json:"state"`
	ExpiresAt string   `json:"expires_at,omitempty"`
	TaskID    string   `json:"task_id,omitempty"`
}

func encodeCredential(credential agent.Credential) credentialWire {
	return credentialWire{
		ID:        corewire.EncodeAgentCredentialID(credential.ID),
		UserID:    corewire.EncodeUserID(credential.UserID),
		Label:     encodeLabel(credential.Label),
		Scopes:    encodeScopeSet(credential.Scopes),
		State:     encodeState(credential.State),
		ExpiresAt: encodeTimePtr(credential.ExpiresAt),
		TaskID:    encodeTaskIDPtr(credential.TaskID),
	}
}

func decodeCredential(wire credentialWire) (agent.Credential, error) {
	id, err := corewire.DecodeAgentCredentialID(wire.ID)
	if err != nil {
		return agent.Credential{}, err
	}
	userID, err := corewire.DecodeUserID(wire.UserID)
	if err != nil {
		return agent.Credential{}, err
	}
	label, err := decodeLabel(wire.Label)
	if err != nil {
		return agent.Credential{}, err
	}
	scopes, err := decodeScopeSet(wire.Scopes)
	if err != nil {
		return agent.Credential{}, err
	}
	state, err := decodeState(wire.State)
	if err != nil {
		return agent.Credential{}, err
	}
	expiresAt, err := decodeTimePtr(wire.ExpiresAt)
	if err != nil {
		return agent.Credential{}, err
	}
	taskID, err := decodeTaskIDPtr(wire.TaskID)
	if err != nil {
		return agent.Credential{}, err
	}
	return agent.Credential{
		ID:        id,
		UserID:    userID,
		Label:     label,
		Scopes:    scopes,
		State:     state,
		ExpiresAt: expiresAt,
		TaskID:    taskID,
	}, nil
}

// ---- result unions ----

type createResultWire struct {
	Variant string                  `json:"variant"`
	Error   *domainwire.DomainError `json:"error,omitempty"`
}

func encodeCreateResult(result agent.CreateStoreResult) createResultWire {
	switch typed := result.(type) {
	case agent.CreateStoreAccepted:
		return createResultWire{Variant: "accepted"}
	case agent.CreateStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return createResultWire{Variant: "rejected", Error: &reason}
	default:
		return createResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeCreateResult(wire createResultWire) (agent.CreateStoreResult, error) {
	switch wire.Variant {
	case "accepted":
		return agent.CreateStoreAccepted{}, nil
	case "rejected":
		return agent.CreateStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown create result variant %q", wire.Variant)
	}
}

// credentialResultWire backs the verify and revoke unions, which each carry a
// single credential on success.
type credentialResultWire struct {
	Variant    string                  `json:"variant"`
	Credential *credentialWire         `json:"credential,omitempty"`
	Error      *domainwire.DomainError `json:"error,omitempty"`
}

func encodeVerifyResult(result agent.VerifyStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case agent.VerifyStoreFound:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "found", Credential: &credential}
	case agent.VerifyStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeVerifyResult(wire credentialResultWire) (agent.VerifyStoreResult, error) {
	switch wire.Variant {
	case "found":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return agent.VerifyStoreFound{Value: credential}, nil
	case "rejected":
		return agent.VerifyStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown verify result variant %q", wire.Variant)
	}
}

func encodeRevokeResult(result agent.RevokeStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case agent.RevokeStoreRevoked:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "revoked", Credential: &credential}
	case agent.RevokeStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeRevokeResult(wire credentialResultWire) (agent.RevokeStoreResult, error) {
	switch wire.Variant {
	case "revoked":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return agent.RevokeStoreRevoked{Value: credential}, nil
	case "rejected":
		return agent.RevokeStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown revoke result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant     string                  `json:"variant"`
	Credentials []credentialWire        `json:"credentials,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result agent.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case agent.ListStoreListed:
		credentials := make([]credentialWire, 0, len(typed.Values))
		for index := range typed.Values {
			credentials = append(credentials, encodeCredential(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Credentials: credentials}
	case agent.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown agent result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (agent.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		credentials := make([]agent.Credential, 0, len(wire.Credentials))
		for index := range wire.Credentials {
			credential, err := decodeCredential(wire.Credentials[index])
			if err != nil {
				return nil, err
			}
			credentials = append(credentials, credential)
		}
		return agent.ListStoreListed{Values: credentials}, nil
	case "rejected":
		return agent.ListStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func decodeCredentialPayload(wire *credentialWire) (agent.Credential, error) {
	if wire == nil {
		return agent.Credential{}, fmt.Errorf("credential result is missing its credential")
	}
	return decodeCredential(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "agent bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
