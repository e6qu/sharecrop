package schema

import "github.com/e6qu/sharecrop/internal/core"

type Sensitivity interface {
	sensitivity()
}

type NotSensitive struct{}

type Sensitive struct {
	Category  SensitivityCategory
	Retention RetentionPolicy
	Redaction RedactionPolicy
}

func (NotSensitive) sensitivity() {}

func (Sensitive) sensitivity() {}

type SensitivityCategory struct {
	value string
}

var (
	SensitivityPII    = SensitivityCategory{value: "pii"}
	SensitivitySecret = SensitivityCategory{value: "secret"}
)

type SensitivityCategoryResult interface {
	sensitivityCategoryResult()
}

type SensitivityCategoryAccepted struct {
	Value SensitivityCategory
}

type SensitivityCategoryRejected struct {
	Reason core.DomainError
}

func (SensitivityCategoryAccepted) sensitivityCategoryResult() {}

func (SensitivityCategoryRejected) sensitivityCategoryResult() {}

func ParseSensitivityCategory(raw string) SensitivityCategoryResult {
	switch raw {
	case SensitivityPII.value:
		return SensitivityCategoryAccepted{Value: SensitivityPII}
	case SensitivitySecret.value:
		return SensitivityCategoryAccepted{Value: SensitivitySecret}
	default:
		return SensitivityCategoryRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "sensitivity category is invalid")}
	}
}

func (category SensitivityCategory) String() string {
	return category.value
}

type RetentionPolicy struct {
	value string
}

var (
	RetentionStandard        = RetentionPolicy{value: "standard"}
	RetentionDeleteOnRequest = RetentionPolicy{value: "delete_on_request"}
)

type RetentionPolicyResult interface {
	retentionPolicyResult()
}

type RetentionPolicyAccepted struct {
	Value RetentionPolicy
}

type RetentionPolicyRejected struct {
	Reason core.DomainError
}

func (RetentionPolicyAccepted) retentionPolicyResult() {}

func (RetentionPolicyRejected) retentionPolicyResult() {}

func ParseRetentionPolicy(raw string) RetentionPolicyResult {
	switch raw {
	case RetentionStandard.value:
		return RetentionPolicyAccepted{Value: RetentionStandard}
	case RetentionDeleteOnRequest.value:
		return RetentionPolicyAccepted{Value: RetentionDeleteOnRequest}
	default:
		return RetentionPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "retention policy is invalid")}
	}
}

func (policy RetentionPolicy) String() string {
	return policy.value
}

type RedactionPolicy struct {
	value string
}

var (
	RedactionReplace = RedactionPolicy{value: "replace"}
	RedactionRemove  = RedactionPolicy{value: "remove"}
)

type RedactionPolicyResult interface {
	redactionPolicyResult()
}

type RedactionPolicyAccepted struct {
	Value RedactionPolicy
}

type RedactionPolicyRejected struct {
	Reason core.DomainError
}

func (RedactionPolicyAccepted) redactionPolicyResult() {}

func (RedactionPolicyRejected) redactionPolicyResult() {}

func ParseRedactionPolicy(raw string) RedactionPolicyResult {
	switch raw {
	case RedactionReplace.value:
		return RedactionPolicyAccepted{Value: RedactionReplace}
	case RedactionRemove.value:
		return RedactionPolicyAccepted{Value: RedactionRemove}
	default:
		return RedactionPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "redaction policy is invalid")}
	}
}

func (policy RedactionPolicy) String() string {
	return policy.value
}
