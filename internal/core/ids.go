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

type TaskReservationID struct {
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

type TaskCollectibleRewardID struct {
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

type AuditEventID struct {
	value id.ID
}

type NotificationID struct {
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

type TaskReservationIDResult interface {
	taskReservationIDResult()
}

type TaskReservationIDCreated struct {
	Value TaskReservationID
}

type TaskReservationIDRejected struct {
	Reason DomainError
}

func (TaskReservationIDCreated) taskReservationIDResult() {}

func (TaskReservationIDRejected) taskReservationIDResult() {}

func NewTaskReservationID() TaskReservationIDResult {
	return taskReservationIDFromIDResult(id.New())
}

func ParseTaskReservationID(raw string) TaskReservationIDResult {
	return taskReservationIDFromIDResult(id.Parse(raw))
}

func (id TaskReservationID) String() string {
	return id.value.String()
}

func taskReservationIDFromIDResult(result id.IDResult) TaskReservationIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskReservationIDCreated{Value: TaskReservationID{value: typed.Value}}
	case id.IDRejected:
		return TaskReservationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskReservationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
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

type TaskCollectibleRewardIDResult interface {
	taskCollectibleRewardIDResult()
}

type TaskCollectibleRewardIDCreated struct {
	Value TaskCollectibleRewardID
}

type TaskCollectibleRewardIDRejected struct {
	Reason DomainError
}

func (TaskCollectibleRewardIDCreated) taskCollectibleRewardIDResult() {}

func (TaskCollectibleRewardIDRejected) taskCollectibleRewardIDResult() {}

func NewTaskCollectibleRewardID() TaskCollectibleRewardIDResult {
	return taskCollectibleRewardIDFromIDResult(id.New())
}

func (id TaskCollectibleRewardID) String() string {
	return id.value.String()
}

func taskCollectibleRewardIDFromIDResult(result id.IDResult) TaskCollectibleRewardIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskCollectibleRewardIDCreated{Value: TaskCollectibleRewardID{value: typed.Value}}
	case id.IDRejected:
		return TaskCollectibleRewardIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskCollectibleRewardIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
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

type SeriesCommentID struct {
	value id.ID
}

type SeriesCommentIDResult interface {
	seriesCommentIDResult()
}

type SeriesCommentIDCreated struct {
	Value SeriesCommentID
}

type SeriesCommentIDRejected struct {
	Reason DomainError
}

func (SeriesCommentIDCreated) seriesCommentIDResult() {}

func (SeriesCommentIDRejected) seriesCommentIDResult() {}

func NewSeriesCommentID() SeriesCommentIDResult {
	return seriesCommentIDFromIDResult(id.New())
}

func ParseSeriesCommentID(raw string) SeriesCommentIDResult {
	return seriesCommentIDFromIDResult(id.Parse(raw))
}

func (id SeriesCommentID) String() string {
	return id.value.String()
}

func seriesCommentIDFromIDResult(result id.IDResult) SeriesCommentIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return SeriesCommentIDCreated{Value: SeriesCommentID{value: typed.Value}}
	case id.IDRejected:
		return SeriesCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return SeriesCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type TaskCommentID struct {
	value id.ID
}

type TaskCommentIDResult interface {
	taskCommentIDResult()
}

type TaskCommentIDCreated struct {
	Value TaskCommentID
}

type TaskCommentIDRejected struct {
	Reason DomainError
}

func (TaskCommentIDCreated) taskCommentIDResult() {}

func (TaskCommentIDRejected) taskCommentIDResult() {}

func NewTaskCommentID() TaskCommentIDResult {
	return taskCommentIDFromIDResult(id.New())
}

func ParseTaskCommentID(raw string) TaskCommentIDResult {
	return taskCommentIDFromIDResult(id.Parse(raw))
}

func (id TaskCommentID) String() string {
	return id.value.String()
}

func taskCommentIDFromIDResult(result id.IDResult) TaskCommentIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return TaskCommentIDCreated{Value: TaskCommentID{value: typed.Value}}
	case id.IDRejected:
		return TaskCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return TaskCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type SubmissionCommentID struct {
	value id.ID
}

type SubmissionCommentIDResult interface {
	submissionCommentIDResult()
}

type SubmissionCommentIDCreated struct {
	Value SubmissionCommentID
}

type SubmissionCommentIDRejected struct {
	Reason DomainError
}

func (SubmissionCommentIDCreated) submissionCommentIDResult() {}

func (SubmissionCommentIDRejected) submissionCommentIDResult() {}

func NewSubmissionCommentID() SubmissionCommentIDResult {
	return submissionCommentIDFromIDResult(id.New())
}

func ParseSubmissionCommentID(raw string) SubmissionCommentIDResult {
	return submissionCommentIDFromIDResult(id.Parse(raw))
}

func (id SubmissionCommentID) String() string {
	return id.value.String()
}

func submissionCommentIDFromIDResult(result id.IDResult) SubmissionCommentIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return SubmissionCommentIDCreated{Value: SubmissionCommentID{value: typed.Value}}
	case id.IDRejected:
		return SubmissionCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return SubmissionCommentIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type AuditEventIDResult interface {
	auditEventIDResult()
}

type AuditEventIDCreated struct {
	Value AuditEventID
}

type AuditEventIDRejected struct {
	Reason DomainError
}

func (AuditEventIDCreated) auditEventIDResult() {}

func (AuditEventIDRejected) auditEventIDResult() {}

func NewAuditEventID() AuditEventIDResult {
	return auditEventIDFromIDResult(id.New())
}

func ParseAuditEventID(raw string) AuditEventIDResult {
	return auditEventIDFromIDResult(id.Parse(raw))
}

func (id AuditEventID) String() string {
	return id.value.String()
}

func auditEventIDFromIDResult(result id.IDResult) AuditEventIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return AuditEventIDCreated{Value: AuditEventID{value: typed.Value}}
	case id.IDRejected:
		return AuditEventIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return AuditEventIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}

type NotificationIDResult interface {
	notificationIDResult()
}

type NotificationIDCreated struct {
	Value NotificationID
}

type NotificationIDRejected struct {
	Reason DomainError
}

func (NotificationIDCreated) notificationIDResult() {}

func (NotificationIDRejected) notificationIDResult() {}

func NewNotificationID() NotificationIDResult {
	return notificationIDFromIDResult(id.New())
}

func ParseNotificationID(raw string) NotificationIDResult {
	return notificationIDFromIDResult(id.Parse(raw))
}

func (id NotificationID) String() string {
	return id.value.String()
}

func notificationIDFromIDResult(result id.IDResult) NotificationIDResult {
	switch typed := result.(type) {
	case id.IDCreated:
		return NotificationIDCreated{Value: NotificationID{value: typed.Value}}
	case id.IDRejected:
		return NotificationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, typed.Description)}
	default:
		return NotificationIDRejected{Reason: NewDomainError(ErrorCodeInvalidID, "unknown id result")}
	}
}
