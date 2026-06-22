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
)

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
