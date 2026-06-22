package task

import "github.com/e6qu/sharecrop/internal/core"

type Task struct {
	ID             core.TaskID
	Owner          Owner
	Title          Title
	Description    Description
	Reward         RewardSpec
	Participation  ParticipationPolicy
	AssigneeScope  AssigneeScope
	ReservationTTL ReservationTTL
	State          State
	Visibility     Visibility
	Placement      SeriesPlacement
	ResponseSchema ResponseSchemaSource
	Payload        DataPayload
	CreatedBy      core.UserID
}

type Series struct {
	ID        core.TaskSeriesID
	Owner     Owner
	Title     SeriesTitle
	CreatedBy core.UserID
}

type CapabilityToken struct {
	ID     core.TaskCapabilityTokenID
	TaskID core.TaskID
	State  CapabilityTokenState
}

type Reservation struct {
	ID          core.TaskReservationID
	TaskID      core.TaskID
	Assignee    Assignee
	State       ReservationState
	RequestedBy core.UserID
}

type Owner interface {
	owner()
}

type UserOwner struct {
	UserID core.UserID
}

type TeamOwner struct {
	TeamID core.TeamID
}

type OrganizationOwner struct {
	OrganizationID core.OrganizationID
}

type OrganizationTeamOwner struct {
	OrganizationID core.OrganizationID
	TeamID         core.TeamID
}

func (UserOwner) owner() {}

func (TeamOwner) owner() {}

func (OrganizationOwner) owner() {}

func (OrganizationTeamOwner) owner() {}

type RewardSpec interface {
	rewardSpec()
}

type NoRewardSpec struct{}

type CreditRewardSpec struct {
	Amount CreditRewardAmount
}

func (NoRewardSpec) rewardSpec() {}

func (CreditRewardSpec) rewardSpec() {}

type RewardKind struct {
	value string
}

var (
	RewardKindNone   = RewardKind{value: "none"}
	RewardKindCredit = RewardKind{value: "credit"}
)

func (kind RewardKind) String() string {
	return kind.value
}

type ParticipationPolicy struct {
	value string
}

var (
	ParticipationPolicyOpen                = ParticipationPolicy{value: "open"}
	ParticipationPolicyReservationRequired = ParticipationPolicy{value: "reservation_required"}
	ParticipationPolicyApprovalRequired    = ParticipationPolicy{value: "approval_required"}
)

type ParticipationPolicyResult interface {
	participationPolicyResult()
}

type ParticipationPolicyAccepted struct {
	Value ParticipationPolicy
}

type ParticipationPolicyRejected struct {
	Reason core.DomainError
}

func (ParticipationPolicyAccepted) participationPolicyResult() {}

func (ParticipationPolicyRejected) participationPolicyResult() {}

func ParseParticipationPolicy(raw string) ParticipationPolicyResult {
	switch raw {
	case ParticipationPolicyOpen.value:
		return ParticipationPolicyAccepted{Value: ParticipationPolicyOpen}
	case ParticipationPolicyReservationRequired.value:
		return ParticipationPolicyAccepted{Value: ParticipationPolicyReservationRequired}
	case ParticipationPolicyApprovalRequired.value:
		return ParticipationPolicyAccepted{Value: ParticipationPolicyApprovalRequired}
	default:
		return ParticipationPolicyRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task participation policy is invalid")}
	}
}

func (policy ParticipationPolicy) String() string {
	return policy.value
}

type AssigneeScope struct {
	value string
}

var (
	AssigneeScopeUser             = AssigneeScope{value: "user"}
	AssigneeScopeOrganizationTeam = AssigneeScope{value: "organization_team"}
)

type AssigneeScopeResult interface {
	assigneeScopeResult()
}

type AssigneeScopeAccepted struct {
	Value AssigneeScope
}

type AssigneeScopeRejected struct {
	Reason core.DomainError
}

func (AssigneeScopeAccepted) assigneeScopeResult() {}

func (AssigneeScopeRejected) assigneeScopeResult() {}

func ParseAssigneeScope(raw string) AssigneeScopeResult {
	switch raw {
	case AssigneeScopeUser.value:
		return AssigneeScopeAccepted{Value: AssigneeScopeUser}
	case AssigneeScopeOrganizationTeam.value:
		return AssigneeScopeAccepted{Value: AssigneeScopeOrganizationTeam}
	default:
		return AssigneeScopeRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task assignee scope is invalid")}
	}
}

func (scope AssigneeScope) String() string {
	return scope.value
}

type ReservationTTL struct {
	hours int
}

type ReservationTTLResult interface {
	reservationTTLResult()
}

