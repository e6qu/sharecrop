// Package domainwire holds JSON wire encodings for the core domain types that
// recur across every store bridge - starting with core.DomainError, whose
// ErrorCode must survive the serialization boundary as its canonical value
// rather than a stringly-typed approximation. Keeping these in one place means
// each store's bridge serializes shared types identically.
package domainwire

import "github.com/e6qu/sharecrop/internal/core"

// DomainError is the wire form of core.DomainError: the code as its string form
// plus the human description.
type DomainError struct {
	Code        string `json:"code"`
	Description string `json:"description"`
}

// errorCodesByString reverses ErrorCode.String so a serialized code can be
// rebuilt as the canonical core.ErrorCode value.
var errorCodesByString = func() map[string]core.ErrorCode {
	codes := []core.ErrorCode{
		core.ErrorCodeInvalidID,
		core.ErrorCodeInvalidEnum,
		core.ErrorCodeInvalidState,
		core.ErrorCodeInvalidArgument,
		core.ErrorCodeNotFound,
		core.ErrorCodePermissionDenied,
		core.ErrorCodeConflict,
	}
	byString := make(map[string]core.ErrorCode, len(codes))
	for _, code := range codes {
		byString[code.String()] = code
	}
	return byString
}()

// EncodeDomainError serializes a DomainError.
func EncodeDomainError(reason core.DomainError) DomainError {
	return DomainError{Code: reason.Code().String(), Description: reason.Description()}
}

// DecodeDomainError rebuilds a DomainError. An unrecognized code degrades to
// invalid_state carrying a descriptive message rather than silently losing it.
func DecodeDomainError(wire DomainError) core.DomainError {
	code, matched := errorCodesByString[wire.Code]
	if !matched {
		return core.NewDomainError(core.ErrorCodeInvalidState, "bridge: unknown error code "+wire.Code)
	}
	return core.NewDomainError(code, wire.Description)
}
