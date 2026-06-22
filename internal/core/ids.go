package core

import "github.com/e6qu/sharecrop/internal/core/id"

type UserID struct {
	value id.ID
}

type TaskID struct {
	value id.ID
}

type TaskSeriesID struct {
	value id.ID
}

type TaskCapabilityTokenID struct {
	value id.ID
}

type SubmissionID struct {
	value id.ID
}

type SubmissionReceiptTokenID struct {
	value id.ID
}

type OrganizationID struct {
	value id.ID
}

type TeamID struct {
	value id.ID
}

type OrganizationMembershipID struct {
	value id.ID
}

type AgentCredentialID struct {
	value id.ID
}

type CollectibleID struct {
	value id.ID
}

type CreditAccountID struct {
	value id.ID
}

type LedgerEntryID struct {
	value id.ID
}

type GuestID struct {
	value id.ID
}

type RefreshTokenID struct {
	value id.ID
}

type UserIDResult interface {
	userIDResult()
}

type UserIDCreated struct {
	Value UserID
}

type UserIDRejected struct {
	Reason DomainError
}

func (UserIDCreated) userIDResult() {}

func (UserIDRejected) userIDResult() {}

func NewUserID() UserIDResult {
	return userIDFromIDResult(id.New())
}

func ParseUserID(raw string) UserIDResult {
	return userIDFromIDResult(id.Parse(raw))
}

func (id UserID) String() string {
	return id.value.String()
}

func userIDFromIDResult(result id.IDResult) UserIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return UserIDCreated{Value: UserID{value: typed.Value}}
	case id.IDRejected:
		return UserIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return UserIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TaskIDResult interface {
	taskIDResult()
}

type TaskIDCreated struct {
	Value TaskID
}

type TaskIDRejected struct {
	Reason DomainError
}

func (TaskIDCreated) taskIDResult() {}

func (TaskIDRejected) taskIDResult() {}

func NewTaskID() TaskIDResult {
	return taskIDFromIDResult(id.New())
}

func ParseTaskID(raw string) TaskIDResult {
	return taskIDFromIDResult(id.Parse(raw))
}

func (id TaskID) String() string {
	return id.value.String()
}

func taskIDFromIDResult(result id.IDResult) TaskIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskIDCreated{Value: TaskID{value: typed.Value}}
	case id.IDRejected:
		return TaskIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TaskSeriesIDResult interface {
	taskSeriesIDResult()
}

type TaskSeriesIDCreated struct {
	Value TaskSeriesID
}

type TaskSeriesIDRejected struct {
	Reason DomainError
}

func (TaskSeriesIDCreated) taskSeriesIDResult() {}

func (TaskSeriesIDRejected) taskSeriesIDResult() {}

func NewTaskSeriesID() TaskSeriesIDResult {
	return taskSeriesIDFromIDResult(id.New())
}

func ParseTaskSeriesID(raw string) TaskSeriesIDResult {
	return taskSeriesIDFromIDResult(id.Parse(raw))
}

func (id TaskSeriesID) String() string {
	return id.value.String()
}

func taskSeriesIDFromIDResult(result id.IDResult) TaskSeriesIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskSeriesIDCreated{Value: TaskSeriesID{value: typed.Value}}
	case id.IDRejected:
		return TaskSeriesIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskSeriesIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TaskCapabilityTokenIDResult interface {
	taskCapabilityTokenIDResult()
}

type TaskCapabilityTokenIDCreated struct {
	Value TaskCapabilityTokenID
}

type TaskCapabilityTokenIDRejected struct {
	Reason DomainError
}

func (TaskCapabilityTokenIDCreated) taskCapabilityTokenIDResult() {}

func (TaskCapabilityTokenIDRejected) taskCapabilityTokenIDResult() {}

func NewTaskCapabilityTokenID() TaskCapabilityTokenIDResult {
	return taskCapabilityTokenIDFromIDResult(id.New())
}

func ParseTaskCapabilityTokenID(raw string) TaskCapabilityTokenIDResult {
	return taskCapabilityTokenIDFromIDResult(id.Parse(raw))
}

func (id TaskCapabilityTokenID) String() string {
	return id.value.String()
}

func taskCapabilityTokenIDFromIDResult(result id.IDResult) TaskCapabilityTokenIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskCapabilityTokenIDCreated{Value: TaskCapabilityTokenID{value: typed.Value}}
	case id.IDRejected:
		return TaskCapabilityTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskCapabilityTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type SubmissionIDResult interface {
	submissionIDResult()
}

type SubmissionIDCreated struct {
	Value SubmissionID
}

type SubmissionIDRejected struct {
	Reason DomainError
}

func (SubmissionIDCreated) submissionIDResult() {}

func (SubmissionIDRejected) submissionIDResult() {}

func NewSubmissionID() SubmissionIDResult {
	return submissionIDFromIDResult(id.New())
}

