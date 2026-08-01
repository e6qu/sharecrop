package core

type ErrorCode struct {
	value string
}

type DomainError struct {
	code        ErrorCode
	description string
}

var (
	ErrorCodeInvalidID        = ErrorCode{value: "invalid_id"}
	ErrorCodeInvalidEnum      = ErrorCode{value: "invalid_enum"}
	ErrorCodeInvalidState     = ErrorCode{value: "invalid_state"}
	ErrorCodeInvalidArgument  = ErrorCode{value: "invalid_argument"}
	ErrorCodeNotFound         = ErrorCode{value: "not_found"}
	ErrorCodePermissionDenied = ErrorCode{value: "permission_denied"}
	ErrorCodeConflict         = ErrorCode{value: "conflict"}
	ErrorCodeUnauthenticated  = ErrorCode{value: "unauthenticated"}
	ErrorCodeRateLimited      = ErrorCode{value: "rate_limited"}
	// ErrorCodeUnavailable marks a server-side failure (upstream provider,
	// storage, or session infrastructure) rather than a caller mistake. It is
	// written by HTTP handlers for 5xx responses; domain constructors do not
	// produce it.
	ErrorCodeUnavailable = ErrorCode{value: "unavailable"}
)

// AllErrorCodes lists every error code an API error response can carry, in
// wire order. The OpenAPI generator embeds this list in the shared
// ErrorResponse schema, so a code added here appears in the generated
// document without a second hand-maintained list.
func AllErrorCodes() []ErrorCode {
	return []ErrorCode{
		ErrorCodeInvalidID,
		ErrorCodeInvalidEnum,
		ErrorCodeInvalidState,
		ErrorCodeInvalidArgument,
		ErrorCodeNotFound,
		ErrorCodePermissionDenied,
		ErrorCodeConflict,
		ErrorCodeUnauthenticated,
		ErrorCodeRateLimited,
		ErrorCodeUnavailable,
	}
}

func NewDomainError(code ErrorCode, description string) DomainError {
	return DomainError{
		code:        code,
		description: description,
	}
}

func (e DomainError) Code() ErrorCode {
	return e.code
}

func (e DomainError) Description() string {
	return e.description
}

func (c ErrorCode) String() string {
	return c.value
}