type ReservationTTLAccepted struct {
	Value ReservationTTL
}

type ReservationTTLRejected struct {
	Reason core.DomainError
}

func (ReservationTTLAccepted) reservationTTLResult() {}

func (ReservationTTLRejected) reservationTTLResult() {}

func DefaultReservationTTL() ReservationTTL {
	return ReservationTTL{hours: 48}
}

func NewReservationTTL(hours int) ReservationTTLResult {
	if hours < 1 {
		return ReservationTTLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "reservation expiry must be at least 1 hour")}
	}
	if hours > 720 {
		return ReservationTTLRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidArgument, "reservation expiry must be at most 720 hours")}
	}
	return ReservationTTLAccepted{Value: ReservationTTL{hours: hours}}
}

func (ttl ReservationTTL) Hours() int {
	return ttl.hours
}

type Assignee interface {
	assignee()
}

type UserAssignee struct {
	UserID core.UserID
}

type OrganizationTeamAssignee struct {
	OrganizationID core.OrganizationID
	TeamID         core.TeamID
}

func (UserAssignee) assignee() {}

func (OrganizationTeamAssignee) assignee() {}

type ReservationState struct {
	value string
}

var (
	ReservationStateRequested            = ReservationState{value: "requested"}
	ReservationStateActive               = ReservationState{value: "active"}
	ReservationStateDeclined             = ReservationState{value: "declined"}
	ReservationStateCancelledByRequester = ReservationState{value: "cancelled_by_requester"}
	ReservationStateCancelledByWorker    = ReservationState{value: "cancelled_by_worker"}
	ReservationStateExpired              = ReservationState{value: "expired"}
	ReservationStateSubmitted            = ReservationState{value: "submitted"}
)

type ReservationStateResult interface {
	reservationStateResult()
}

type ReservationStateAccepted struct {
	Value ReservationState
}

type ReservationStateRejected struct {
	Reason core.DomainError
}

func (ReservationStateAccepted) reservationStateResult() {}

func (ReservationStateRejected) reservationStateResult() {}

func ParseReservationState(raw string) ReservationStateResult {
	switch raw {
	case ReservationStateRequested.value:
		return ReservationStateAccepted{Value: ReservationStateRequested}
	case ReservationStateActive.value:
		return ReservationStateAccepted{Value: ReservationStateActive}
	case ReservationStateDeclined.value:
		return ReservationStateAccepted{Value: ReservationStateDeclined}
	case ReservationStateCancelledByRequester.value:
		return ReservationStateAccepted{Value: ReservationStateCancelledByRequester}
	case ReservationStateCancelledByWorker.value:
		return ReservationStateAccepted{Value: ReservationStateCancelledByWorker}
	case ReservationStateExpired.value:
		return ReservationStateAccepted{Value: ReservationStateExpired}
	case ReservationStateSubmitted.value:
		return ReservationStateAccepted{Value: ReservationStateSubmitted}
	default:
		return ReservationStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task reservation state is invalid")}
	}
}

func (state ReservationState) String() string {
	return state.value
}

type AvailabilityKind struct {
	value string
}

var (
	AvailabilityAvailable       = AvailabilityKind{value: "available"}
	AvailabilityReserved        = AvailabilityKind{value: "reserved"}
	AvailabilityAwaitingApproval = AvailabilityKind{value: "awaiting_approval"}
	AvailabilityClosed          = AvailabilityKind{value: "closed"}
)

func (kind AvailabilityKind) String() string {
	return kind.value
}

type ViewerAction struct {
	value string
}

var (
	ViewerActionSubmit          = ViewerAction{value: "submit"}
	ViewerActionReserve         = ViewerAction{value: "reserve"}
	ViewerActionRequestApproval = ViewerAction{value: "request_approval"}
	ViewerActionWait            = ViewerAction{value: "wait"}
	ViewerActionNone            = ViewerAction{value: "none"}
)

func (action ViewerAction) String() string {
	return action.value
}

type OwnerKind struct {
	value string
}

var (
	OwnerKindUser             = OwnerKind{value: "user"}
	OwnerKindTeam             = OwnerKind{value: "team"}
	OwnerKindOrganization     = OwnerKind{value: "organization"}
	OwnerKindOrganizationTeam = OwnerKind{value: "organization_team"}
)

func (kind OwnerKind) String() string {
	return kind.value
}

type State struct {
	value string
}

var (
	StateDraft     = State{value: "draft"}
	StateOpen      = State{value: "open"}
	StateClosed    = State{value: "closed"}
	StateCancelled = State{value: "cancelled"}
	StateExpired   = State{value: "expired"}
)

