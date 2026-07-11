// Package agentbridge is the WASI bridge for internal/agent's Store (agent MCP
// credentials): hand-written per-type codecs (this file) plus a generated
// dispatcher and guest client (bridge_gen.go). Shared core types are serialized
// by internal/wasibridge/corewire; the agent value types it shares with the
// orgcred bridge (Label, State, ScopeSet, CreateStoreResult) live in
// internal/wasibridge/agentwire. Only what is specific to the agent store's
// Credential and result unions is here.
//
// Like auth, the credential SecretHash is an opaque stored string, so it
// round-trips through agent.SecretHashFromString rather than by re-hashing.
package agentbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

// ---- value types specific to the agent store ----

func encodeSecretHash(hash agent.SecretHash) string { return hash.String() }

func decodeSecretHash(raw string) (agent.SecretHash, error) {
	return agent.SecretHashFromString(raw), nil
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
		Label:     agentwire.EncodeLabel(credential.Label),
		Scopes:    agentwire.EncodeScopeSet(credential.Scopes),
		State:     agentwire.EncodeState(credential.State),
		ExpiresAt: corewire.EncodeTimePtr(credential.ExpiresAt),
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
	label, err := agentwire.DecodeLabel(wire.Label)
	if err != nil {
		return agent.Credential{}, err
	}
	scopes, err := agentwire.DecodeScopeSet(wire.Scopes)
	if err != nil {
		return agent.Credential{}, err
	}
	state, err := agentwire.DecodeState(wire.State)
	if err != nil {
		return agent.Credential{}, err
	}
	expiresAt, err := corewire.DecodeTimePtr(wire.ExpiresAt)
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

// ---- result unions specific to the agent store ----
//
// CreateStoreResult is shared with orgcred, so it is serialized by agentwire;
// the verify/revoke/list results carry an agent.Credential and stay here.

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
