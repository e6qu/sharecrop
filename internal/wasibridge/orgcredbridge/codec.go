// Package orgcredbridge is the WASI bridge for internal/orgcred's Store
// (organization-wide credentials): hand-written per-type codecs (this file) plus
// a generated dispatcher and guest client (bridge_gen.go). It shares agent's
// value types (Label, State, ScopeSet, CreateStoreResult) via
// internal/wasibridge/agentwire and core types via corewire; only its own
// Credential and result unions are here.
package orgcredbridge

import (
	"fmt"

	"github.com/e6qu/sharecrop/internal/core"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/wasibridge/agentwire"
	"github.com/e6qu/sharecrop/internal/wasibridge/corewire"
	"github.com/e6qu/sharecrop/internal/wasibridge/domainwire"
)

func encodeSecretHash(hash orgcred.SecretHash) string { return hash.String() }

func decodeSecretHash(raw string) (orgcred.SecretHash, error) {
	return orgcred.SecretHashFromString(raw), nil
}

// ---- orgcred.Credential ----

type credentialWire struct {
	ID             string   `json:"id"`
	OrganizationID string   `json:"organization_id"`
	Label          string   `json:"label"`
	Scopes         []string `json:"scopes"`
	State          string   `json:"state"`
	ExpiresAt      string   `json:"expires_at,omitempty"`
}

func encodeCredential(credential orgcred.Credential) credentialWire {
	return credentialWire{
		ID:             corewire.EncodeOrgCredentialID(credential.ID),
		OrganizationID: corewire.EncodeOrganizationID(credential.OrganizationID),
		Label:          agentwire.EncodeLabel(credential.Label),
		Scopes:         agentwire.EncodeScopeSet(credential.Scopes),
		State:          agentwire.EncodeState(credential.State),
		ExpiresAt:      corewire.EncodeTimePtr(credential.ExpiresAt),
	}
}

func decodeCredential(wire credentialWire) (orgcred.Credential, error) {
	id, err := corewire.DecodeOrgCredentialID(wire.ID)
	if err != nil {
		return orgcred.Credential{}, err
	}
	organizationID, err := corewire.DecodeOrganizationID(wire.OrganizationID)
	if err != nil {
		return orgcred.Credential{}, err
	}
	label, err := agentwire.DecodeLabel(wire.Label)
	if err != nil {
		return orgcred.Credential{}, err
	}
	scopes, err := agentwire.DecodeScopeSet(wire.Scopes)
	if err != nil {
		return orgcred.Credential{}, err
	}
	state, err := agentwire.DecodeState(wire.State)
	if err != nil {
		return orgcred.Credential{}, err
	}
	expiresAt, err := corewire.DecodeTimePtr(wire.ExpiresAt)
	if err != nil {
		return orgcred.Credential{}, err
	}
	return orgcred.Credential{
		ID:             id,
		OrganizationID: organizationID,
		Label:          label,
		Scopes:         scopes,
		State:          state,
		ExpiresAt:      expiresAt,
	}, nil
}

// ---- result unions (CreateStoreResult is shared via agentwire) ----

type credentialResultWire struct {
	Variant    string                  `json:"variant"`
	Credential *credentialWire         `json:"credential,omitempty"`
	Error      *domainwire.DomainError `json:"error,omitempty"`
}

func encodeVerifyResult(result orgcred.VerifyStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case orgcred.VerifyStoreFound:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "found", Credential: &credential}
	case orgcred.VerifyStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown orgcred result %T", result))}
	}
}

func decodeVerifyResult(wire credentialResultWire) (orgcred.VerifyStoreResult, error) {
	switch wire.Variant {
	case "found":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return orgcred.VerifyStoreFound{Value: credential}, nil
	case "rejected":
		return orgcred.VerifyStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown verify result variant %q", wire.Variant)
	}
}

func encodeRevokeResult(result orgcred.RevokeStoreResult) credentialResultWire {
	switch typed := result.(type) {
	case orgcred.RevokeStoreRevoked:
		credential := encodeCredential(typed.Value)
		return credentialResultWire{Variant: "revoked", Credential: &credential}
	case orgcred.RevokeStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return credentialResultWire{Variant: "rejected", Error: &reason}
	default:
		return credentialResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown orgcred result %T", result))}
	}
}

func decodeRevokeResult(wire credentialResultWire) (orgcred.RevokeStoreResult, error) {
	switch wire.Variant {
	case "revoked":
		credential, err := decodeCredentialPayload(wire.Credential)
		if err != nil {
			return nil, err
		}
		return orgcred.RevokeStoreRevoked{Value: credential}, nil
	case "rejected":
		return orgcred.RevokeStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown revoke result variant %q", wire.Variant)
	}
}

type listResultWire struct {
	Variant     string                  `json:"variant"`
	Credentials []credentialWire        `json:"credentials,omitempty"`
	Error       *domainwire.DomainError `json:"error,omitempty"`
}

func encodeListResult(result orgcred.ListStoreResult) listResultWire {
	switch typed := result.(type) {
	case orgcred.ListStoreListed:
		credentials := make([]credentialWire, 0, len(typed.Values))
		for index := range typed.Values {
			credentials = append(credentials, encodeCredential(typed.Values[index]))
		}
		return listResultWire{Variant: "listed", Credentials: credentials}
	case orgcred.ListStoreRejected:
		reason := domainwire.EncodeDomainError(typed.Reason)
		return listResultWire{Variant: "rejected", Error: &reason}
	default:
		return listResultWire{Variant: "rejected", Error: rejectionError(fmt.Sprintf("unknown orgcred result %T", result))}
	}
}

func decodeListResult(wire listResultWire) (orgcred.ListStoreResult, error) {
	switch wire.Variant {
	case "listed":
		credentials := make([]orgcred.Credential, 0, len(wire.Credentials))
		for index := range wire.Credentials {
			credential, err := decodeCredential(wire.Credentials[index])
			if err != nil {
				return nil, err
			}
			credentials = append(credentials, credential)
		}
		return orgcred.ListStoreListed{Values: credentials}, nil
	case "rejected":
		return orgcred.ListStoreRejected{Reason: decodeReason(wire.Error)}, nil
	default:
		return nil, fmt.Errorf("unknown list result variant %q", wire.Variant)
	}
}

func decodeCredentialPayload(wire *credentialWire) (orgcred.Credential, error) {
	if wire == nil {
		return orgcred.Credential{}, fmt.Errorf("credential result is missing its credential")
	}
	return decodeCredential(*wire)
}

func decodeReason(wire *domainwire.DomainError) core.DomainError {
	if wire == nil {
		return core.NewDomainError(core.ErrorCodeInvalidState, "orgcred bridge: rejected result is missing its error")
	}
	return domainwire.DecodeDomainError(*wire)
}

func rejectionError(message string) *domainwire.DomainError {
	reason := domainwire.EncodeDomainError(core.NewDomainError(core.ErrorCodeInvalidState, message))
	return &reason
}