type StateResult interface {
	stateResult()
}

type StateAccepted struct {
	Value State
}

type StateRejected struct {
	Reason core.DomainError
}

func (StateAccepted) stateResult() {}

func (StateRejected) stateResult() {}

func ParseState(raw string) StateResult {
	switch raw {
	case StateDraft.value:
		return StateAccepted{Value: StateDraft}
	case StateOpen.value:
		return StateAccepted{Value: StateOpen}
	case StateClosed.value:
		return StateAccepted{Value: StateClosed}
	case StateCancelled.value:
		return StateAccepted{Value: StateCancelled}
	case StateExpired.value:
		return StateAccepted{Value: StateExpired}
	default:
		return StateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task state is invalid")}
	}
}

func (state State) String() string {
	return state.value
}

type StateTransitionResult interface {
	stateTransitionResult()
}

type StateTransitionAccepted struct {
	Value State
}

type StateTransitionRejected struct {
	Reason core.DomainError
}

func (StateTransitionAccepted) stateTransitionResult() {}

func (StateTransitionRejected) stateTransitionResult() {}

func OpenState(current State) StateTransitionResult {
	if current != StateDraft {
		return StateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "only draft tasks can be opened")}
	}
	return StateTransitionAccepted{Value: StateOpen}
}

func CancelState(current State) StateTransitionResult {
	switch current {
	case StateDraft, StateOpen:
		return StateTransitionAccepted{Value: StateCancelled}
	default:
		return StateTransitionRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidState, "task cannot be cancelled from this state")}
	}
}

type Visibility interface {
	visibility()
}

type PublicVisibility struct{}

type UserVisibility struct {
	UserID core.UserID
}

type TeamVisibility struct {
	TeamID core.TeamID
}

type OrganizationVisibility struct {
	OrganizationID core.OrganizationID
}

type OrganizationTeamVisibility struct {
	OrganizationID core.OrganizationID
	TeamID         core.TeamID
}

func (PublicVisibility) visibility() {}

func (UserVisibility) visibility() {}

func (TeamVisibility) visibility() {}

func (OrganizationVisibility) visibility() {}

func (OrganizationTeamVisibility) visibility() {}

type VisibilityKind struct {
	value string
}

var (
	VisibilityKindPublic           = VisibilityKind{value: "public"}
	VisibilityKindUser             = VisibilityKind{value: "user"}
	VisibilityKindTeam             = VisibilityKind{value: "team"}
	VisibilityKindOrganization     = VisibilityKind{value: "organization"}
	VisibilityKindOrganizationTeam = VisibilityKind{value: "organization_team"}
)

func (kind VisibilityKind) String() string {
	return kind.value
}

type SeriesPlacement interface {
	seriesPlacement()
}

type StandalonePlacement struct{}

type NewSeriesPlacement struct {
	Title    SeriesTitle
	Position SeriesPosition
}

type ExistingSeriesPlacement struct {
	SeriesID core.TaskSeriesID
	Position SeriesPosition
}

func (StandalonePlacement) seriesPlacement() {}

func (NewSeriesPlacement) seriesPlacement() {}

func (ExistingSeriesPlacement) seriesPlacement() {}

type DataPayload interface {
	dataPayload()
}

type NoDataPayload struct{}

type JSONDataPayload struct {
	Source PayloadSource
}

func (NoDataPayload) dataPayload() {}

func (JSONDataPayload) dataPayload() {}

type CapabilityTokenState struct {
	value string
}

var (
	CapabilityTokenStateActive  = CapabilityTokenState{value: "active"}
	CapabilityTokenStateRevoked = CapabilityTokenState{value: "revoked"}
)

type CapabilityTokenStateResult interface {
	capabilityTokenStateResult()
}

type CapabilityTokenStateAccepted struct {
	Value CapabilityTokenState
}

type CapabilityTokenStateRejected struct {
	Reason core.DomainError
}

func (CapabilityTokenStateAccepted) capabilityTokenStateResult() {}

func (CapabilityTokenStateRejected) capabilityTokenStateResult() {}

func ParseCapabilityTokenState(raw string) CapabilityTokenStateResult {
	switch raw {
	case CapabilityTokenStateActive.value:
		return CapabilityTokenStateAccepted{Value: CapabilityTokenStateActive}
	case CapabilityTokenStateRevoked.value:
		return CapabilityTokenStateAccepted{Value: CapabilityTokenStateRevoked}
	default:
		return CapabilityTokenStateRejected{Reason: core.NewDomainError(core.ErrorCodeInvalidEnum, "task capability token state is invalid")}
	}
}

func (state CapabilityTokenState) String() string {
	return state.value
}