func ParseSubmissionID(raw string) SubmissionIDResult {
	return submissionIDFromIDResult(id.Parse(raw))
}

func (id SubmissionID) String() string {
	return id.value.String()
}

func submissionIDFromIDResult(result id.IDResult) SubmissionIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return SubmissionIDCreated{Value: SubmissionID{value: typed.Value}}
	case id.IDRejected:
		return SubmissionIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return SubmissionIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type SubmissionReceiptTokenIDResult interface {
	submissionReceiptTokenIDResult()
}

type SubmissionReceiptTokenIDCreated struct {
	Value SubmissionReceiptTokenID
}

type SubmissionReceiptTokenIDRejected struct {
	Reason DomainError
}

func (SubmissionReceiptTokenIDCreated) submissionReceiptTokenIDResult() {}

func (SubmissionReceiptTokenIDRejected) submissionReceiptTokenIDResult() {}

func NewSubmissionReceiptTokenID() SubmissionReceiptTokenIDResult {
	return submissionReceiptTokenIDFromIDResult(id.New())
}

func ParseSubmissionReceiptTokenID(raw string) SubmissionReceiptTokenIDResult {
	return submissionReceiptTokenIDFromIDResult(id.Parse(raw))
}

func (id SubmissionReceiptTokenID) String() string {
	return id.value.String()
}

func submissionReceiptTokenIDFromIDResult(result id.IDResult) SubmissionReceiptTokenIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return SubmissionReceiptTokenIDCreated{Value: SubmissionReceiptTokenID{value: typed.Value}}
	case id.IDRejected:
		return SubmissionReceiptTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return SubmissionReceiptTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type OrganizationIDResult interface {
	organizationIDResult()
}

type OrganizationIDCreated struct {
	Value OrganizationID
}

type OrganizationIDRejected struct {
	Reason DomainError
}

func (OrganizationIDCreated) organizationIDResult() {}

func (OrganizationIDRejected) organizationIDResult() {}

func NewOrganizationID() OrganizationIDResult {
	return organizationIDFromIDResult(id.New())
}

func ParseOrganizationID(raw string) OrganizationIDResult {
	return organizationIDFromIDResult(id.Parse(raw))
}

func (id OrganizationID) String() string {
	return id.value.String()
}

func organizationIDFromIDResult(result id.IDResult) OrganizationIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return OrganizationIDCreated{Value: OrganizationID{value: typed.Value}}
	case id.IDRejected:
		return OrganizationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return OrganizationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TeamIDResult interface {
	teamIDResult()
}

type TeamIDCreated struct {
	Value TeamID
}

type TeamIDRejected struct {
	Reason DomainError
}

func (TeamIDCreated) teamIDResult() {}

func (TeamIDRejected) teamIDResult() {}

func NewTeamID() TeamIDResult {
	return teamIDFromIDResult(id.New())
}

func ParseTeamID(raw string) TeamIDResult {
	return teamIDFromIDResult(id.Parse(raw))
}

func (id TeamID) String() string {
	return id.value.String()
}

func teamIDFromIDResult(result id.IDResult) TeamIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TeamIDCreated{Value: TeamID{value: typed.Value}}
	case id.IDRejected:
		return TeamIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TeamIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type OrganizationMembershipIDResult interface {
	organizationMembershipIDResult()
}

type OrganizationMembershipIDCreated struct {
	Value OrganizationMembershipID
}

type OrganizationMembershipIDRejected struct {
	Reason DomainError
}

func (OrganizationMembershipIDCreated) organizationMembershipIDResult() {}

func (OrganizationMembershipIDRejected) organizationMembershipIDResult() {}

func NewOrganizationMembershipID() OrganizationMembershipIDResult {
	return organizationMembershipIDFromIDResult(id.New())
}

func ParseOrganizationMembershipID(raw string) OrganizationMembershipIDResult {
	return organizationMembershipIDFromIDResult(id.Parse(raw))
}

func (id OrganizationMembershipID) String() string {
	return id.value.String()
}

func organizationMembershipIDFromIDResult(result id.IDResult) OrganizationMembershipIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return OrganizationMembershipIDCreated{Value: OrganizationMembershipID{value: typed.Value}}
	case id.IDRejected:
		return OrganizationMembershipIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return OrganizationMembershipIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type CollectibleIDResult interface {
	collectibleIDResult()
}

type CollectibleIDCreated struct {
	Value CollectibleID
}

type CollectibleIDRejected struct {
	Reason DomainError
}

func (CollectibleIDCreated) collectibleIDResult() {}

func (CollectibleIDRejected) collectibleIDResult() {}

func NewCollectibleID() CollectibleIDResult {
	return collectibleIDFromIDResult(id.New())
}

func ParseCollectibleID(raw string) CollectibleIDResult {
	return collectibleIDFromIDResult(id.Parse(raw))
}

func (id CollectibleID) String() string {
	return id.value.String()
}

func collectibleIDFromIDResult(result id.IDResult) CollectibleIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return CollectibleIDCreated{Value: CollectibleID{value: typed.Value}}
	case id.IDRejected:
		return CollectibleIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return CollectibleIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type AgentCredentialIDResult interface {
	agentCredentialIDResult()
}

type AgentCredentialIDCreated struct {
	Value AgentCredentialID
}

type AgentCredentialIDRejected struct {
	Reason DomainError
}

func (AgentCredentialIDCreated) agentCredentialIDResult() {}

func (AgentCredentialIDRejected) agentCredentialIDResult() {}

func NewAgentCredentialID() AgentCredentialIDResult {
	return agentCredentialIDFromIDResult(id.New())
}

func ParseAgentCredentialID(raw string) AgentCredentialIDResult {
	return agentCredentialIDFromIDResult(id.Parse(raw))
}

func (id AgentCredentialID) String() string {
	return id.value.String()
}

func agentCredentialIDFromIDResult(result id.IDResult) AgentCredentialIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return AgentCredentialIDCreated{Value: AgentCredentialID{value: typed.Value}}
	case id.IDRejected:
		return AgentCredentialIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return AgentCredentialIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type CreditAccountIDResult interface {
	creditAccountIDResult()
}

type CreditAccountIDCreated struct {
	Value CreditAccountID
}

type CreditAccountIDRejected struct {
	Reason DomainError
}

func (CreditAccountIDCreated) creditAccountIDResult() {}

func (CreditAccountIDRejected) creditAccountIDResult() {}

func NewCreditAccountID() CreditAccountIDResult {
	return creditAccountIDFromIDResult(id.New())
}

func ParseCreditAccountID(raw string) CreditAccountIDResult {
	return creditAccountIDFromIDResult(id.Parse(raw))
}

func (id CreditAccountID) String() string {
	return id.value.String()
}

func creditAccountIDFromIDResult(result id.IDResult) CreditAccountIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return CreditAccountIDCreated{Value: CreditAccountID{value: typed.Value}}
	case id.IDRejected:
		return CreditAccountIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return CreditAccountIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type LedgerEntryIDResult interface {
	ledgerEntryIDResult()
}

type LedgerEntryIDCreated struct {
	Value LedgerEntryID
}

type LedgerEntryIDRejected struct {
	Reason DomainError
}

func (LedgerEntryIDCreated) ledgerEntryIDResult() {}

func (LedgerEntryIDRejected) ledgerEntryIDResult() {}

func NewLedgerEntryID() LedgerEntryIDResult {
	return ledgerEntryIDFromIDResult(id.New())
}

func ParseLedgerEntryID(raw string) LedgerEntryIDResult {
	return ledgerEntryIDFromIDResult(id.Parse(raw))
}

func (id LedgerEntryID) String() string {
	return id.value.String()
}

func ledgerEntryIDFromIDResult(result id.IDResult) LedgerEntryIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return LedgerEntryIDCreated{Value: LedgerEntryID{value: typed.Value}}
	case id.IDRejected:
		return LedgerEntryIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return LedgerEntryIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type GuestIDResult interface {
	guestIDResult()
}

type GuestIDCreated struct {
	Value GuestID
}

type GuestIDRejected struct {
	Reason DomainError
}

func (GuestIDCreated) guestIDResult() {}

func (GuestIDRejected) guestIDResult() {}

func NewGuestID() GuestIDResult {
	return guestIDFromIDResult(id.New())
}

func ParseGuestID(raw string) GuestIDResult {
	return guestIDFromIDResult(id.Parse(raw))
}

func (id GuestID) String() string {
	return id.value.String()
}

func guestIDFromIDResult(result id.IDResult) GuestIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return GuestIDCreated{Value: GuestID{value: typed.Value}}
	case id.IDRejected:
		return GuestIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return GuestIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type RefreshTokenIDResult interface {
	refreshTokenIDResult()
}

type RefreshTokenIDCreated struct {
	Value RefreshTokenID
}

type RefreshTokenIDRejected struct {
	Reason DomainError
}

func (RefreshTokenIDCreated) refreshTokenIDResult() {}

func (RefreshTokenIDRejected) refreshTokenIDResult() {}

func NewRefreshTokenID() RefreshTokenIDResult {
	return refreshTokenIDFromIDResult(id.New())
}

func ParseRefreshTokenID(raw string) RefreshTokenIDResult {
	return refreshTokenIDFromIDResult(id.Parse(raw))
}

func (id RefreshTokenID) String() string {
	return id.value.String()
}

func refreshTokenIDFromIDResult(result id.IDResult) RefreshTokenIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return RefreshTokenIDCreated{Value: RefreshTokenID{value: typed.Value}}
	case id.IDRejected:
		return RefreshTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return RefreshTokenIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}
